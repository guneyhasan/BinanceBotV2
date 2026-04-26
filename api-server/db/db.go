package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api-server/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

func (s *Store) GetConfig(ctx context.Context) (*models.Config, error) {
	var c models.Config
	var updatedAt interface{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, trade_amount_usd, leverage, margin_mode, commission_rate, updated_at FROM config LIMIT 1`,
	).Scan(&c.ID, &c.TradeAmountUSD, &c.Leverage, &c.MarginMode, &c.CommissionRate, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.UpdatedAt = fmt.Sprintf("%v", updatedAt)
	return &c, nil
}

func (s *Store) UpdateConfig(ctx context.Context, u models.ConfigUpdate) error {
	sets := []string{}
	args := []interface{}{}
	idx := 1

	if u.TradeAmountUSD != nil {
		sets = append(sets, fmt.Sprintf("trade_amount_usd=$%d", idx))
		args = append(args, *u.TradeAmountUSD)
		idx++
	}
	if u.Leverage != nil {
		sets = append(sets, fmt.Sprintf("leverage=$%d", idx))
		args = append(args, *u.Leverage)
		idx++
	}
	if u.MarginMode != nil {
		sets = append(sets, fmt.Sprintf("margin_mode=$%d", idx))
		args = append(args, *u.MarginMode)
		idx++
	}
	if u.CommissionRate != nil {
		sets = append(sets, fmt.Sprintf("commission_rate=$%d", idx))
		args = append(args, *u.CommissionRate)
		idx++
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at=NOW()")

	query := "UPDATE config SET " + strings.Join(sets, ", ") + " WHERE id=1"
	_, err := s.Pool.Exec(ctx, query, args...)
	return err
}

func (s *Store) GetTrades(ctx context.Context, coin, signalType string, active *bool) ([]models.Trade, error) {
	query := "SELECT id, coin, signal_type, side, account_type, quantity, entry_price, leverage, commission, is_active, binance_order_id, created_at FROM trades WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if coin != "" {
		query += fmt.Sprintf(" AND coin=$%d", idx)
		args = append(args, coin)
		idx++
	}
	if signalType != "" {
		query += fmt.Sprintf(" AND signal_type=$%d", idx)
		args = append(args, signalType)
		idx++
	}
	if active != nil {
		query += fmt.Sprintf(" AND is_active=$%d", idx)
		args = append(args, *active)
		idx++
	}
	query += " ORDER BY created_at DESC LIMIT 500"

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []models.Trade
	for rows.Next() {
		var t models.Trade
		var createdAt interface{}
		err := rows.Scan(&t.ID, &t.Coin, &t.SignalType, &t.Side, &t.AccountType, &t.Quantity, &t.EntryPrice, &t.Leverage, &t.Commission, &t.IsActive, &t.BinanceOrderID, &createdAt)
		if err != nil {
			return nil, err
		}
		t.CreatedAt = fmt.Sprintf("%v", createdAt)
		trades = append(trades, t)
	}
	return trades, nil
}

func (s *Store) GetPnLRecords(ctx context.Context, coin, side string) ([]models.PnLRecord, error) {
	query := `SELECT id, coin, closed_trade_id, closing_trade_id, closed_signal_type, closing_signal_type,
		entry_price, exit_price, quantity, leverage, gross_pnl, open_commission, close_commission, total_commission, net_pnl, side, closed_at
		FROM pnl_records WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if coin != "" {
		query += fmt.Sprintf(" AND coin=$%d", idx)
		args = append(args, coin)
		idx++
	}
	if side != "" {
		query += fmt.Sprintf(" AND side=$%d", idx)
		args = append(args, side)
		idx++
	}
	query += " ORDER BY closed_at DESC LIMIT 1000"

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.PnLRecord
	for rows.Next() {
		var r models.PnLRecord
		var closedAt interface{}
		err := rows.Scan(&r.ID, &r.Coin, &r.ClosedTradeID, &r.ClosingTradeID, &r.ClosedSignalType, &r.ClosingSignalType,
			&r.EntryPrice, &r.ExitPrice, &r.Quantity, &r.Leverage, &r.GrossPnL, &r.OpenCommission, &r.CloseCommission, &r.TotalCommission, &r.NetPnL, &r.Side, &closedAt)
		if err != nil {
			return nil, err
		}
		r.ClosedAt = fmt.Sprintf("%v", closedAt)
		records = append(records, r)
	}
	return records, nil
}

func (s *Store) GetPnLSeries(ctx context.Context, coin, side string) ([]models.PnLSeries, error) {
	records, err := s.GetPnLRecordsChronological(ctx, coin, side)
	if err != nil {
		return nil, err
	}

	var series []models.PnLSeries
	var current *models.PnLSeries
	seriesIdx := 0

	for _, r := range records {
		sigNum := getSignalNumber(r.ClosedSignalType)

		if current == nil || sigNum <= getLastSignalNumber(current) {
			if current != nil {
				series = append(series, *current)
			}
			seriesIdx++
			current = &models.PnLSeries{
				SeriesIndex: seriesIdx,
				Coin:        r.Coin,
				Side:        r.Side,
			}
		}

		current.Sequence = append(current.Sequence, r.ClosedSignalType)
		current.TotalGross += r.GrossPnL
		current.TotalComm += r.TotalCommission
		current.TotalNet += r.NetPnL
		current.Records = append(current.Records, r)
	}
	if current != nil {
		series = append(series, *current)
	}

	return series, nil
}

func (s *Store) GetPnLRecordsChronological(ctx context.Context, coin, side string) ([]models.PnLRecord, error) {
	query := `SELECT id, coin, closed_trade_id, closing_trade_id, closed_signal_type, closing_signal_type,
		entry_price, exit_price, quantity, leverage, gross_pnl, open_commission, close_commission, total_commission, net_pnl, side, closed_at
		FROM pnl_records WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if coin != "" {
		query += fmt.Sprintf(" AND coin=$%d", idx)
		args = append(args, coin)
		idx++
	}
	if side != "" {
		query += fmt.Sprintf(" AND side=$%d", idx)
		args = append(args, side)
		idx++
	}
	query += " ORDER BY closed_at ASC"

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.PnLRecord
	for rows.Next() {
		var r models.PnLRecord
		var closedAt interface{}
		err := rows.Scan(&r.ID, &r.Coin, &r.ClosedTradeID, &r.ClosingTradeID, &r.ClosedSignalType, &r.ClosingSignalType,
			&r.EntryPrice, &r.ExitPrice, &r.Quantity, &r.Leverage, &r.GrossPnL, &r.OpenCommission, &r.CloseCommission, &r.TotalCommission, &r.NetPnL, &r.Side, &closedAt)
		if err != nil {
			return nil, err
		}
		r.ClosedAt = fmt.Sprintf("%v", closedAt)
		records = append(records, r)
	}
	return records, nil
}

func getSignalNumber(signal string) int {
	if len(signal) == 0 {
		return 0
	}
	n := int(signal[len(signal)-1] - '0')
	return n
}

func getLastSignalNumber(s *models.PnLSeries) int {
	if len(s.Sequence) == 0 {
		return 0
	}
	return getSignalNumber(s.Sequence[len(s.Sequence)-1])
}

func (s *Store) GetPnLSummary(ctx context.Context, coin string) ([]models.PnLSummary, error) {
	query := `SELECT
		COALESCE(coin,'ALL') as coin,
		COALESCE(side,'ALL') as side,
		COUNT(*) as trade_count,
		COALESCE(SUM(gross_pnl), 0) as total_gross,
		COALESCE(SUM(total_commission), 0) as total_comm,
		COALESCE(SUM(net_pnl), 0) as total_net,
		COALESCE(SUM(CASE WHEN net_pnl > 0 THEN 1 ELSE 0 END), 0) as wins,
		COALESCE(SUM(CASE WHEN net_pnl <= 0 THEN 1 ELSE 0 END), 0) as losses
		FROM pnl_records`
	args := []interface{}{}
	idx := 1

	if coin != "" {
		query += fmt.Sprintf(" WHERE coin=$%d", idx)
		args = append(args, coin)
		idx++
	}
	query += " GROUP BY ROLLUP(coin, side) ORDER BY coin NULLS LAST, side NULLS LAST"

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.PnLSummary
	for rows.Next() {
		var sm models.PnLSummary
		err := rows.Scan(&sm.Coin, &sm.Side, &sm.TradeCount, &sm.TotalGrossPnL, &sm.TotalCommission, &sm.TotalNetPnL, &sm.WinCount, &sm.LossCount)
		if err != nil {
			return nil, err
		}
		if sm.TradeCount > 0 {
			sm.WinRate = float64(sm.WinCount) / float64(sm.TradeCount) * 100
		}
		summaries = append(summaries, sm)
	}
	return summaries, nil
}

func (s *Store) GetPnL24hAccountSummaries(ctx context.Context, from, to time.Time) ([]models.PnL24hAccountSummary, error) {
	rows, err := s.Pool.Query(ctx, `SELECT
		t.account_type,
		COUNT(*) as trade_count,
		COALESCE(SUM(p.gross_pnl), 0) as realized_gross,
		COALESCE(SUM(p.total_commission), 0) as realized_comm,
		COALESCE(SUM(p.net_pnl), 0) as realized_net,
		COALESCE(SUM(CASE WHEN p.net_pnl > 0 THEN 1 ELSE 0 END), 0) as wins,
		COALESCE(SUM(CASE WHEN p.net_pnl <= 0 THEN 1 ELSE 0 END), 0) as losses
		FROM pnl_records p
		JOIN trades t ON t.id = p.closed_trade_id
		WHERE p.closed_at >= $1 AND p.closed_at <= $2
		GROUP BY t.account_type
		ORDER BY t.account_type`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.PnL24hAccountSummary
	for rows.Next() {
		var sm models.PnL24hAccountSummary
		err := rows.Scan(&sm.AccountType, &sm.TradeCount, &sm.RealizedGrossPnL, &sm.RealizedComm, &sm.RealizedNetPnL, &sm.WinCount, &sm.LossCount)
		if err != nil {
			return nil, err
		}
		if sm.TradeCount > 0 {
			sm.WinRate = float64(sm.WinCount) / float64(sm.TradeCount) * 100
		}
		sm.TotalNetPnL = sm.RealizedNetPnL + sm.UnrealizedNetPnL
		summaries = append(summaries, sm)
	}
	return summaries, nil
}

func (s *Store) GetPnL24hCoinSummaries(ctx context.Context, from, to time.Time) ([]models.PnL24hCoinSummary, error) {
	rows, err := s.Pool.Query(ctx, `SELECT
		p.coin,
		COUNT(*) as trade_count,
		COALESCE(SUM(p.gross_pnl), 0) as realized_gross,
		COALESCE(SUM(p.total_commission), 0) as realized_comm,
		COALESCE(SUM(p.net_pnl), 0) as realized_net,
		COALESCE(SUM(CASE WHEN t.account_type = 'AL_ACCOUNT' THEN p.net_pnl ELSE 0 END), 0) as al_net,
		COALESCE(SUM(CASE WHEN t.account_type = 'SAT_ACCOUNT' THEN p.net_pnl ELSE 0 END), 0) as sat_net
		FROM pnl_records p
		JOIN trades t ON t.id = p.closed_trade_id
		WHERE p.closed_at >= $1 AND p.closed_at <= $2
		GROUP BY p.coin
		ORDER BY realized_net DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.PnL24hCoinSummary
	for rows.Next() {
		var sm models.PnL24hCoinSummary
		err := rows.Scan(&sm.Coin, &sm.TradeCount, &sm.RealizedGrossPnL, &sm.RealizedComm, &sm.RealizedNetPnL, &sm.ALNetPnL, &sm.SATNetPnL)
		if err != nil {
			return nil, err
		}
		sm.TotalNetPnL = sm.RealizedNetPnL + sm.UnrealizedNetPnL
		summaries = append(summaries, sm)
	}
	return summaries, nil
}

func (s *Store) GetPnL24hChartPoints(ctx context.Context, from, to time.Time) ([]models.PnL24hChartPoint, error) {
	rows, err := s.Pool.Query(ctx, `SELECT
		date_trunc('hour', p.closed_at) as bucket,
		COALESCE(SUM(p.net_pnl), 0) as realized_net,
		COALESCE(SUM(CASE WHEN t.account_type = 'AL_ACCOUNT' THEN p.net_pnl ELSE 0 END), 0) as al_realized_net,
		COALESCE(SUM(CASE WHEN t.account_type = 'SAT_ACCOUNT' THEN p.net_pnl ELSE 0 END), 0) as sat_realized_net
		FROM pnl_records p
		JOIN trades t ON t.id = p.closed_trade_id
		WHERE p.closed_at >= $1 AND p.closed_at <= $2
		GROUP BY bucket
		ORDER BY bucket ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byHour := map[time.Time]models.PnL24hChartPoint{}
	for rows.Next() {
		var bucket time.Time
		var point models.PnL24hChartPoint
		if err := rows.Scan(&bucket, &point.RealizedNetPnL, &point.ALRealizedNetPnL, &point.SATRealizedNetPnL); err != nil {
			return nil, err
		}
		byHour[bucket.UTC().Truncate(time.Hour)] = point
	}

	start := from.UTC().Truncate(time.Hour)
	end := to.UTC().Truncate(time.Hour)
	points := []models.PnL24hChartPoint{}
	cumulative := 0.0
	alCumulative := 0.0
	satCumulative := 0.0
	for t := start; !t.After(end); t = t.Add(time.Hour) {
		point := byHour[t]
		cumulative += point.RealizedNetPnL
		alCumulative += point.ALRealizedNetPnL
		satCumulative += point.SATRealizedNetPnL
		points = append(points, models.PnL24hChartPoint{
			Time:                     t.Format(time.RFC3339),
			RealizedNetPnL:           point.RealizedNetPnL,
			ALRealizedNetPnL:         point.ALRealizedNetPnL,
			SATRealizedNetPnL:        point.SATRealizedNetPnL,
			CumulativeRealizedPnL:    cumulative,
			ALCumulativeRealizedPnL:  alCumulative,
			SATCumulativeRealizedPnL: satCumulative,
		})
	}
	return points, nil
}

func (s *Store) GetWebhooks(ctx context.Context, coin, signal, status string) ([]models.WebhookLog, error) {
	query := `SELECT id, request_id, coin, signal_type, raw_body, status, trade_id, error_message, retry_count, received_at, completed_at
		FROM webhook_logs WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if coin != "" {
		query += fmt.Sprintf(" AND coin=$%d", idx)
		args = append(args, coin)
		idx++
	}
	if signal != "" {
		query += fmt.Sprintf(" AND signal_type=$%d", idx)
		args = append(args, signal)
		idx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status=$%d", idx)
		args = append(args, status)
		idx++
	}
	query += " ORDER BY received_at DESC LIMIT 500"

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []models.WebhookLog
	for rows.Next() {
		var w models.WebhookLog
		var rawBody []byte
		var receivedAt, completedAt interface{}
		err := rows.Scan(&w.ID, &w.RequestID, &w.Coin, &w.SignalType, &rawBody, &w.Status, &w.TradeID, &w.ErrorMessage, &w.RetryCount, &receivedAt, &completedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(rawBody, &w.RawBody)
		w.ReceivedAt = fmt.Sprintf("%v", receivedAt)
		if completedAt != nil {
			s := fmt.Sprintf("%v", completedAt)
			w.CompletedAt = &s
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, nil
}

func (s *Store) GetWebhookDetail(ctx context.Context, id int) (*models.WebhookLog, error) {
	var w models.WebhookLog
	var rawBody []byte
	var receivedAt, completedAt interface{}

	err := s.Pool.QueryRow(ctx,
		`SELECT id, request_id, coin, signal_type, raw_body, status, trade_id, error_message, retry_count, received_at, completed_at
		FROM webhook_logs WHERE id=$1`, id,
	).Scan(&w.ID, &w.RequestID, &w.Coin, &w.SignalType, &rawBody, &w.Status, &w.TradeID, &w.ErrorMessage, &w.RetryCount, &receivedAt, &completedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(rawBody, &w.RawBody)
	w.ReceivedAt = fmt.Sprintf("%v", receivedAt)
	if completedAt != nil {
		s := fmt.Sprintf("%v", completedAt)
		w.CompletedAt = &s
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT id, webhook_id, attempt_number, step, status, account_type, binance_request, binance_response, error_category, error_message, duration_ms, created_at
		FROM execution_logs WHERE webhook_id=$1 ORDER BY id ASC`, id,
	)
	if err != nil {
		return &w, nil
	}
	defer rows.Close()

	for rows.Next() {
		var e models.ExecutionLog
		var binReq, binResp []byte
		var createdAt interface{}
		err := rows.Scan(&e.ID, &e.WebhookID, &e.AttemptNumber, &e.Step, &e.Status, &e.AccountType, &binReq, &binResp, &e.ErrorCategory, &e.ErrorMessage, &e.DurationMs, &createdAt)
		if err != nil {
			continue
		}
		json.Unmarshal(binReq, &e.BinanceRequest)
		json.Unmarshal(binResp, &e.BinanceResponse)
		e.CreatedAt = fmt.Sprintf("%v", createdAt)
		w.Executions = append(w.Executions, e)
	}
	return &w, nil
}

func (s *Store) GetSystemStats(ctx context.Context) (*models.SystemStats, error) {
	var stats models.SystemStats

	err := s.Pool.QueryRow(ctx, `SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN status='COMPLETED' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status='RETRY_SUCCESS' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status='FAILED' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status IN ('RECEIVED','PROCESSING') THEN 1 ELSE 0 END), 0),
		COALESCE(AVG(CASE WHEN retry_count > 0 THEN retry_count END), 0)
		FROM webhook_logs`,
	).Scan(&stats.TotalWebhooks, &stats.CompletedCount, &stats.RetrySuccessCount, &stats.FailedCount, &stats.ProcessingCount, &stats.AvgRetryCount)
	if err != nil {
		return nil, err
	}

	total := stats.TotalWebhooks
	if total > 0 {
		successTotal := stats.CompletedCount + stats.RetrySuccessCount
		stats.SuccessRate = float64(successTotal) / float64(total) * 100
		retryTotal := stats.RetrySuccessCount + stats.FailedCount
		stats.RetryRate = float64(retryTotal) / float64(total) * 100
		stats.FailRate = float64(stats.FailedCount) / float64(total) * 100
	}

	rows, err := s.Pool.Query(ctx,
		`SELECT COALESCE(error_category,'UNKNOWN'), COUNT(*) FROM execution_logs WHERE status='FAILED' GROUP BY error_category ORDER BY COUNT(*) DESC`,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ec models.ErrorCategory
			rows.Scan(&ec.Category, &ec.Count)
			stats.ErrorBreakdown = append(stats.ErrorBreakdown, ec)
		}
	}

	failedWebhooks, _ := s.GetWebhooks(ctx, "", "", "FAILED")
	stats.FailedDetails = failedWebhooks

	return &stats, nil
}

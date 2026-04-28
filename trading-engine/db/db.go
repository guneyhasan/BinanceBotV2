package db

import (
	"context"
	"fmt"
	"time"

	"trading-engine/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) GetConfig(ctx context.Context) (*models.Config, error) {
	var cfg models.Config
	err := s.pool.QueryRow(ctx,
		`SELECT id, trade_amount_usd, leverage, margin_mode, commission_rate FROM config LIMIT 1`,
	).Scan(&cfg.ID, &cfg.TradeAmountUSD, &cfg.Leverage, &cfg.MarginMode, &cfg.CommissionRate)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	return &cfg, nil
}

func (s *Store) FindActiveOppositeTrade(ctx context.Context, coin string, signalTypes []string) (*models.Trade, error) {
	var t models.Trade
	var orderID *int64
	err := s.pool.QueryRow(ctx,
		`SELECT id, coin, signal_type, side, account_type, quantity, entry_price, leverage, commission, is_active, binance_order_id, created_at
		 FROM trades WHERE coin=$1 AND signal_type = ANY($2) AND is_active=true
		 ORDER BY created_at DESC LIMIT 1`,
		coin, signalTypes,
	).Scan(&t.ID, &t.Coin, &t.SignalType, &t.Side, &t.AccountType, &t.Quantity, &t.EntryPrice, &t.Leverage, &t.Commission, &t.IsActive, &orderID, &t.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active trade: %w", err)
	}
	t.BinanceOrderID = orderID
	return &t, nil
}

func (s *Store) DeactivateTrade(ctx context.Context, tradeID int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE trades SET is_active=false, updated_at=NOW() WHERE id=$1`,
		tradeID,
	)
	return err
}

func (s *Store) InsertTrade(ctx context.Context, t *models.Trade) (int, error) {
	var id int
	err := s.pool.QueryRow(ctx,
		`INSERT INTO trades (coin, signal_type, side, account_type, quantity, entry_price, leverage, commission, is_active, binance_order_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		t.Coin, t.SignalType, t.Side, t.AccountType, t.Quantity, t.EntryPrice, t.Leverage, t.Commission, t.IsActive, t.BinanceOrderID,
	).Scan(&id)
	return id, err
}

func (s *Store) InsertPnLRecord(ctx context.Context,
	coin string,
	closedTradeID, closingTradeID int,
	closedSignal, closingSignal string,
	entryPrice, exitPrice, quantity decimal.Decimal,
	leverage int,
	grossPnL, openComm, closeComm, totalComm, netPnL decimal.Decimal,
	side string,
) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO pnl_records (coin, closed_trade_id, closing_trade_id, closed_signal_type, closing_signal_type,
		 entry_price, exit_price, quantity, leverage, gross_pnl, open_commission, close_commission, total_commission, net_pnl, side, closed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())`,
		coin, closedTradeID, closingTradeID, closedSignal, closingSignal,
		entryPrice, exitPrice, quantity, leverage, grossPnL, openComm, closeComm, totalComm, netPnL, side,
	)
	return err
}

func (s *Store) GetWebhookIDByRequestID(ctx context.Context, requestID string) (int, error) {
	var id int
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM webhook_logs WHERE request_id=$1`, requestID,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateWebhookStatus(ctx context.Context, requestID, status string, tradeID *int, errMsg string) error {
	var completedAt *time.Time
	if status == "COMPLETED" || status == "FAILED" || status == "RETRY_SUCCESS" {
		now := time.Now().UTC()
		completedAt = &now
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE webhook_logs SET status=$2, trade_id=$3, error_message=$4, completed_at=$5, updated_at=NOW() WHERE request_id=$1`,
		requestID, status, tradeID, errMsg, completedAt,
	)
	return err
}

func (s *Store) IncrementWebhookRetry(ctx context.Context, requestID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE webhook_logs SET retry_count=retry_count+1, updated_at=NOW() WHERE request_id=$1`,
		requestID,
	)
	return err
}

func (s *Store) InsertExecutionLog(ctx context.Context,
	webhookID, attempt int,
	step, status, accountType string,
	binReq, binResp []byte,
	errCategory, errMsg string,
	durationMs int,
) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO execution_logs (webhook_id, attempt_number, step, status, account_type, binance_request, binance_response, error_category, error_message, duration_ms)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		webhookID, attempt, step, status, accountType, binReq, binResp, errCategory, errMsg, durationMs,
	)
	return err
}

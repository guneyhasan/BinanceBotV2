package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"trading-engine/binance"
	"trading-engine/db"
	"trading-engine/models"

	"github.com/shopspring/decimal"
)

type Engine struct {
	store     *db.Store
	alClient  *binance.Client
	satClient *binance.Client
}

type TradeResult struct {
	Success        bool   `json:"success"`
	Signal         string `json:"signal"`
	Coin           string `json:"coin"`
	Side           string `json:"side"`
	Quantity       string `json:"quantity"`
	Price          string `json:"price"`
	ClosedTrade    string `json:"closed_trade,omitempty"`
	NetPnL         string `json:"net_pnl,omitempty"`
	Error          string `json:"error,omitempty"`
	RetryAttempt   int    `json:"retry_attempt"`
	RequestID      string `json:"request_id"`
}

func New(store *db.Store, alClient, satClient *binance.Client) *Engine {
	return &Engine{store: store, alClient: alClient, satClient: satClient}
}

func (e *Engine) ProcessSignal(ctx context.Context, msg models.QueueMessage, attempt int) (*TradeResult, error) {
	result := &TradeResult{
		Signal:       msg.Signal,
		Coin:         msg.Ticker,
		RetryAttempt: attempt,
		RequestID:    msg.RequestID,
	}

	webhookID, _ := e.store.GetWebhookIDByRequestID(ctx, msg.RequestID)
	if webhookID > 0 {
		e.store.UpdateWebhookStatus(ctx, msg.RequestID, "PROCESSING", nil, "")
	}

	mapping := models.GetSignalMapping(msg.Signal)
	if mapping.AccountType == "" {
		err := fmt.Errorf("invalid signal: %s", msg.Signal)
		result.Error = err.Error()
		e.logExecution(ctx, webhookID, attempt, "VALIDATION", "FAILED", "", nil, nil, "VALIDATION", err.Error())
		return result, err
	}

	cfg, err := e.store.GetConfig(ctx)
	if err != nil {
		result.Error = "config load failed: " + err.Error()
		e.logExecution(ctx, webhookID, attempt, "PRICE_FETCH", "FAILED", "", nil, nil, "DATABASE", err.Error())
		return result, err
	}

	openClient := e.getClient(mapping.AccountType)
	closeClient := e.getClient(mapping.OppositeAccount)

	// Set leverage and margin type on both accounts
	for _, cl := range []*binance.Client{openClient, closeClient} {
		if err := cl.SetLeverage(msg.Ticker, cfg.Leverage); err != nil {
			log.Printf("set leverage warning: %v", err)
		}
		if err := cl.SetMarginType(msg.Ticker, cfg.MarginMode); err != nil {
			log.Printf("set margin type warning: %v", err)
		}
	}

	// Step 1: Get current price
	start := time.Now()
	currentPrice, err := openClient.GetPrice(msg.Ticker)
	dur := int(time.Since(start).Milliseconds())
	if err != nil {
		result.Error = "price fetch failed: " + err.Error()
		e.logExecution(ctx, webhookID, attempt, "PRICE_FETCH", "FAILED", "", nil, nil, "BINANCE_API", err.Error())
		return result, err
	}
	e.logExecution(ctx, webhookID, attempt, "PRICE_FETCH", "SUCCESS", "", nil,
		jsonBytes(map[string]string{"price": currentPrice.String()}), "", "", dur)

	// Step 2: Check for opposite active trade and close if found
	oppTrade, err := e.store.FindActiveOppositeTrade(ctx, msg.Ticker, mapping.OppositeSignals)
	if err != nil {
		result.Error = "find opposite trade failed: " + err.Error()
		e.logExecution(ctx, webhookID, attempt, "CLOSE_POSITION", "FAILED", mapping.OppositeAccount, nil, nil, "DATABASE", err.Error())
		return result, err
	}

	var closingTradeID int
	if oppTrade != nil {
		start = time.Now()
		closeSide := mapping.CloseSide
		roundedQty, err := closeClient.RoundQty(msg.Ticker, oppTrade.Quantity)
		if err != nil {
			result.Error = "round qty failed: " + err.Error()
			return result, err
		}

		closeOrder, reqJSON, respJSON, err := closeClient.PlaceMarketOrder(msg.Ticker, closeSide, roundedQty, true)
		dur = int(time.Since(start).Milliseconds())
		if err != nil {
			result.Error = "close position failed: " + err.Error()
			e.logExecution(ctx, webhookID, attempt, "CLOSE_POSITION", "FAILED", mapping.OppositeAccount, reqJSON, respJSON, "BINANCE_API", err.Error(), dur)
			return result, err
		}
		e.logExecution(ctx, webhookID, attempt, "CLOSE_POSITION", "SUCCESS", mapping.OppositeAccount, reqJSON, respJSON, "", "", dur)

		if err := e.store.DeactivateTrade(ctx, oppTrade.ID); err != nil {
			result.Error = "deactivate trade failed: " + err.Error()
			e.logExecution(ctx, webhookID, attempt, "DB_WRITE", "FAILED", "", nil, nil, "DATABASE", err.Error())
			return result, err
		}

		// Calculate PnL
		exitPrice := currentPrice
		if closeOrder != nil && closeOrder.AvgPrice != "" {
			if p, err := decimal.NewFromString(closeOrder.AvgPrice); err == nil && p.IsPositive() {
				exitPrice = p
			}
		}

		lev := decimal.NewFromInt(int64(oppTrade.Leverage))
		var grossPnL decimal.Decimal
		if oppTrade.Side == "LONG" {
			grossPnL = exitPrice.Sub(oppTrade.EntryPrice).Mul(oppTrade.Quantity).Mul(lev)
		} else {
			grossPnL = oppTrade.EntryPrice.Sub(exitPrice).Mul(oppTrade.Quantity).Mul(lev)
		}
		openComm := oppTrade.Quantity.Mul(oppTrade.EntryPrice).Mul(cfg.CommissionRate).Mul(lev)
		closeComm := oppTrade.Quantity.Mul(exitPrice).Mul(cfg.CommissionRate).Mul(lev)
		totalComm := openComm.Add(closeComm)
		netPnL := grossPnL.Sub(totalComm)

		// Record close trade in trades table
		closeTrade := &models.Trade{
			Coin:        msg.Ticker,
			SignalType:  msg.Signal,
			Side:        closeSide,
			AccountType: mapping.OppositeAccount,
			Quantity:    roundedQty,
			EntryPrice:  exitPrice,
			Leverage:    cfg.Leverage,
			Commission:  closeComm,
		}
		if closeOrder != nil {
			oid := closeOrder.OrderID
			closeTrade.BinanceOrderID = &oid
		}
		closingTradeID, err = e.store.InsertTrade(ctx, closeTrade)
		if err != nil {
			log.Printf("insert close trade record failed: %v", err)
		}

		if err := e.store.InsertPnLRecord(ctx, msg.Ticker,
			oppTrade.ID, closingTradeID,
			oppTrade.SignalType, msg.Signal,
			oppTrade.EntryPrice, exitPrice, oppTrade.Quantity,
			oppTrade.Leverage,
			grossPnL, openComm, closeComm, totalComm, netPnL,
			oppTrade.Side,
		); err != nil {
			log.Printf("insert pnl record failed: %v", err)
			e.logExecution(ctx, webhookID, attempt, "PNL_CALC", "FAILED", "", nil, nil, "DATABASE", err.Error())
		} else {
			e.logExecution(ctx, webhookID, attempt, "PNL_CALC", "SUCCESS", "", nil,
				jsonBytes(map[string]string{"gross_pnl": grossPnL.String(), "net_pnl": netPnL.String(), "commission": totalComm.String()}), "", "")
		}

		result.ClosedTrade = fmt.Sprintf("%s (qty=%s)", oppTrade.SignalType, oppTrade.Quantity.String())
		result.NetPnL = netPnL.StringFixed(4)
	}

	// Step 3: Open new position
	newQty := cfg.TradeAmountUSD.Div(currentPrice)
	roundedNewQty, err := openClient.RoundQty(msg.Ticker, newQty)
	if err != nil {
		result.Error = "round new qty failed: " + err.Error()
		return result, err
	}

	var openSide string
	if strings.HasPrefix(msg.Signal, "AL") {
		openSide = "BUY"
	} else {
		openSide = "SELL"
	}

	start = time.Now()
	openOrder, reqJSON, respJSON, err := openClient.PlaceMarketOrder(msg.Ticker, openSide, roundedNewQty, false)
	dur = int(time.Since(start).Milliseconds())
	if err != nil {
		result.Error = "open position failed: " + err.Error()
		e.logExecution(ctx, webhookID, attempt, "OPEN_POSITION", "FAILED", mapping.AccountType, reqJSON, respJSON, "BINANCE_API", err.Error(), dur)
		return result, err
	}
	e.logExecution(ctx, webhookID, attempt, "OPEN_POSITION", "SUCCESS", mapping.AccountType, reqJSON, respJSON, "", "", dur)

	entryPrice := currentPrice
	if openOrder != nil && openOrder.AvgPrice != "" {
		if p, err := decimal.NewFromString(openOrder.AvgPrice); err == nil && p.IsPositive() {
			entryPrice = p
		}
	}

	openComm := roundedNewQty.Mul(entryPrice).Mul(cfg.CommissionRate).Mul(decimal.NewFromInt(int64(cfg.Leverage)))

	newTrade := &models.Trade{
		Coin:        msg.Ticker,
		SignalType:  msg.Signal,
		Side:        mapping.Side,
		AccountType: mapping.AccountType,
		Quantity:    roundedNewQty,
		EntryPrice:  entryPrice,
		Leverage:    cfg.Leverage,
		Commission:  openComm,
	}
	if openOrder != nil {
		oid := openOrder.OrderID
		newTrade.BinanceOrderID = &oid
	}

	newTradeID, err := e.store.InsertTrade(ctx, newTrade)
	if err != nil {
		result.Error = "insert trade failed: " + err.Error()
		e.logExecution(ctx, webhookID, attempt, "DB_WRITE", "FAILED", "", nil, nil, "DATABASE", err.Error())
		return result, err
	}
	e.logExecution(ctx, webhookID, attempt, "DB_WRITE", "SUCCESS", "", nil, nil, "", "")

	// Update webhook status
	status := "COMPLETED"
	if attempt > 1 {
		status = "RETRY_SUCCESS"
	}
	e.store.UpdateWebhookStatus(ctx, msg.RequestID, status, &newTradeID, "")

	result.Success = true
	result.Side = mapping.Side
	result.Quantity = roundedNewQty.String()
	result.Price = entryPrice.String()

	return result, nil
}

func (e *Engine) getClient(accountType string) *binance.Client {
	if accountType == "AL_ACCOUNT" {
		return e.alClient
	}
	return e.satClient
}

func (e *Engine) logExecution(ctx context.Context, webhookID, attempt int, step, status, accountType string, reqJSON, respJSON []byte, errCat, errMsg string, durMs ...int) {
	if webhookID == 0 {
		return
	}
	d := 0
	if len(durMs) > 0 {
		d = durMs[0]
	}
	if err := e.store.InsertExecutionLog(ctx, webhookID, attempt, step, status, accountType, reqJSON, respJSON, errCat, errMsg, d); err != nil {
		log.Printf("insert execution_log failed: %v", err)
	}
}

func (e *Engine) ProcessSignalRetryLog(ctx context.Context, requestID string) {
	e.store.IncrementWebhookRetry(ctx, requestID)
}

func (e *Engine) ProcessSignalFail(ctx context.Context, requestID, errMsg string) {
	e.store.UpdateWebhookStatus(ctx, requestID, "FAILED", nil, errMsg)
}

func jsonBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

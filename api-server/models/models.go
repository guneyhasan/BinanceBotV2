package models

import "time"

type Config struct {
	ID             int     `json:"id"`
	TradeAmountUSD float64 `json:"trade_amount_usd"`
	Leverage       int     `json:"leverage"`
	MarginMode     string  `json:"margin_mode"`
	CommissionRate float64 `json:"commission_rate"`
	UpdatedAt      string  `json:"updated_at"`
}

type ConfigUpdate struct {
	TradeAmountUSD *float64 `json:"trade_amount_usd"`
	Leverage       *int     `json:"leverage"`
	MarginMode     *string  `json:"margin_mode"`
	CommissionRate *float64 `json:"commission_rate"`
}

type Trade struct {
	ID             int     `json:"id"`
	Coin           string  `json:"coin"`
	SignalType     string  `json:"signal_type"`
	Side           string  `json:"side"`
	AccountType    string  `json:"account_type"`
	Quantity       float64 `json:"quantity"`
	EntryPrice     float64 `json:"entry_price"`
	Leverage       int     `json:"leverage"`
	Commission     float64 `json:"commission"`
	IsActive       bool    `json:"is_active"`
	BinanceOrderID *int64  `json:"binance_order_id"`
	CreatedAt      string  `json:"created_at"`
}

type PnLRecord struct {
	ID                int     `json:"id"`
	Coin              string  `json:"coin"`
	ClosedTradeID     int     `json:"closed_trade_id"`
	ClosingTradeID    int     `json:"closing_trade_id"`
	ClosedSignalType  string  `json:"closed_signal_type"`
	ClosingSignalType string  `json:"closing_signal_type"`
	EntryPrice        float64 `json:"entry_price"`
	ExitPrice         float64 `json:"exit_price"`
	Quantity          float64 `json:"quantity"`
	Leverage          int     `json:"leverage"`
	GrossPnL          float64 `json:"gross_pnl"`
	OpenCommission    float64 `json:"open_commission"`
	CloseCommission   float64 `json:"close_commission"`
	TotalCommission   float64 `json:"total_commission"`
	NetPnL            float64 `json:"net_pnl"`
	Side              string  `json:"side"`
	ClosedAt          string  `json:"closed_at"`
}

type WebhookLog struct {
	ID           int            `json:"id"`
	RequestID    string         `json:"request_id"`
	Coin         string         `json:"coin"`
	SignalType   string         `json:"signal_type"`
	RawBody      interface{}    `json:"raw_body"`
	Status       string         `json:"status"`
	TradeID      *int           `json:"trade_id"`
	ErrorMessage *string        `json:"error_message"`
	RetryCount   int            `json:"retry_count"`
	ReceivedAt   string         `json:"received_at"`
	CompletedAt  *string        `json:"completed_at"`
	Executions   []ExecutionLog `json:"executions,omitempty"`
}

type ExecutionLog struct {
	ID              int         `json:"id"`
	WebhookID       int         `json:"webhook_id"`
	AttemptNumber   int         `json:"attempt_number"`
	Step            string      `json:"step"`
	Status          string      `json:"status"`
	AccountType     *string     `json:"account_type"`
	BinanceRequest  interface{} `json:"binance_request"`
	BinanceResponse interface{} `json:"binance_response"`
	ErrorCategory   *string     `json:"error_category"`
	ErrorMessage    *string     `json:"error_message"`
	DurationMs      *int        `json:"duration_ms"`
	CreatedAt       string      `json:"created_at"`
}

type PnLSeries struct {
	SeriesIndex int         `json:"series_index"`
	Coin        string      `json:"coin"`
	Side        string      `json:"side"`
	Sequence    []string    `json:"sequence"`
	TotalGross  float64     `json:"total_gross_pnl"`
	TotalComm   float64     `json:"total_commission"`
	TotalNet    float64     `json:"total_net_pnl"`
	Records     []PnLRecord `json:"records"`
}

type PnLSummary struct {
	Coin            string  `json:"coin,omitempty"`
	Side            string  `json:"side,omitempty"`
	TradeCount      int     `json:"trade_count"`
	TotalGrossPnL   float64 `json:"total_gross_pnl"`
	TotalCommission float64 `json:"total_commission"`
	TotalNetPnL     float64 `json:"total_net_pnl"`
	WinCount        int     `json:"win_count"`
	LossCount       int     `json:"loss_count"`
	WinRate         float64 `json:"win_rate"`
}

type UnrealizedPnLItem struct {
	TradeID         int     `json:"trade_id"`
	Coin            string  `json:"coin"`
	SignalType      string  `json:"signal_type"`
	Side            string  `json:"side"`
	AccountType     string  `json:"account_type"`
	Quantity        float64 `json:"quantity"`
	EntryPrice      float64 `json:"entry_price"`
	CurrentPrice    float64 `json:"current_price"`
	Leverage        int     `json:"leverage"`
	GrossPnL        float64 `json:"gross_pnl"`
	OpenCommission  float64 `json:"open_commission"`
	CloseCommission float64 `json:"close_commission"`
	TotalCommission float64 `json:"total_commission"`
	NetPnL          float64 `json:"net_pnl"`
}

type UnrealizedPnLResponse struct {
	Items           []UnrealizedPnLItem `json:"items"`
	TotalGrossPnL   float64             `json:"total_gross_pnl"`
	TotalCommission float64             `json:"total_commission"`
	TotalNetPnL     float64             `json:"total_net_pnl"`
	UpdatedAt       string              `json:"updated_at"`
}

type PnL24hAccountSummary struct {
	AccountType      string  `json:"account_type"`
	Label            string  `json:"label"`
	TradeCount       int     `json:"trade_count"`
	RealizedGrossPnL float64 `json:"realized_gross_pnl"`
	RealizedComm     float64 `json:"realized_commission"`
	RealizedNetPnL   float64 `json:"realized_net_pnl"`
	UnrealizedNetPnL float64 `json:"unrealized_net_pnl"`
	TotalNetPnL      float64 `json:"total_net_pnl"`
	WinCount         int     `json:"win_count"`
	LossCount        int     `json:"loss_count"`
	WinRate          float64 `json:"win_rate"`
}

type PnL24hCoinSummary struct {
	Coin             string  `json:"coin"`
	TradeCount       int     `json:"trade_count"`
	RealizedGrossPnL float64 `json:"realized_gross_pnl"`
	RealizedComm     float64 `json:"realized_commission"`
	RealizedNetPnL   float64 `json:"realized_net_pnl"`
	UnrealizedNetPnL float64 `json:"unrealized_net_pnl"`
	TotalNetPnL      float64 `json:"total_net_pnl"`
	ALNetPnL         float64 `json:"al_net_pnl"`
	SATNetPnL        float64 `json:"sat_net_pnl"`
}

type PnL24hChartPoint struct {
	Time                     string  `json:"time"`
	RealizedNetPnL           float64 `json:"realized_net_pnl"`
	ALRealizedNetPnL         float64 `json:"al_realized_net_pnl"`
	SATRealizedNetPnL        float64 `json:"sat_realized_net_pnl"`
	CumulativeRealizedPnL    float64 `json:"cumulative_realized_pnl"`
	ALCumulativeRealizedPnL  float64 `json:"al_cumulative_realized_pnl"`
	SATCumulativeRealizedPnL float64 `json:"sat_cumulative_realized_pnl"`
}

type AccountBalance struct {
	AccountType        string  `json:"account_type"`
	Label              string  `json:"label"`
	Asset              string  `json:"asset"`
	WalletBalance      float64 `json:"wallet_balance"`
	AvailableBalance   float64 `json:"available_balance"`
	CrossWalletBalance float64 `json:"cross_wallet_balance"`
	UnrealizedPnL      float64 `json:"unrealized_pnl"`
	Error              string  `json:"error,omitempty"`
}

type BalanceChartPoint struct {
	Time       string  `json:"time"`
	ALBalance  float64 `json:"al_balance"`
	SATBalance float64 `json:"sat_balance"`
}

type PnL24hResponse struct {
	From                  string                 `json:"from"`
	To                    string                 `json:"to"`
	UpdatedAt             string                 `json:"updated_at"`
	Accounts              []PnL24hAccountSummary `json:"accounts"`
	Coins                 []PnL24hCoinSummary    `json:"coins"`
	Chart                 []PnL24hChartPoint     `json:"chart"`
	Balances              []AccountBalance       `json:"balances"`
	BalanceChart          []BalanceChartPoint    `json:"balance_chart"`
	TotalRealizedGross    float64                `json:"total_realized_gross_pnl"`
	TotalRealizedComm     float64                `json:"total_realized_commission"`
	TotalRealizedNetPnL   float64                `json:"total_realized_net_pnl"`
	TotalUnrealizedNetPnL float64                `json:"total_unrealized_net_pnl"`
	TotalNetPnL           float64                `json:"total_net_pnl"`
}

type SystemStats struct {
	TotalWebhooks     int             `json:"total_webhooks"`
	CompletedCount    int             `json:"completed_count"`
	RetrySuccessCount int             `json:"retry_success_count"`
	FailedCount       int             `json:"failed_count"`
	ProcessingCount   int             `json:"processing_count"`
	SuccessRate       float64         `json:"success_rate"`
	RetryRate         float64         `json:"retry_rate"`
	FailRate          float64         `json:"fail_rate"`
	AvgRetryCount     float64         `json:"avg_retry_count"`
	ErrorBreakdown    []ErrorCategory `json:"error_breakdown"`
	FailedDetails     []WebhookLog    `json:"failed_details"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type ErrorCategory struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

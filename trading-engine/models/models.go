package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Config struct {
	ID             int             `json:"id"`
	TradeAmountUSD decimal.Decimal `json:"trade_amount_usd"`
	Leverage       int             `json:"leverage"`
	MarginMode     string          `json:"margin_mode"`
	CommissionRate decimal.Decimal `json:"commission_rate"`
}

type Trade struct {
	ID             int             `json:"id"`
	Coin           string          `json:"coin"`
	SignalType     string          `json:"signal_type"`
	Side           string          `json:"side"`
	AccountType    string          `json:"account_type"`
	Quantity       decimal.Decimal `json:"quantity"`
	EntryPrice     decimal.Decimal `json:"entry_price"`
	Leverage       int             `json:"leverage"`
	Commission     decimal.Decimal `json:"commission"`
	IsActive       bool            `json:"is_active"`
	BinanceOrderID *int64          `json:"binance_order_id"`
	CreatedAt      time.Time       `json:"created_at"`
}

type QueueMessage struct {
	RequestID  string `json:"request_id"`
	Signal     string `json:"signal"`
	Ticker     string `json:"ticker"`
	ReceivedAt string `json:"received_at"`
}

type SignalMapping struct {
	OppositeSignals []string
	Side            string // LONG or SHORT
	AccountType     string // AL_ACCOUNT or SAT_ACCOUNT
	OppositeAccount string
	CloseSide       string // side to send to close opposite position
}

func GetSignalMapping(signal string) SignalMapping {
	switch signal {
	case "AL1":
		return SignalMapping{
			OppositeSignals: []string{"SAT3", "SAT2", "SAT1"},
			Side:            "LONG",
			AccountType:     "AL_ACCOUNT",
			OppositeAccount: "SAT_ACCOUNT",
			CloseSide:       "BUY",
		}
	case "AL2":
		return SignalMapping{
			OppositeSignals: []string{"SAT3", "SAT2", "SAT1"},
			Side:            "LONG",
			AccountType:     "AL_ACCOUNT",
			OppositeAccount: "SAT_ACCOUNT",
			CloseSide:       "BUY",
		}
	case "AL3":
		return SignalMapping{
			OppositeSignals: []string{"SAT3", "SAT2", "SAT1"},
			Side:            "LONG",
			AccountType:     "AL_ACCOUNT",
			OppositeAccount: "SAT_ACCOUNT",
			CloseSide:       "BUY",
		}
	case "SAT1":
		return SignalMapping{
			OppositeSignals: []string{"AL3", "AL2", "AL1"},
			Side:            "SHORT",
			AccountType:     "SAT_ACCOUNT",
			OppositeAccount: "AL_ACCOUNT",
			CloseSide:       "SELL",
		}
	case "SAT2":
		return SignalMapping{
			OppositeSignals: []string{"AL3", "AL2", "AL1"},
			Side:            "SHORT",
			AccountType:     "SAT_ACCOUNT",
			OppositeAccount: "AL_ACCOUNT",
			CloseSide:       "SELL",
		}
	case "SAT3":
		return SignalMapping{
			OppositeSignals: []string{"AL3", "AL2", "AL1"},
			Side:            "SHORT",
			AccountType:     "SAT_ACCOUNT",
			OppositeAccount: "AL_ACCOUNT",
			CloseSide:       "SELL",
		}
	default:
		return SignalMapping{}
	}
}

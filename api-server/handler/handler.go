package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"api-server/db"
	"api-server/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store              *db.Store
	telegramToken      string
	telegramSignalChat string
	telegramTradeChat  string
	binanceBaseURL     string
	httpClient         *http.Client
}

func New(store *db.Store) *Handler {
	return &Handler{
		store:              store,
		telegramToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		telegramSignalChat: os.Getenv("TELEGRAM_SIGNAL_CHAT_ID"),
		telegramTradeChat:  os.Getenv("TELEGRAM_TRADE_CHAT_ID"),
		binanceBaseURL:     strings.TrimRight(envOrDefault("BINANCE_BASE_URL", "https://testnet.binancefuture.com"), "/"),
		httpClient:         &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-server", "time": time.Now().UTC()})
}

func (h *Handler) GetConfig(c *gin.Context) {
	cfg, err := h.store.GetConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *Handler) UpdateConfig(c *gin.Context) {
	var u models.ConfigUpdate
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if err := h.store.UpdateConfig(c.Request.Context(), u); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cfg, _ := h.store.GetConfig(c.Request.Context())
	c.JSON(http.StatusOK, cfg)
}

func (h *Handler) GetTrades(c *gin.Context) {
	coin := c.Query("coin")
	signalType := c.Query("signal_type")
	var active *bool
	if a := c.Query("active"); a != "" {
		v := a == "true"
		active = &v
	}
	trades, err := h.store.GetTrades(c.Request.Context(), coin, signalType, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if trades == nil {
		trades = []models.Trade{}
	}
	c.JSON(http.StatusOK, trades)
}

func (h *Handler) GetActiveTrades(c *gin.Context) {
	active := true
	trades, err := h.store.GetTrades(c.Request.Context(), "", "", &active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if trades == nil {
		trades = []models.Trade{}
	}
	c.JSON(http.StatusOK, trades)
}

func (h *Handler) GetPnL(c *gin.Context) {
	coin := c.Query("coin")
	side := c.Query("side")
	records, err := h.store.GetPnLRecords(c.Request.Context(), coin, side)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if records == nil {
		records = []models.PnLRecord{}
	}
	c.JSON(http.StatusOK, records)
}

func (h *Handler) GetPnLSeries(c *gin.Context) {
	coin := c.Query("coin")
	side := c.Query("side")
	series, err := h.store.GetPnLSeries(c.Request.Context(), coin, side)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if series == nil {
		series = []models.PnLSeries{}
	}
	c.JSON(http.StatusOK, series)
}

func (h *Handler) GetPnLSummary(c *gin.Context) {
	coin := c.Query("coin")
	summaries, err := h.store.GetPnLSummary(c.Request.Context(), coin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if summaries == nil {
		summaries = []models.PnLSummary{}
	}
	c.JSON(http.StatusOK, summaries)
}

func (h *Handler) GetPnLCombined(c *gin.Context) {
	summaries, err := h.store.GetPnLSummary(c.Request.Context(), "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if summaries == nil {
		summaries = []models.PnLSummary{}
	}
	c.JSON(http.StatusOK, summaries)
}

func (h *Handler) GetUnrealizedPnL(c *gin.Context) {
	active := true
	trades, err := h.store.GetTrades(c.Request.Context(), "", "", &active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if trades == nil {
		trades = []models.Trade{}
	}

	cfg, err := h.store.GetConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	priceCache := map[string]float64{}
	items := make([]models.UnrealizedPnLItem, 0, len(trades))
	response := models.UnrealizedPnLResponse{
		Items:     items,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, trade := range trades {
		currentPrice, ok := priceCache[trade.Coin]
		if !ok {
			currentPrice, err = h.fetchBinancePrice(trade.Coin)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
			priceCache[trade.Coin] = currentPrice
		}

		lev := float64(trade.Leverage)
		var grossPnL float64
		if trade.Side == "LONG" {
			grossPnL = (currentPrice - trade.EntryPrice) * trade.Quantity * lev
		} else {
			grossPnL = (trade.EntryPrice - currentPrice) * trade.Quantity * lev
		}

		openCommission := trade.Commission
		closeCommission := trade.Quantity * currentPrice * cfg.CommissionRate * lev
		totalCommission := openCommission + closeCommission
		netPnL := grossPnL - totalCommission

		item := models.UnrealizedPnLItem{
			TradeID:         trade.ID,
			Coin:            trade.Coin,
			SignalType:      trade.SignalType,
			Side:            trade.Side,
			AccountType:     trade.AccountType,
			Quantity:        trade.Quantity,
			EntryPrice:      trade.EntryPrice,
			CurrentPrice:    currentPrice,
			Leverage:        trade.Leverage,
			GrossPnL:        grossPnL,
			OpenCommission:  openCommission,
			CloseCommission: closeCommission,
			TotalCommission: totalCommission,
			NetPnL:          netPnL,
		}

		response.Items = append(response.Items, item)
		response.TotalGrossPnL += grossPnL
		response.TotalCommission += totalCommission
		response.TotalNetPnL += netPnL
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) fetchBinancePrice(symbol string) (float64, error) {
	endpoint := fmt.Sprintf("%s/fapi/v1/ticker/price?symbol=%s", h.binanceBaseURL, url.QueryEscape(symbol))
	resp, err := h.httpClient.Get(endpoint)
	if err != nil {
		return 0, fmt.Errorf("price request failed for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("price request failed for %s: status %d", symbol, resp.StatusCode)
	}

	var body struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("price decode failed for %s: %w", symbol, err)
	}

	price, err := strconv.ParseFloat(body.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("price parse failed for %s: %w", symbol, err)
	}
	return price, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (h *Handler) GetWebhooks(c *gin.Context) {
	coin := c.Query("coin")
	signal := c.Query("signal")
	status := c.Query("status")
	webhooks, err := h.store.GetWebhooks(c.Request.Context(), coin, signal, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if webhooks == nil {
		webhooks = []models.WebhookLog{}
	}
	c.JSON(http.StatusOK, webhooks)
}

func (h *Handler) GetWebhookDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	webhook, err := h.store.GetWebhookDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if webhook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, webhook)
}

func (h *Handler) GetSystemHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "api-server",
		"time":    time.Now().UTC(),
	})
}

func (h *Handler) GetSystemStats(c *gin.Context) {
	stats, err := h.store.GetSystemStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

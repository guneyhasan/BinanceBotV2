package handler

import (
	"net/http"
	"os"
	"strconv"
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
}

func New(store *db.Store) *Handler {
	return &Handler{
		store:              store,
		telegramToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		telegramSignalChat: os.Getenv("TELEGRAM_SIGNAL_CHAT_ID"),
		telegramTradeChat:  os.Getenv("TELEGRAM_TRADE_CHAT_ID"),
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

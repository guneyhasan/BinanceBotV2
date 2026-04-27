package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"webhook-gateway/rabbitmq"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validSignals = map[string]bool{
	"AL1": true, "AL2": true, "AL3": true,
	"SAT1": true, "SAT2": true, "SAT3": true,
}

type Handler struct {
	db     *pgxpool.Pool
	pub    *rabbitmq.Publisher
	secret string
}

type WebhookRequest struct {
	Signal string `json:"signal"`
	Ticker string `json:"ticker"`
	Secret string `json:"secret"`
}

type QueueMessage struct {
	RequestID  string `json:"request_id"`
	Signal     string `json:"signal"`
	Ticker     string `json:"ticker"`
	ReceivedAt string `json:"received_at"`
}

func New(db *pgxpool.Pool, pub *rabbitmq.Publisher, secret string) *Handler {
	return &Handler{db: db, pub: pub, secret: secret}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "webhook-gateway"})
}

func (h *Handler) Webhook(c *gin.Context) {
	var req WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}

	if h.secret != "" && req.Secret != h.secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	req.Signal = strings.ToUpper(strings.TrimSpace(req.Signal))
	req.Ticker = strings.ToUpper(strings.TrimSpace(req.Ticker))
	req.Secret = ""

	if !validSignals[req.Signal] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal, must be AL1/AL2/AL3/SAT1/SAT2/SAT3"})
		return
	}
	if req.Ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticker is required"})
		return
	}

	requestID := uuid.New().String()
	now := time.Now().UTC()

	rawBody, _ := json.Marshal(req)

	_, err := h.db.Exec(context.Background(),
		`INSERT INTO webhook_logs (request_id, coin, signal_type, raw_body, status, received_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'RECEIVED', $5, $5)`,
		requestID, req.Ticker, req.Signal, rawBody, now,
	)
	if err != nil {
		log.Printf("failed to insert webhook_log: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	msg := QueueMessage{
		RequestID:  requestID,
		Signal:     req.Signal,
		Ticker:     req.Ticker,
		ReceivedAt: now.Format(time.RFC3339Nano),
	}
	msgBytes, _ := json.Marshal(msg)

	if err := h.pub.Publish("trading_signals", msgBytes); err != nil {
		log.Printf("failed to publish to trading_signals: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "queue publish error"})
		return
	}

	if err := h.pub.Publish("telegram_raw", msgBytes); err != nil {
		log.Printf("failed to publish to telegram_raw (non-critical): %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "accepted",
		"request_id": requestID,
		"signal":     req.Signal,
		"ticker":     req.Ticker,
	})
}

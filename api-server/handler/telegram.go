package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type telegramTestRequest struct {
	Target string `json:"target"`
}

type telegramSendRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func (h *Handler) TestTelegram(c *gin.Context) {
	var req telegramTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	chatID, label := h.telegramChat(req.Target)
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "telegram chat id is not configured"})
		return
	}
	if h.telegramToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "telegram bot token is not configured"})
		return
	}

	text := fmt.Sprintf(
		"<b>BinanceBotV2 Telegram Test</b>\n\nKanal: <code>%s</code>\nZaman: <code>%s</code>",
		label,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err := h.sendTelegramMessage(c.Request.Context(), chatID, text); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent", "target": req.Target})
}

func (h *Handler) telegramChat(target string) (string, string) {
	switch target {
	case "signal":
		return h.telegramSignalChat, "Sinyal Bildirimleri"
	case "trade":
		return h.telegramTradeChat, "Islem Bildirimleri"
	default:
		return "", ""
	}
}

func (h *Handler) sendTelegramMessage(ctx context.Context, chatID, text string) error {
	body, err := json.Marshal(telegramSendRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.telegramToken)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

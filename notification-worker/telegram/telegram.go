package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type sendMessageReq struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func (c *Client) SendMessage(chatID, text string) error {
	if c.token == "" || chatID == "" {
		log.Printf("telegram not configured, skipping message")
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)

	// Telegram has a 4096 char limit
	if len(text) > 4000 {
		text = text[:4000] + "\n...(truncated)"
	}

	body, _ := json.Marshal(sendMessageReq{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	})

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			return nil
		}

		// Rate limited
		if resp.StatusCode == 429 {
			time.Sleep(time.Duration(attempt) * 5 * time.Second)
			lastErr = fmt.Errorf("rate limited: %s", string(respBody))
			continue
		}

		lastErr = fmt.Errorf("telegram error %d: %s", resp.StatusCode, string(respBody))
		break
	}

	return lastErr
}

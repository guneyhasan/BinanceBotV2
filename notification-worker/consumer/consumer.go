package consumer

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"notification-worker/telegram"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	url            string
	conn           *amqp.Connection
	ch             *amqp.Channel
	tg             *telegram.Client
	signalChatID   string
	tradeChatID    string
	done           chan struct{}
	wg             sync.WaitGroup
}

func New(url string, tg *telegram.Client, signalChatID, tradeChatID string) *Consumer {
	return &Consumer{
		url:          url,
		tg:           tg,
		signalChatID: signalChatID,
		tradeChatID:  tradeChatID,
		done:         make(chan struct{}),
	}
}

func (c *Consumer) connect() error {
	var err error
	for i := 0; i < 30; i++ {
		c.conn, err = amqp.Dial(c.url)
		if err == nil {
			break
		}
		log.Printf("rabbitmq dial attempt %d: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("rabbitmq connection failed: %w", err)
	}

	c.ch, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}

	c.ch.Qos(5, 0, false)
	return nil
}

func (c *Consumer) Start() error {
	if err := c.connect(); err != nil {
		return err
	}

	rawDeliveries, err := c.ch.Consume("telegram_raw", "notif-raw", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume telegram_raw: %w", err)
	}

	tradeDeliveries, err := c.ch.Consume("telegram_trades", "notif-trades", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume telegram_trades: %w", err)
	}

	c.wg.Add(2)
	go c.processRawSignals(rawDeliveries)
	go c.processTradeResults(tradeDeliveries)

	go c.reconnectLoop()

	log.Println("notification-worker started")
	return nil
}

func (c *Consumer) processRawSignals(deliveries <-chan amqp.Delivery) {
	defer c.wg.Done()
	for {
		select {
		case <-c.done:
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			var msg struct {
				RequestID  string `json:"request_id"`
				Signal     string `json:"signal"`
				Ticker     string `json:"ticker"`
				ReceivedAt string `json:"received_at"`
			}
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				d.Ack(false)
				continue
			}

			text := fmt.Sprintf(
				"<b>Yeni Sinyal Geldi</b>\n\n"+
					"Sinyal: <code>%s</code>\n"+
					"Coin: <code>%s</code>\n"+
					"Zaman: <code>%s</code>\n"+
					"ID: <code>%s</code>",
				msg.Signal, msg.Ticker, msg.ReceivedAt, msg.RequestID,
			)

			if err := c.tg.SendMessage(c.signalChatID, text); err != nil {
				log.Printf("telegram signal send failed (will ack anyway): %v", err)
			}
			d.Ack(false)
		}
	}
}

func (c *Consumer) processTradeResults(deliveries <-chan amqp.Delivery) {
	defer c.wg.Done()
	for {
		select {
		case <-c.done:
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			var result struct {
				Success      bool   `json:"success"`
				Signal       string `json:"signal"`
				Coin         string `json:"coin"`
				Side         string `json:"side"`
				Quantity     string `json:"quantity"`
				Price        string `json:"price"`
				ClosedTrade  string `json:"closed_trade"`
				NetPnL       string `json:"net_pnl"`
				Error        string `json:"error"`
				RetryAttempt int    `json:"retry_attempt"`
				RequestID    string `json:"request_id"`
			}
			if err := json.Unmarshal(d.Body, &result); err != nil {
				d.Ack(false)
				continue
			}

			var text string
			if result.Success {
				text = fmt.Sprintf(
					"<b>Islem Basarili</b>\n\n"+
						"Sinyal: <code>%s</code>\n"+
						"Coin: <code>%s</code>\n"+
						"Yon: <code>%s</code>\n"+
						"Miktar: <code>%s</code>\n"+
						"Fiyat: <code>%s</code>",
					result.Signal, result.Coin, result.Side, result.Quantity, result.Price,
				)
				if result.ClosedTrade != "" {
					text += fmt.Sprintf("\nKapatilan: <code>%s</code>", result.ClosedTrade)
				}
				if result.NetPnL != "" {
					text += fmt.Sprintf("\nNet K/Z: <code>%s USD</code>", result.NetPnL)
				}
				if result.RetryAttempt > 1 {
					text += fmt.Sprintf("\n<i>(%d. denemede basarili)</i>", result.RetryAttempt)
				}
			} else {
				text = fmt.Sprintf(
					"<b>Islem Hatasi</b>\n\n"+
						"Sinyal: <code>%s</code>\n"+
						"Coin: <code>%s</code>\n"+
						"Hata: <code>%s</code>\n"+
						"Deneme: <code>%d</code>\n"+
						"ID: <code>%s</code>",
					result.Signal, result.Coin, result.Error, result.RetryAttempt, result.RequestID,
				)
			}

			if err := c.tg.SendMessage(c.tradeChatID, text); err != nil {
				log.Printf("telegram trade send failed (will ack anyway): %v", err)
			}
			d.Ack(false)
		}
	}
}

func (c *Consumer) reconnectLoop() {
	for {
		select {
		case <-c.done:
			return
		case reason, ok := <-c.conn.NotifyClose(make(chan *amqp.Error)):
			if !ok {
				return
			}
			log.Printf("rabbitmq notification connection lost: %v, reconnecting...", reason)
			for {
				if err := c.connect(); err == nil {
					rawDel, err1 := c.ch.Consume("telegram_raw", "notif-raw", false, false, false, false, nil)
					tradeDel, err2 := c.ch.Consume("telegram_trades", "notif-trades", false, false, false, false, nil)
					if err1 == nil && err2 == nil {
						c.wg.Add(2)
						go c.processRawSignals(rawDel)
						go c.processTradeResults(tradeDel)
						break
					}
				}
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (c *Consumer) Stop() {
	close(c.done)
	c.wg.Wait()
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

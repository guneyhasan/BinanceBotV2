package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"trading-engine/engine"
	"trading-engine/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

const maxRetries = 5

type Consumer struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
	eng  *engine.Engine
	done chan struct{}
	wg   sync.WaitGroup
	mu   sync.Mutex
}

func New(url string, eng *engine.Engine) *Consumer {
	return &Consumer{
		url:  url,
		eng:  eng,
		done: make(chan struct{}),
	}
}

func (c *Consumer) connect() error {
	var err error
	for i := 0; i < 30; i++ {
		c.conn, err = amqp.Dial(c.url)
		if err == nil {
			break
		}
		log.Printf("rabbitmq consumer dial attempt %d: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("rabbitmq connection failed: %w", err)
	}

	c.ch, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq channel: %w", err)
	}

	if err := c.declareQueues(); err != nil {
		return err
	}

	if err := c.ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	return nil
}

func (c *Consumer) declareQueues() error {
	if _, err := c.ch.QueueDeclare("trading_signals_dlq", true, false, false, false, nil); err != nil {
		return fmt.Errorf("queue declare trading_signals_dlq: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": "trading_signals_dlq",
	}
	if _, err := c.ch.QueueDeclare("trading_signals", true, false, false, false, args); err != nil {
		return fmt.Errorf("queue declare trading_signals: %w", err)
	}

	if _, err := c.ch.QueueDeclare("telegram_trades", true, false, false, false, nil); err != nil {
		return fmt.Errorf("queue declare telegram_trades: %w", err)
	}

	return nil
}

func (c *Consumer) Start(ctx context.Context) error {
	if err := c.connect(); err != nil {
		return err
	}

	deliveries, err := c.ch.Consume("trading_signals", "trading-engine", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.processLoop(ctx, deliveries)
	}()

	go c.reconnectLoop(ctx)

	log.Println("trading-engine consumer started")
	return nil
}

func (c *Consumer) processLoop(ctx context.Context, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case d, ok := <-deliveries:
			if !ok {
				log.Println("delivery channel closed, waiting for reconnect")
				return
			}
			c.handleDelivery(ctx, d)
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, d amqp.Delivery) {
	var msg models.QueueMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		log.Printf("invalid message body, nacking: %v", err)
		d.Nack(false, false)
		return
	}

	log.Printf("processing signal: %s %s (request_id: %s)", msg.Signal, msg.Ticker, msg.RequestID)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := c.eng.ProcessSignal(ctx, msg, attempt)
		if err == nil && result.Success {
			log.Printf("signal processed successfully: %s %s (attempt %d)", msg.Signal, msg.Ticker, attempt)
			c.publishTradeNotification(result)
			d.Ack(false)
			return
		}
		lastErr = err
		if attempt < maxRetries {
			c.eng.ProcessSignalRetryLog(ctx, msg.RequestID)
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("attempt %d/%d failed for %s %s: %v, retrying in %v", attempt, maxRetries, msg.Signal, msg.Ticker, err, backoff)

			c.publishTradeNotification(&engine.TradeResult{
				Success:      false,
				Signal:       msg.Signal,
				Coin:         msg.Ticker,
				Error:        fmt.Sprintf("attempt %d/%d failed: %v, retrying...", attempt, maxRetries, err),
				RetryAttempt: attempt,
				RequestID:    msg.RequestID,
			})

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				d.Nack(false, true)
				return
			}
		}
	}

	log.Printf("all %d attempts failed for %s %s: %v, sending to DLQ", maxRetries, msg.Signal, msg.Ticker, lastErr)
	errMsg := ""
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	c.eng.ProcessSignalFail(ctx, msg.RequestID, errMsg)

	c.publishTradeNotification(&engine.TradeResult{
		Success:      false,
		Signal:       msg.Signal,
		Coin:         msg.Ticker,
		Error:        fmt.Sprintf("PERMANENTLY FAILED after %d attempts: %v", maxRetries, lastErr),
		RetryAttempt: maxRetries,
		RequestID:    msg.RequestID,
	})

	d.Nack(false, false)
}

func (c *Consumer) publishTradeNotification(result *engine.TradeResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ch == nil {
		return
	}
	body, _ := json.Marshal(result)
	err := c.ch.Publish("", "telegram_trades", false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	})
	if err != nil {
		log.Printf("failed to publish trade notification (non-critical): %v", err)
	}
}

func (c *Consumer) reconnectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case reason, ok := <-c.conn.NotifyClose(make(chan *amqp.Error)):
			if !ok {
				return
			}
			log.Printf("rabbitmq consumer connection lost: %v, reconnecting...", reason)
			c.mu.Lock()
			for {
				if err := c.connect(); err == nil {
					deliveries, err := c.ch.Consume("trading_signals", "trading-engine", false, false, false, false, nil)
					if err == nil {
						c.wg.Add(1)
						go func() {
							defer c.wg.Done()
							c.processLoop(ctx, deliveries)
						}()
						c.mu.Unlock()
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

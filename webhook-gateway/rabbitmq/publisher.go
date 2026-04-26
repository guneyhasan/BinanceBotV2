package rabbitmq

import (
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
}

func NewPublisher(url string) (*Publisher, error) {
	p := &Publisher{url: url}
	if err := p.connect(); err != nil {
		return nil, err
	}
	go p.reconnectLoop()
	return p, nil
}

func (p *Publisher) connect() error {
	var err error
	for i := 0; i < 30; i++ {
		p.conn, err = amqp.Dial(p.url)
		if err == nil {
			break
		}
		log.Printf("rabbitmq dial attempt %d: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("rabbitmq connection failed after retries: %w", err)
	}

	p.ch, err = p.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq channel: %w", err)
	}

	queues := []string{"trading_signals", "telegram_raw", "telegram_trades", "trading_signals_dlq"}
	for _, q := range queues {
		args := amqp.Table{}
		if q == "trading_signals" {
			args["x-dead-letter-exchange"] = ""
			args["x-dead-letter-routing-key"] = "trading_signals_dlq"
		}
		_, err = p.ch.QueueDeclare(q, true, false, false, false, args)
		if err != nil {
			return fmt.Errorf("queue declare %s: %w", q, err)
		}
	}

	log.Println("rabbitmq publisher connected")
	return nil
}

func (p *Publisher) reconnectLoop() {
	for {
		reason, ok := <-p.conn.NotifyClose(make(chan *amqp.Error))
		if !ok {
			return
		}
		log.Printf("rabbitmq connection lost: %v, reconnecting...", reason)
		p.mu.Lock()
		for {
			if err := p.connect(); err == nil {
				break
			}
			time.Sleep(5 * time.Second)
		}
		p.mu.Unlock()
	}
}

func (p *Publisher) Publish(queue string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.ch.Publish("", queue, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    time.Now(),
	})
}

func (p *Publisher) Close() {
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

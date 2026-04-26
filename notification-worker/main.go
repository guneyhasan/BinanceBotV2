package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-worker/consumer"
	"notification-worker/telegram"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rabbitURL := envOrDefault("RABBITMQ_URL", "amqp://tradingbot:change_me_in_production@localhost:5672/")
	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	signalChatID := os.Getenv("TELEGRAM_SIGNAL_CHAT_ID")
	tradeChatID := os.Getenv("TELEGRAM_TRADE_CHAT_ID")

	tg := telegram.NewClient(tgToken)
	cons := consumer.New(rabbitURL, tg, signalChatID, tradeChatID)

	if err := cons.Start(); err != nil {
		log.Fatalf("consumer start failed: %v", err)
	}

	log.Println("notification-worker running")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down notification-worker...")
	cons.Stop()
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

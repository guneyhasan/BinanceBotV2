package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trading-engine/binance"
	"trading-engine/consumer"
	"trading-engine/db"
	"trading-engine/engine"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rabbitURL := envOrDefault("RABBITMQ_URL", "amqp://tradingbot:change_me_in_production@localhost:5672/")
	postgresDSN := envOrDefault("POSTGRES_DSN", "postgres://tradingbot:change_me_in_production@localhost:5432/tradingbot?sslmode=disable")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, postgresDSN)
	if err != nil {
		log.Fatalf("postgres connection failed: %v", err)
	}
	defer pool.Close()

	for i := 0; i < 30; i++ {
		if err = pool.Ping(ctx); err == nil {
			break
		}
		log.Printf("waiting for postgres... attempt %d", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("postgres not reachable: %v", err)
	}

	alClient := binance.NewClient(
		envOrDefault("BINANCE_BASE_URL", "https://testnet.binancefuture.com"),
		os.Getenv("BINANCE_AL_API_KEY"),
		os.Getenv("BINANCE_AL_API_SECRET"),
	)
	satClient := binance.NewClient(
		envOrDefault("BINANCE_BASE_URL", "https://testnet.binancefuture.com"),
		os.Getenv("BINANCE_SAT_API_KEY"),
		os.Getenv("BINANCE_SAT_API_SECRET"),
	)

	store := db.NewStore(pool)
	eng := engine.New(store, alClient, satClient)

	cons := consumer.New(rabbitURL, eng)
	if err := cons.Start(ctx); err != nil {
		log.Fatalf("consumer start failed: %v", err)
	}

	log.Println("trading-engine running")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down trading-engine...")
	cancel()
	cons.Stop()
	log.Println("trading-engine stopped")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

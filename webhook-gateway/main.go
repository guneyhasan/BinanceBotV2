package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webhook-gateway/handler"
	"webhook-gateway/rabbitmq"

	"github.com/gin-gonic/gin"
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

	pub, err := rabbitmq.NewPublisher(rabbitURL)
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer pub.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	h := handler.New(pool, pub, os.Getenv("WEBHOOK_SECRET"))
	r.GET("/health", h.Health)
	r.POST("/webhook", h.Webhook)

	srv := &http.Server{Addr: ":8080", Handler: r}

	go func() {
		log.Println("webhook-gateway listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down webhook-gateway...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

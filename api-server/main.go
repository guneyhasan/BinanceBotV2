package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-server/db"
	"api-server/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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

	store := db.NewStore(pool)
	h := handler.New(store)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", h.Health)

	api := r.Group("/api")
	{
		api.GET("/config", h.GetConfig)
		api.PUT("/config", h.UpdateConfig)
		api.GET("/trades", h.GetTrades)
		api.GET("/trades/active", h.GetActiveTrades)
		api.GET("/pnl", h.GetPnL)
		api.GET("/pnl/series", h.GetPnLSeries)
		api.GET("/pnl/summary", h.GetPnLSummary)
		api.GET("/pnl/combined", h.GetPnLCombined)
		api.GET("/webhooks", h.GetWebhooks)
		api.GET("/webhooks/:id", h.GetWebhookDetail)
		api.GET("/system/health", h.GetSystemHealth)
		api.GET("/system/stats", h.GetSystemStats)
		api.POST("/telegram/test", h.TestTelegram)
	}

	srv := &http.Server{Addr: ":8081", Handler: r}

	go func() {
		log.Println("api-server listening on :8081")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down api-server...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

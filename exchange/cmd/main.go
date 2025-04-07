package main

import (
	"Exchange/internal/config"
	"Exchange/internal/http_client"
	"Exchange/internal/storage/redis"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

const (
	envDevMac = "dev_mac"
	envDevWin = "dev_win"
	envProd   = "prod"
)

func main() {
	// TODO: init config

	cfg := config.MustLoad()

	// TODO: init logger

	log := setupLogger(cfg.Env)

	log.Info("starting application",
		slog.String("env", cfg.Env),
		slog.Any("cfg", cfg),
	)

	priceClient := http_client.New(*cfg, *log)
	redisClient := redis.New(cfg.RedisCfg)

	ctx := context.Background()
	for {
		prices, _ := priceClient.GetPrice()
		redisClient.SavePrices(ctx, prices)
		// renewedPrices, _ := redisClient.GetAllPrices(context.Background())
		for _, priceResp := range prices {
			price, _ := redisClient.GetPrice(ctx, priceResp.Symbol)
			fmt.Println(priceResp.Symbol, ":", price)
		}
		time.Sleep(30 * time.Second)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envDevMac:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDevWin:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

package main

import (
	"Exchange/internal/config"
	"Exchange/internal/http_client"
	"Exchange/internal/services/order"
	"Exchange/internal/services/trade"
	user "Exchange/internal/services/user"
	"Exchange/internal/storage/postgres"
	"Exchange/internal/storage/redis"
	handler "Exchange/transport"
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
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

	// postgres://postgres:postgres@localhost:5432/exchange?sslmode=disable
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.PostgresCfgWin.Username,
		cfg.PostgresCfgWin.Password,
		cfg.PostgresCfgWin.Host,
		cfg.PostgresCfgWin.Port,
		cfg.PostgresCfgWin.Database)

	storage, err := postgres.New(connString)
	if err != nil {
		log.Error("failed to connect to postgres")
	}

	priceClient := http_client.New(*cfg, *log)
	redisClient := redis.New(cfg.RedisCfg)

	ctx := context.Background()
	go func() {
		for {
			prices, _ := priceClient.GetPrice()
			_ = redisClient.SavePrices(ctx, prices)
			/*
				renewedPrices, _ := redisClient.GetAllPrices(context.Background())
				for _, priceResp := range prices {
					price, _ := redisClient.GetPrice(ctx, priceResp.Symbol)
					fmt.Println(priceResp.Symbol, ":", price)
				}
			*/
			time.Sleep(5 * time.Second)
		}
	}()

	validate := validator.New()

	userService := user.New(*log, storage, storage)
	orderService := order.New(*log, storage, storage, storage)
	tradeService := trade.New(log, *orderService, *redisClient)

	userHandler := handler.NewUserHandler(log, userService, validate)
	tradeHandler := handler.NewTradeHandler(log, tradeService, validate)

	// Настройка маршрутов
	r := chi.NewRouter()
	r.Mount("/user", userHandler.Routes())
	r.Mount("/trade", tradeHandler.Routes())

	// Запуск сервера
	port := ":8080"
	log.Info("Starting server on " + port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Error("Server failed", "error", err)
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

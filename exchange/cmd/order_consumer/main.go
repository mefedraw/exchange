package main

import (
	"Exchange/internal/config"
	"Exchange/internal/storage/redis"
	"context"
	"github.com/nats-io/nats.go"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	// Инициализация логгера
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()
	cfg := config.MustLoad()

	redis := redis.New(cfg.RedisCfg)

	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		logger.Error("NATS connection failed", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("JetStream init failed", "error", err)
		os.Exit(1)
	}

	const pricesSubj = "prices."
	// Подписка с правильными опциями
	sub, err := js.Subscribe(pricesSubj+"*", func(msg *nats.Msg) {
		// logger.Info("Received message", "subject", msg.Subject, "body", string(msg.Data))

		// todo: msg handling
		const quoteAsset = "USDT"
		key := strings.TrimSuffix(msg.Subject, quoteAsset)
		key = strings.TrimPrefix(key, pricesSubj)
		key = key + "/" + quoteAsset
		liqOrders, err := redis.GetLiqOrders(ctx, key, string(msg.Data))
		if err != nil {
			logger.Error("Get liq orders failed", "error", err)
		}
		for _, liqOrder := range liqOrders {
			logger.Info("order for liquidation", "order_id", liqOrder.String())
		}
		msg.Ack() // Подтверждаем обработку
	},
		nats.Durable("ORDER_PROCESSOR"),
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	if err != nil {
		logger.Error("Subscribe failed", "error", err)
		os.Exit(1)
	}
	defer sub.Unsubscribe()

	logger.Info("Service started successfully")

	// Ожидание сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Info("Shutting down...")
}

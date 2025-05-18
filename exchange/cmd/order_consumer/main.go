package main

import (
	"Exchange/internal/config"
	"Exchange/internal/services/order"
	"Exchange/internal/services/trade"
	"Exchange/internal/storage/postgres"
	"Exchange/internal/storage/redis"
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
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

	//todo: REDIS
	redis := redis.New(cfg.RedisCfg)

	//todo: POSTGRES
	var connString string
	if cfg.Env == "dev_mac" {
		slog.Info("connecting to postgres via dev_mac")
		connString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.PostgresCfgMac.Username,
			cfg.PostgresCfgMac.Password,
			cfg.PostgresCfgMac.Host,
			cfg.PostgresCfgMac.Port,
			cfg.PostgresCfgMac.Database)
	} else if cfg.Env == "dev_win" {
		slog.Info("connecting to postgres via dev_win")
		connString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.PostgresCfgWin.Username,
			cfg.PostgresCfgWin.Password,
			cfg.PostgresCfgWin.Host,
			cfg.PostgresCfgWin.Port,
			cfg.PostgresCfgWin.Database)
	}

	storage, err := postgres.New(connString)
	if err != nil {
		slog.Error("failed to connect to postgres")
	}
	orderService := order.New(*logger, storage, storage, storage)
	tradeService := trade.New(logger, *orderService, *redis)

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
		currentPrice := string(msg.Data)
		liqOrders, err := redis.GetLiqOrders(ctx, key, currentPrice)
		if err != nil {
			logger.Error("Get liq orders failed", "error", err)
		}
		for _, liqOrderId := range liqOrders {
			logger.Info("order for liquidation", "order_id", liqOrderId.String())
			curPriceDecimal, _ := decimal.NewFromString(currentPrice)
			id, err := tradeService.LiquidateTradeDeal(ctx, liqOrderId, curPriceDecimal)
			if err != nil {
				logger.Error("liquidation was failed", "id", id, "error", err)
			} else {
				logger.Info("order was successfully liquidated", "order_id", id.String())
			}
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

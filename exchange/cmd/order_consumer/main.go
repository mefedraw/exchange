package main

import (
	"Exchange/internal/config"
	"Exchange/internal/storage/redis"
	"context"
	"github.com/nats-io/nats.go"
	"log/slog"
	"strings"
)

func main() {
	ctx := context.Context(context.Background())
	cfg := config.MustLoad()
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		slog.Error("order consumer nats.Connect err:", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		slog.Error("order consumer jetstream creating err:", err)
	}
	redis := redis.New(cfg.RedisCfg)

	const liqOrdersTopic = "liq-orders."
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "LIQ-ORDERS-STREAM",
		Subjects: []string{"liq-orders."},
	})
	if err != nil {
		slog.Error("order consumer AddStream err:", err)
	}

	const prefix = "prices."
	js.Subscribe("prices.*", func(msg *nats.Msg) {
		if strings.HasPrefix(msg.Subject, prefix) {
			symbol := msg.Subject[len(prefix):]
			liqOrders, err := redis.GetLiqOrders(ctx, symbol, string(msg.Data))
			if err != nil {
				slog.Error("order consumer GetLiqOrders err:", err)
			}
			for _, liqOrderId := range liqOrders {
				_, err = js.Publish(liqOrdersTopic, []byte(liqOrderId.String()))
				if err != nil {
					slog.Error("liq-order Publish err:", err)
				}
				slog.Info("order for liquidation: ", liqOrderId)
			}
		}
	}, nats.Durable("ORDER_PROCESSOR"))

}

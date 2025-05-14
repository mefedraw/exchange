package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"log/slog"
)

func main() {
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		slog.Error("nats.Connect err:", err)
	}
	js, err := nc.JetStream()
	// Создание потока
	js.AddStream(&nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"orders.*"},
	})

	// Публикация
	js.Publish("orders.new", []byte("order123"))

	// Подписка (durable consumer)
	js.Subscribe("orders.*", func(msg *nats.Msg) {
		fmt.Printf("Received: %s\n", msg.Data)
		msg.Ack() // Подтверждение обработки
	}, nats.Durable("ORDER_PROCESSOR"))
}

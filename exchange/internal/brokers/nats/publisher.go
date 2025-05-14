package nats

import (
	"github.com/nats-io/nats.go"
	"log/slog"
)

type Publisher struct {
	log slog.Logger
	Js  *nats.JetStreamContext
}

func New(nc *nats.Conn) (*Publisher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &Publisher{Js: &js}, nil
}

//
//func (p *Publisher) Publish(ctx context.Context, subject string, msg interface{}) error {
//	const op = "nats.Publish"
//	data, err := json.Marshal(msg)
//	if err != nil {
//		p.log.Error("marshalling message", "op", op, "error", err, "msg", msg)
//		return fmt.Errorf("marshal %T: %w", msg, err)
//	}
//	slog.Debug("message published", "msg", msg)
//
//
//	if err != nil {
//		p.log.Error("publishing message", "op", op, "error", err)
//		return fmt.Errorf("publishing message: %w", err)
//	}
//
//}

package liquidation

import (
	"Exchange/internal/domain/models"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"log"
	"log/slog"
	"runtime"
	"sync"
)

type pg interface {
}
type Liquidator struct {
	Nc         *nats.Conn
	finder     finder
	liquidator liquidator
}

type finder interface {
	GetLiqOrders(ctx context.Context, markPrice decimal.Decimal, ticker string) ([]uuid.UUID, error)
}

type liquidator interface {
	LiquidateTradeDeal(ctx context.Context, orderId uuid.UUID) (uuid.UUID, error)
}

func NewLiquidator(nc *nats.Conn, orderService finder) (*Liquidator, error) {
	return &Liquidator{
		Nc:     nc,
		finder: orderService,
	}, nil
}

func (l *Liquidator) Process() {
	ctx := context.Background()
	updates := make(chan models.PriceResponse, 1024)
	var wg sync.WaitGroup
	workerCount := runtime.NumCPU()
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for pu := range updates {
				l.handlePriceUpdate(ctx, pu)
			}
		}(i)
	}

	_, _ = l.Nc.Subscribe("prices.*", func(msg *nats.Msg) { // sub, err
		var pu models.PriceResponse
		if err := json.Unmarshal(msg.Data, &pu); err != nil {
			log.Printf("invalid msg: %v", err)
			return
		}

		updates <- pu
	})
}

func (l *Liquidator) handlePriceUpdate(ctx context.Context, pu models.PriceResponse) {
	const op = "liquidation.handlePriceUpdate"
	liqOrders, err := l.finder.GetLiqOrders(ctx, decimal.RequireFromString(pu.Price), pu.Symbol)
	if err != nil {
		slog.Error("get liq orders error:", "op", op, "err", err)
	}
	slog.Info("liq orders get", "len", len(liqOrders))

	for _, order := range liqOrders {
		orderId, err := l.liquidator.LiquidateTradeDeal(ctx, order)
		if err != nil {
			slog.Error("get liq order error:", "op", op, "err", err)
		}

		slog.Debug("get liq order id:", "op", op, "orderId", orderId)
	}
}

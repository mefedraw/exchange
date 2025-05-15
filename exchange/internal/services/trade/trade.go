package trade

import (
	"Exchange/internal/domain/models"
	"Exchange/internal/services/order"
	"Exchange/internal/storage/postgres"
	"Exchange/internal/storage/redis"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"log/slog"
	"strings"
)

var (
	ErrNegativeMargin     = errors.New("margin can't be negative")
	ErrNegativeEntryPrice = errors.New("entry price can't be negative")
	ErrInvalidLeverage    = errors.New("invalid leverage")
)

type Trade struct {
	log          slog.Logger
	orderService order.Order
	redis        redis.Redis
}

func (t *Trade) GetUserOrders(ctx context.Context, id int64) ([]models.Order, error) {
	const op = "order.GetAllUserOrders"
	// t.log.With("op", op)
	orders, err := t.orderService.Manager.GetUserOrders(ctx, id)
	if err != nil {
		t.log.Error("failed to get all user order", "id", id, "error", err)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	t.log.Debug("got all user orders", "id", id)
	return orders, nil
}

func New(log *slog.Logger, orderService order.Order, redis redis.Redis) *Trade {
	return &Trade{
		log:          *log,
		orderService: orderService,
		redis:        redis,
	}
}

func (t *Trade) OpenTradeDeal(ctx context.Context,
	userId int64,
	ticker string,
	orderType models.OrderType,
	margin decimal.Decimal,
	leverage uint8) (uuid.UUID, error) {
	const op = "Trade.OpenTradeDeal"

	if margin.LessThanOrEqual(decimal.Zero) {
		return uuid.Nil, ErrNegativeMargin
	}
	if leverage <= 0 {
		return uuid.Nil, ErrInvalidLeverage
	}

	entryPrice, err := t.redis.GetPrice(ctx, ticker)
	t.log.Info("OpenTradeDeal: ticker:", ticker)
	if err != nil {
		t.log.Error("Error getting entryPrice", "error", err, "ticker", ticker)
		return uuid.Nil, fmt.Errorf("failed to get entry price. %s: %w", op, err)
	}
	entryPriceDec, err := decimal.NewFromString(entryPrice)
	if err != nil {
		t.log.Error("Error converting entryPrice", "error", err, "entryPrice", entryPrice)
		return uuid.Nil, fmt.Errorf("failed to convert entryPrice. %s: %w", op, err)
	}

	// todo: liquidation price
	lev := decimal.NewFromInt(int64(leverage))
	var liqPrice decimal.Decimal
	if orderType == models.Long {
		// long: entryPrice * (leverage-1)/leverage
		liqPrice = entryPriceDec.Mul(lev.Sub(decimal.NewFromInt(1))).Div(lev)
	} else {
		// short: entryPrice * (leverage+1)/leverage
		liqPrice = entryPriceDec.Mul(lev.Add(decimal.NewFromInt(1))).Div(lev)
	}

	id, err := t.orderService.OpenOrder(ctx, userId, ticker, orderType, margin, leverage, entryPriceDec, liqPrice)
	if err != nil {
		t.log.Error("Error opening order", "error", err, "userId", userId)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	err = t.redis.SaveOrder(ctx, models.Order{Ticker: ticker, Type: orderType, Id: id, LiquidationPrice: liqPrice})
	if err != nil {
		t.log.Error("Error saving order to redis", "error", err, "orderId", id)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (t *Trade) CloseTradeDeal(ctx context.Context, orderId uuid.UUID, ticker string) (uuid.UUID, error) {
	const op = "Trade.CloseTradeDeal"

	order, err := t.orderService.GetOrder(ctx, orderId)
	if err != nil {
		if errors.Is(err, postgres.ErrOrderNotExists) {
			t.log.Error("Order not exists", "orderId", orderId)
			return uuid.Nil, fmt.Errorf("%s: %w", op, err)
		}
		t.log.Error("Error getting order", "error", err, "orderId", orderId)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// getting closePrice
	closePrice, err := t.redis.GetPrice(ctx, ticker)
	if err != nil {
		t.log.Error("Error getting closePrice", "error", err, "ticker", ticker)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	closePriceDec, err := decimal.NewFromString(closePrice)
	if err != nil {
		t.log.Error("Error converting closePrice", "error", err, "closePrice", closePrice)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// calculating BalanceIncrease
	// -> ((closeP-entryP)/entry)*leverage*margin+margin

	orderProfit := calculateOrderProfit(order, closePriceDec)
	balanceInc := order.Margin.Add(orderProfit)

	id, err := t.orderService.CloseOrder(ctx, orderId, closePriceDec, balanceInc)
	if err != nil {
		t.log.Error("Error closing order", "error", err, "orderId", orderId)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	t.redis.RemoveOrder(ctx, id.String())

	return id, nil
}

func (t *Trade) LiquidateTradeDeal(ctx context.Context, orderId uuid.UUID, closePrice decimal.Decimal) (uuid.UUID, error) {
	const op = "trade.LiquidateTradeDeal"
	orderId, err := t.orderService.LiquidateOrder(ctx, orderId, closePrice)
	if err != nil {
		t.log.Error("Error getting order", "error", err, "orderId", orderId)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	return orderId, nil
}

func calculateOrderProfit(order models.Order, closePriceDec decimal.Decimal) decimal.Decimal {
	priceDiff := closePriceDec.Sub(order.EntryPrice)
	priceChange := priceDiff.Div(order.EntryPrice)

	// if order is short -> use reverse logic
	if order.Type == models.Short {
		priceChange = priceChange.Mul(decimal.NewFromInt(-1))
	}

	profit := priceChange.Mul(decimal.NewFromInt(int64(order.Leverage))).Mul(order.Margin)

	return profit
}

func handleTicker(ticker string) string {
	parts := strings.Split(ticker, "/")
	return parts[0] + parts[1]
}

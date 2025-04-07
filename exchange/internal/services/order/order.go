package order

import (
	"Exchange/internal/domain/models"
	"Exchange/internal/services/user"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"log/slog"
	"strings"
	"time"
)

var (
	ErrInvalidTicker     = errors.New("ticker is invalid")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Order struct {
	log     slog.Logger
	manager Manager
	tp      TradingPairManager
	um      user.Manager
}

type Manager interface {
	CreateOrder(ctx context.Context,
		id uuid.UUID,
		userId int64,
		pairId int64,
		orderType models.OrderType,
		margin decimal.Decimal,
		leverage uint8,
		entryPrice decimal.Decimal,
		status models.OrderStatus,
		createdAt time.Time) (uuid.UUID, error)
	GetOrder(ctx context.Context, id uuid.UUID) (models.Order, error)
	GetUserOrders(ctx context.Context, userId int64) ([]models.Order, error)
	OpenOrder(
		ctx context.Context,
		id uuid.UUID,
		userId int64,
		pairId int64,
		orderType models.OrderType,
		margin decimal.Decimal,
		leverage uint8,
		entryPrice decimal.Decimal,
		status models.OrderStatus,
		createdAt time.Time,
	) (orderID uuid.UUID, err error)
	CloseOrder(
		ctx context.Context,
		orderID uuid.UUID,
		closePrice decimal.Decimal,
		balanceIncrease decimal.Decimal,
	) (orderId uuid.UUID, err error)
}

type TradingPairManager interface {
	GetTradingPairId(baseAsset, quoteAsset string) (int64, error)
}

func New(log slog.Logger, manager Manager, tp TradingPairManager, userManager user.Manager) *Order {
	return &Order{
		log:     log,
		manager: manager,
		tp:      tp,
		um:      userManager,
	}
}

func (o *Order) CreateOrder(ctx context.Context,
	userId int64,
	ticker string,
	orderType models.OrderType,
	margin decimal.Decimal,
	leverage uint8,
	entryPrice decimal.Decimal) (uuid.UUID, error) {
	const op = "order.CreateOrder"

	baseAsset, quoteAsset, err := checkTicker(ticker)
	if err != nil {
		o.log.Error("Invalid ticker", "ticker", ticker, "err", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	pairId, err := o.tp.GetTradingPairId(baseAsset, quoteAsset)
	if err != nil {
		o.log.Error("failed to get trading pair id", "error", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	//check if user exists
	_, err = o.um.GetUserById(ctx, userId)
	if err != nil {
		o.log.Error("failed to get user", "userId", userId, "err", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	orderId := uuid.New()
	orderStatus := models.Open
	createdAt := time.Now()

	orderId, err = o.manager.CreateOrder(ctx, orderId, userId, pairId, orderType, margin, leverage, entryPrice, orderStatus, createdAt)
	if err != nil {
		o.log.Error("failed to create order", "error", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return orderId, nil
}

func (o *Order) OpenOrder(ctx context.Context,
	userId int64,
	ticker string,
	orderType models.OrderType,
	margin decimal.Decimal,
	leverage uint8,
	entryPrice decimal.Decimal) (uuid.UUID, error) {
	const op = "order.CreateOrder"

	baseAsset, quoteAsset, err := checkTicker(ticker)
	if err != nil {
		o.log.Error("Invalid ticker", "ticker", ticker, "err", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	pairId, err := o.tp.GetTradingPairId(baseAsset, quoteAsset)
	if err != nil {
		o.log.Error("failed to get trading pair id", "error", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	//check if user exists
	currUser, err := o.um.GetUserById(ctx, userId)
	if err != nil {
		o.log.Error("failed to get user", "userId", userId, "err", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	//check if user has enough funds to open order
	if currUser.Balance.LessThan(margin) {
		o.log.Info("insufficient balance for order", "userId", userId, "balance", currUser.Balance)
		return uuid.Nil, fmt.Errorf("%s: %w", op, ErrInsufficientFunds)
	}

	orderId := uuid.New()
	orderStatus := models.Open
	createdAt := time.Now()

	orderId, err = o.manager.OpenOrder(ctx, orderId, userId, pairId, orderType, margin, leverage, entryPrice, orderStatus, createdAt)
	if err != nil {
		o.log.Error("failed to create order", "error", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return orderId, nil
}

func (o *Order) CloseOrder(ctx context.Context,
	orderID uuid.UUID,
	closePrice decimal.Decimal,
	balanceIncrease decimal.Decimal) (uuid.UUID, error) {
	const op = "order.CloseOrder"

	// check if order exists
	_, err := o.manager.GetOrder(ctx, orderID)
	if err != nil {
		o.log.Error("failed to get order", "order", orderID, "error", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// TODO: check that order owner equals method caller
	// 2 ways:
	// 1) liquidation calls CloseOrder() and sets order status
	// 2) liquidation calls its own method

	orderId, err := o.manager.CloseOrder(ctx, orderID, closePrice, balanceIncrease)
	if err != nil {
		o.log.Error("failed to close order", "error", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return orderId, nil
}

func checkTicker(ticker string) (string, string, error) {
	const op = "order.CheckTicker"
	parts := strings.Split(ticker, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%s: %w", op, ErrInvalidTicker)
	}

	baseAsset := parts[0]
	quoteAsset := parts[1]

	return baseAsset, quoteAsset, nil
}

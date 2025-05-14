package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type OrderType string

const (
	Long  OrderType = "long"
	Short OrderType = "short"
)

type OrderStatus string

const (
	Open       OrderStatus = "open"
	Closed     OrderStatus = "closed"
	Liquidated OrderStatus = "liquidated"
	Canceled   OrderStatus = "canceled"
)

type Order struct {
	Id               uuid.UUID
	UserId           int64
	PairId           int64
	Type             OrderType
	Margin           decimal.Decimal
	Leverage         uint8
	EntryPrice       decimal.Decimal
	ClosePrice       *decimal.Decimal
	Status           OrderStatus
	CreatedAt        time.Time
	LiquidationPrice decimal.Decimal
	Ticker           string
}

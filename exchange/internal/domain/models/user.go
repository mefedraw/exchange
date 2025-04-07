package models

import (
	"github.com/shopspring/decimal"
	"time"
)

type User struct {
	Id       int64
	Email    string
	PassHash string
	Balance  decimal.Decimal
	Created  time.Time
}

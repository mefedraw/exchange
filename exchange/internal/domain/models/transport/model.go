package transport

import "github.com/shopspring/decimal"

type ErrorResponse struct {
	Error string `json:"error"`
}
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type RegisterResponse struct {
	Id int64 `json:"id"`
}

type GetBalanceRequest struct {
	Id int64 `json:"id" validate:"required"`
}

type BalanceRequest struct {
	UserID int64 `json:"id" validate:"required,gt=0"`
}

type IncreaseBalanceRequest struct {
	Id     int64           `json:"id" validate:"required,gt=0"`
	Amount decimal.Decimal `json:"amount" validate:"required"`
}

type DecreaseBalanceRequest struct {
	Id     int64           `json:"id" validate:"required,gt=0"`
	Amount decimal.Decimal `json:"amount" validate:"required"`
}

type BalanceResponse struct {
	UserID  int64           `json:"id"`
	Balance decimal.Decimal `json:"balance"`
}

package handler

import (
	"Exchange/internal/domain/models"
	"Exchange/internal/domain/models/transport"
	"Exchange/internal/services/trade"
	"Exchange/internal/storage/postgres"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"log/slog"
	"net/http"
)

type TradeHandler struct {
	log          *slog.Logger
	tradeService tradeService
	validate     *validator.Validate
}

type tradeService interface {
	OpenTradeDeal(ctx context.Context,
		userId int64,
		ticker string,
		orderType models.OrderType,
		margin decimal.Decimal,
		leverage uint8) (uuid.UUID, error)
	CloseTradeDeal(ctx context.Context, orderId uuid.UUID, ticker string) (uuid.UUID, error)
	GetUserOrders(ctx context.Context, id int64) ([]models.Order, error)
}

func NewTradeHandler(log *slog.Logger, tradeService tradeService, validate *validator.Validate) *TradeHandler {
	return &TradeHandler{
		log:          log,
		tradeService: tradeService,
		validate:     validate,
	}
}

func (t *TradeHandler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	router.Route("/api/trade", func(router chi.Router) {
		router.Group(func(routerWithAuth chi.Router) {
			// routerWithAuth.Use(h.authMiddleware)

			routerWithAuth.Post("/open", t.PostOpenTrade)
			routerWithAuth.Post("/close", t.PostCloseTrade)
			routerWithAuth.Get("/orders", t.GetUserOrders)
		})
	})

	return router
}

func (h *TradeHandler) PostOpenTrade(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req transport.OpenTradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid request format",
		})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.log.Error("Validation failed", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid trade parameters",
		})
		return
	}

	orderID, err := h.tradeService.OpenTradeDeal(r.Context(), req.UserID, req.Ticker, req.OrderType, req.Margin, req.Leverage)
	if err != nil {
		h.log.Error("Failed to open trade", "error", err, "userId", req.UserID)

		switch {
		case errors.Is(err, trade.ErrNegativeMargin):
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(transport.ErrorResponse{
				Error: "Margin must be positive",
			})
		case errors.Is(err, trade.ErrInvalidLeverage):
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(transport.ErrorResponse{
				Error: "Invalid leverage value",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(transport.ErrorResponse{
				Error: "Failed to open trade",
			})
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transport.OpenTradeResponse{
		OrderID: orderID,
	})
}

func (h *TradeHandler) PostCloseTrade(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req transport.CloseTradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid request format",
		})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.log.Error("Validation failed", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Order ID and ticker are required",
		})
		return
	}

	orderID, err := h.tradeService.CloseTradeDeal(r.Context(), req.OrderID, req.Ticker)
	if err != nil {
		h.log.Error("Failed to close trade", "error", err, "orderId", req.OrderID)

		if errors.Is(err, postgres.ErrOrderNotExists) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(transport.ErrorResponse{
				Error: "Order not found",
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to close trade",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transport.CloseTradeResponse{
		OrderID: orderID,
	})
}

func (t *TradeHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Парсим запрос
	var req transport.GetOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.log.Error("Error decoding orders request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to decode request body",
		})
		return
	}

	// 2. Валидируем запрос
	if err := t.validate.Struct(&req); err != nil {
		t.log.Error("Error validating balance request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid request format",
		})
		return
	}

	// 3. Получаем баланс
	orders, err := t.tradeService.GetUserOrders(r.Context(), req.Id)
	if err != nil {
		t.log.Error("Error getting balance:", "error", err, "userId", req.Id)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to get orders",
		})
		return
	}

	// 4. Формируем ответ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transport.GetOrdersResponse{
		Orders: orders,
	})
}

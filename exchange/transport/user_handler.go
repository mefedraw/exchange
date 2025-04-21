package handler

import (
	"Exchange/internal/domain/models"
	"Exchange/internal/domain/models/transport"
	"Exchange/internal/services/order"
	"Exchange/internal/services/user"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

type UserHandler struct {
	log         *slog.Logger
	userService userService
	validate    *validator.Validate
}

type userService interface {
	RegisterNewUser(ctx context.Context, email string, password string) (int64, error)
	GetBalance(ctx context.Context, id int64) (decimal.Decimal, error)
	IncreaseBalance(ctx context.Context, id int64, increaseAmount decimal.Decimal) (decimal.Decimal, error)
	DecreaseBalance(ctx context.Context, id int64, decreaseAmount decimal.Decimal) (decimal.Decimal, error)
	GetUserOrders(ctx context.Context, id int64) ([]models.Order, error)
	Login(ctx context.Context, email, password string) (int64, string, error)
}

func NewUserHandler(log *slog.Logger, userService userService, validate *validator.Validate) *UserHandler {
	return &UserHandler{
		log:         log,
		userService: userService,
		validate:    validate,
	}
}

func (h *UserHandler) Routes() chi.Router {
	router := chi.NewRouter()

	// router.Use(middleware.Logger(h.log))
	router.Use(middleware.Recoverer)

	router.Route("/api/user", func(router chi.Router) {
		router.Post("/register", h.PostRegister)
		router.Post("/login", h.PostLogin)

		router.Group(func(routerWithAuth chi.Router) {
			// routerWithAuth.Use(h.authMiddleware) // middleware для аутентификации

			routerWithAuth.Post("/balance", h.GetBalance)
			routerWithAuth.Post("/balance/increase", h.PostIncreaseBalance)
			routerWithAuth.Post("/balance/decrease", h.PostDecreaseBalance)
		})
	})

	return router
}

// PostRegister обрабатывает регистрацию нового пользователя
func (h *UserHandler) PostRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var regReq transport.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
		h.log.Error("Error decoding register request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to decode request body",
		})
		return
	}

	if err := h.validate.Struct(&regReq); err != nil {
		h.log.Error("Error validating register request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid email or password format",
		})
		return
	}

	userID, err := h.userService.RegisterNewUser(r.Context(), regReq.Email, regReq.Password)
	if err != nil {
		h.log.Error("Error registering user:", err)

		if errors.Is(err, user.ErrUserAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(transport.ErrorResponse{
				Error: "User already exists",
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to register user",
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		ID int64 `json:"id"`
	}{
		ID: userID,
	})
}

func (h *UserHandler) PostLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var loginReq transport.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		h.log.Error("Error decoding login request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to decode request body",
		})
		return
	}

	if err := h.validate.Struct(&loginReq); err != nil {
		h.log.Error("Error validating login request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid email or password format",
		})
		return
	}

	userID, email, err := h.userService.Login(r.Context(), loginReq.Email, loginReq.Password)
	if err != nil {
		h.log.Error("Error logging in user:", err)

		if errors.Is(err, user.ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(transport.ErrorResponse{
				Error: "Invalid email or password",
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to login user",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}{
		ID:    userID,
		Email: email,
	})
}

func (h *UserHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Парсим запрос
	var req transport.BalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Error decoding balance request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to decode request body",
		})
		return
	}

	// 2. Валидируем запрос
	if err := h.validate.Struct(&req); err != nil {
		h.log.Error("Error validating balance request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid request format",
		})
		return
	}

	// 3. Получаем баланс
	balance, err := h.userService.GetBalance(r.Context(), req.UserID)
	if err != nil {
		h.log.Error("Error getting balance:", "error", err, "userId", req.UserID)

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to get balance",
		})
		return
	}

	// 4. Формируем ответ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transport.BalanceResponse{
		UserID:  req.UserID,
		Balance: balance,
	})
}

// PostIncreaseBalance increases user's balance
func (h *UserHandler) PostIncreaseBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req transport.IncreaseBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode request", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		err := json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Invalid request format",
		})
		if err != nil {
			return
		}
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.log.Error("Validation failed", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Validation failed: userId and amount (positive) are required",
		})
		return
	}

	newBalance, err := h.userService.IncreaseBalance(r.Context(), req.Id, req.Amount)
	if err != nil {
		h.log.Error("Balance increase failed", "error", err, "userId", req.Id)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to increase balance",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transport.BalanceResponse{
		UserID:  req.Id,
		Balance: newBalance,
	})
}

// PostDecreaseBalance decreases user's balance
func (h *UserHandler) PostDecreaseBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req transport.DecreaseBalanceRequest
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
			Error: "Validation failed: userId and amount (positive) are required",
		})
		return
	}

	newBalance, err := h.userService.DecreaseBalance(r.Context(), req.Id, req.Amount)
	if err != nil {
		h.log.Error("Balance decrease failed", "error", err, "userId", req.Id)

		if errors.Is(err, order.ErrInsufficientFunds) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(transport.ErrorResponse{
				Error: "Insufficient funds",
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(transport.ErrorResponse{
			Error: "Failed to decrease balance",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transport.BalanceResponse{
		UserID:  req.Id,
		Balance: newBalance,
	})
}

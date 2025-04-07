package postgres

import (
	"Exchange/internal/domain/models"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"log/slog"
	"time"
)

const (
	uniqueViolation = "23505"
)

var (
	ErrUserAlreadyExists    = errors.New("User already exists")
	ErrTradingPairNotExists = errors.New("Trading pair does not exist")
)

type Storage struct {
	db *pgxpool.Pool
}

func New(connString string) (*Storage, error) {
	const op = "postgresql.New"
	log := slog.With("op", op)
	db, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Error("Failed to connect to database", "err", err, "connString", connString)
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	err = db.Ping(context.Background())
	if err != nil {
		log.Error("Failed to ping database", "err", err, "connString", connString)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) CreateUser(ctx context.Context,
	email string,
	passHash []byte,
	balance decimal.Decimal,
	createdAt time.Time) (int64, error) {
	const op = "postgresql.CreateUser"
	log := slog.With("op", op)

	const queryCreateUser = "INSERT INTO users(email, pass_hash,balance, created) VALUES ($1, $2, $3, $4) RETURNING id"
	var userId int64
	err := s.db.QueryRow(ctx, queryCreateUser, email, passHash, balance, createdAt).Scan(&userId)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			log.Error("User already exists", "email", email)
			return 0, ErrUserAlreadyExists
		}
		log.Error("Failed to create user", "email", email, "err", err)
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return userId, nil
}

func (s *Storage) GetBalance(ctx context.Context, id int64) (decimal.Decimal, error) {
	const op = "postgresql.GetBalance"
	log := slog.With("op", op)

	const queryGetBalance = `SELECT balance FROM users WHERE id = $1`
	var balance decimal.Decimal
	err := s.db.QueryRow(ctx, queryGetBalance, id).Scan(&balance)
	if err != nil {
		log.Error("Failed to get balance", "id", id, "err", err)
		return decimal.Decimal{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Got balance", "id", id, "balance", balance)
	return balance, nil
}

func (s *Storage) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	const op = "postgresql.GetUserByEmail"
	log := slog.With("op", op)
	const queryGetUserByEmail = `SELECT * FROM users WHERE email = $1`
	var user models.User
	err := s.db.QueryRow(ctx, queryGetUserByEmail, email).Scan(&user)
	if err != nil {
		log.Error("Failed to get user", "email", email, "err", err)
		return user, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully get user", "email", email)
	return user, nil
}

func (s *Storage) GetUserById(ctx context.Context, id int64) (models.User, error) {
	const op = "postgresql.GetUserById"
	log := slog.With("op", op)
	const queryGetUserById = `SELECT * FROM users WHERE id = $1`
	var user models.User
	err := s.db.QueryRow(ctx, queryGetUserById, id).Scan(&user)
	if err != nil {
		log.Error("Failed to get user", "id", id, "err", err)
		return user, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully get user", "id", id)
	return user, nil
}

func (s *Storage) IncreaseBalance(ctx context.Context, id int64, increaseAmount decimal.Decimal) (decimal.Decimal, error) {
	const op = "postgresql.IncreaseBalance"
	log := slog.With("op", op)

	const queryIncreaseBalance = "UPDATE users SET balance = balance + $1 WHERE id = $2 RETURNING balance"

	var updatedBalance decimal.Decimal
	err := s.db.QueryRow(ctx, queryIncreaseBalance, increaseAmount, id).Scan(&updatedBalance)
	if err != nil {
		log.Error("Failed to increase balance", "id", id, "err", err)
		return decimal.Zero, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("balance successfully increased", "id", id, "balance", increaseAmount)
	return updatedBalance, nil
}

func (s *Storage) DecreaseBalance(ctx context.Context, id int64, decreaseAmount decimal.Decimal) (decimal.Decimal, error) {
	const op = "postgresql.DecreaseBalance"
	log := slog.With("op", op)

	const queryIncreaseBalance = "UPDATE users SET balance = balance - $1 WHERE id = $2 RETURNING balance"

	var updatedBalance decimal.Decimal
	err := s.db.QueryRow(ctx, queryIncreaseBalance, decreaseAmount, id).Scan(&updatedBalance)
	if err != nil {
		log.Error("Failed to decrease balance", "id", id, "err", err)
		return decimal.Zero, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Balance successfully decreased", "id", id, "balance", decreaseAmount)
	return updatedBalance, nil
}

func (s *Storage) CreateOrder(ctx context.Context,
	id uuid.UUID,
	userId int64,
	pairId int64,
	orderType models.OrderType,
	margin decimal.Decimal,
	leverage uint8,
	entryPrice decimal.Decimal,
	status models.OrderStatus,
	createdAt time.Time) (uuid.UUID, error) {

	const op = "postgresql.CreateOrder"
	log := slog.With("op", op)

	const queryCreateOrder = "INSERT INTO orders(id, user_id, pair_id, type, margin, leverage, entry_price, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id"

	var orderId uuid.UUID
	err := s.db.QueryRow(ctx, queryCreateOrder, id, userId, pairId, orderType, margin, leverage, entryPrice, status, createdAt).Scan(&orderId)
	if err != nil {
		log.Error("Failed to create order", "id", id, "user_id", userId, "pair_id", pairId, "err", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Order successfully created", "id", id, "user_id", userId, "pair_id", pairId)
	return orderId, nil
}

func (s *Storage) GetOrder(ctx context.Context, id uuid.UUID) (models.Order, error) {
	const op = "postgresql.GetOrder"
	log := slog.With("op", op)
	const queryGetOrder = `SELECT * FROM orders WHERE id = $1`
	var order models.Order
	err := s.db.QueryRow(ctx, queryGetOrder, id).Scan(&order)
	if err != nil {
		log.Error("Failed to get order", "id", id, "err", err)
		return order, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully get order", "id", id)
	return order, nil
}

func (s *Storage) GetUserOrders(ctx context.Context, userId int64) ([]models.Order, error) {
	const op = "postgresql.GetUserOrders"
	log := slog.With("op", op)
	const queryGetUserOrders = `SELECT * FROM orders WHERE user_id = $1`
	var orders []models.Order
	rows, err := s.db.Query(ctx, queryGetUserOrders, userId)
	if err != nil {
		log.Error("Failed to get user orders", "user_id", userId, "err", err)
		return orders, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order)
		if err != nil {
			log.Error("Failed to scan user order", "user_id", userId, "err", err)
			return orders, fmt.Errorf("%s: %w", op, err)
		}
		orders = append(orders, order)
	}

	log.Info("Successfully get user orders", "user_id", userId)
	return orders, nil
}

func (s *Storage) OpenOrder(
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
) (orderID uuid.UUID, err error) {
	const op = "postgresql.OpenOrder"
	log := slog.With("op", op)

	// Начинаем транзакцию
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "err", err)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// 1. Создаем ордер
	const queryCreateOrder = `
        INSERT INTO orders(id, user_id, pair_id, type, margin, leverage, 
                          entry_price, status, created_at)
        VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id`

	err = tx.QueryRow(ctx, queryCreateOrder,
		id, userId, pairId, orderType, margin,
		leverage, entryPrice, status, createdAt,
	).Scan(&orderID)
	if err != nil {
		log.Error("Failed to open order", "err", err)
		return uuid.Nil, fmt.Errorf("%s: create order: %w", op, err)
	}

	// 2. Списываем средства
	const queryDecreaseBalance = `
        UPDATE users 
        SET balance = balance - $1 
        WHERE id = $2 
        RETURNING balance`

	var newBalance decimal.Decimal
	err = tx.QueryRow(ctx, queryDecreaseBalance, margin, userId).Scan(&newBalance)
	if err != nil {
		log.Error("Failed to decrease balance", "err", err)
		return uuid.Nil, fmt.Errorf("%s: decrease balance: %w", op, err)
	}

	// Проверяем, что баланс не ушел в минус
	if newBalance.IsNegative() {
		log.Error("Insufficient funds", "user_id", userId, "balance", newBalance)
		return uuid.Nil, fmt.Errorf("%s: insufficient funds", op)
	}

	// 3. Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "err", err)
		return uuid.Nil, fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	log.Info("Transaction completed successfully",
		"order_id", orderID,
		"user_id", userId,
		"new_balance", newBalance)
	return orderID, nil
}

// CloseOrder sets order status to 'closed' and change order owner funds
func (s *Storage) CloseOrder(
	ctx context.Context,
	orderID uuid.UUID,
	closePrice decimal.Decimal,
	balanceIncrease decimal.Decimal,
) (uuid.UUID, error) {
	const op = "postgresql.CloseOrder"
	log := slog.With("op", op, "order_id", orderID)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "err", err)
		return uuid.Nil, fmt.Errorf("%s: begin transaction: %w", op, err)
	}
	defer tx.Rollback(ctx)

	// 1. Получаем данные ордера и блокируем его для изменения
	var (
		userID int64
		status models.OrderStatus
	)
	err = tx.QueryRow(ctx, `
        SELECT user_id, status 
        FROM orders 
        WHERE id = $1 
        FOR UPDATE`, // Блокировка ордера
		orderID,
	).Scan(&userID, &status)

	if errors.Is(err, pgx.ErrNoRows) {
		log.Error("Order not found")
		return uuid.Nil, fmt.Errorf("%s: order not found", op)
	}
	if err != nil {
		log.Error("Failed to get order", "err", err)
		return uuid.Nil, fmt.Errorf("%s: get order: %w", op, err)
	}

	// 2. Проверяем, что ордер можно закрыть
	if status != models.Open {
		log.Error("Order is not open", "status", status)
		return uuid.Nil, fmt.Errorf("%s: order is not open", op)
	}

	// 3. Обновляем ордер (закрываем)
	_, err = tx.Exec(ctx, `
        UPDATE orders 
        SET 
            status = $1,
            close_price = $2
        WHERE id = $3`,
		models.Closed,
		closePrice,
		orderID,
	)
	if err != nil {
		log.Error("Failed to close order", "err", err)
		return uuid.Nil, fmt.Errorf("%s: close order: %w", op, err)
	}

	// 4. Увеличиваем баланс пользователя на указанную сумму
	var newBalance decimal.Decimal
	err = tx.QueryRow(ctx, `
        UPDATE users 
        SET balance = balance + $1 
        WHERE id = $2 
        RETURNING balance`,
		balanceIncrease,
		userID,
	).Scan(&newBalance)
	if err != nil {
		log.Error("Failed to increase user balance", "user_id", userID, "err", err)
		return uuid.Nil, fmt.Errorf("%s: increase balance: %w", op, err)
	}

	// 5. Фиксируем транзакцию
	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "err", err)
		return uuid.Nil, fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	log.Info("Order successfully closed",
		"user_id", userID,
		"balance_increase", balanceIncrease,
		"new_balance", newBalance)
	return orderID, nil
}

func (s *Storage) AddTradingPair(baseAsset, quoteAsset string) (int64, error) {
	const op = "postgresql.AddTradingPair"
	log := slog.With("op", op)

	const queryAddTradingPair = "INSERT INTO trading_pairs(base_asset, quote_asset) VALUES ($1, $2) RETURNING id"
	var id int64
	err := s.db.QueryRow(context.Background(), queryAddTradingPair, baseAsset, quoteAsset).Scan(&id)
	if err != nil {
		log.Error("Failed to add trading pair", "err", err)
		return 0, fmt.Errorf("%s: add trading pair: %w", op, err)
	}

	ticker := baseAsset + "/" + quoteAsset
	log.Info("Added trading pair", "id", id, "ticker", ticker)
	return id, nil
}

func (s *Storage) GetTradingPairId(baseAsset, quoteAsset string) (int64, error) {
	const op = "postgresql.GetTradingPairId"
	log := slog.With("op", op)

	const queryGetTradingPairId = "SELECT id from trading_pairs WHERE base_asset = $1 AND quote_asset = $2"
	var id int64
	err := s.db.QueryRow(context.Background(), queryGetTradingPairId, baseAsset, quoteAsset).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Error("Trading Pair does not exist", "base_asset", baseAsset, "quote_asset", quoteAsset)
			return 0, fmt.Errorf("%s: get trading pair id: %w", op, ErrTradingPairNotExists)
		}
		log.Error("Failed to get trading pair id", "err", err)
		return 0, fmt.Errorf("%s: get trading pair id: %w", op, err)
	}

	return id, nil
}

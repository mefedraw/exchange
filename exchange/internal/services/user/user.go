package user

import (
	"Exchange/internal/domain/models"
	"Exchange/internal/storage/postgres"
	"context"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"time"
)

var (
	ErrUserAlreadyExists  = errors.New("User already exists")
	ErrInsufficientFunds  = errors.New("Insufficient funds")
	ErrInvalidAmount      = errors.New("Invalid amount")
	ErrInvalidCredentials = errors.New("Invalid credentials")
)

type UserService struct {
	log            slog.Logger
	manager        Manager
	balanceManager BalanceManager
}

func (us *UserService) GetUserOrders(ctx context.Context, id int64) ([]models.Order, error) {
	//TODO implement me
	panic("implement me")
}

type Manager interface {
	CreateUser(ctx context.Context,
		email string,
		passHash []byte,
		balance decimal.Decimal,
		createdAt time.Time) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	GetUserById(ctx context.Context, id int64) (models.User, error)
}

type BalanceManager interface {
	GetBalance(ctx context.Context, id int64) (decimal.Decimal, error)
	IncreaseBalance(ctx context.Context, id int64, increaseAmount decimal.Decimal) (decimal.Decimal, error)
	DecreaseBalance(ctx context.Context, id int64, decreaseAmount decimal.Decimal) (decimal.Decimal, error)
}

func New(log slog.Logger, manager Manager, balanceManager BalanceManager) *UserService {
	return &UserService{
		log:            log,
		manager:        manager,
		balanceManager: balanceManager,
	}
}

func (us *UserService) RegisterNewUser(ctx context.Context, email string, password string) (int64, error) {
	const op = "user.RegisterNewUser"

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		us.log.Error("Failed to generate password hash", "err", err)
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := us.manager.CreateUser(ctx, email, passHash, decimal.Zero, time.Now())
	if err != nil {
		if errors.Is(err, postgres.ErrUserAlreadyExists) {
			us.log.Error("Failed to register already exists user", "email", email)
			return 0, ErrUserAlreadyExists
		}
		us.log.Error("Failed to register user", "email", email, "err", err)
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (us *UserService) Login(ctx context.Context, email, password string) (int64, string, error) {
	const op = "user.Login"

	user, err := us.manager.GetUserByEmail(ctx, email)
	if err != nil {
		slog.Error("Failed to get user by email", "email", email, "err", err, "op", op)
		return 0, "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
		slog.Error("invalid credentials", slog.String("error", err.Error()))

		return 0, "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	return user.Id, user.Email, nil
}

/*
func (a *Auth) Login(
	ctx context.Context,
	email, password string,
	appID int) (string, error) {
	const op = "auth.Login"

	log := a.log.With(slog.String("op", op),
		slog.String("email", email))

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", slog.String("error", err.Error()))

			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		log.Error("failed to get user", slog.String("error", err.Error()))
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Error("invalid credentials", slog.String("error", err.Error()))

		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := a.appProvider.App(ctx, int64(appID))
	if err != nil {
		log.Error("failed to get app", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	token, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		log.Error("failed to create token", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}
*/

func (us *UserService) GetBalance(ctx context.Context, id int64) (decimal.Decimal, error) {
	const op = "user.GetBalance"

	balance, err := us.balanceManager.GetBalance(ctx, id)
	if err != nil {
		us.log.Error("Failed to get balance", "id", id, "err", err)
		return decimal.Zero, fmt.Errorf("%s: %w", op, err)
	}

	return balance, nil
}

func (us *UserService) IncreaseBalance(ctx context.Context, id int64, increaseAmount decimal.Decimal) (decimal.Decimal, error) {
	const op = "user.IncreaseBalance"

	if increaseAmount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, ErrInvalidAmount
	}

	updatedBalance, err := us.balanceManager.IncreaseBalance(ctx, id, increaseAmount)
	if err != nil {
		us.log.Error("Failed to increase balance", "id", id, "err", err)
		return decimal.Zero, fmt.Errorf("%s: %w", op, err)
	}

	return updatedBalance, nil
}

func (us *UserService) DecreaseBalance(ctx context.Context, id int64, decreaseAmount decimal.Decimal) (decimal.Decimal, error) {
	const op = "user.DecreaseBalance"

	if decreaseAmount.LessThanOrEqual(decimal.Zero) {
		us.log.Error("Decrease amount below zero", "id", id, "amount", decreaseAmount)
		return decimal.Zero, ErrInvalidAmount
	}

	currentBalance, err := us.balanceManager.GetBalance(ctx, id)
	if err != nil {
		us.log.Error("Failed to get balance", "id", id, "err", err)
		return decimal.Zero, fmt.Errorf("%s: %w", op, err)
	}

	if decreaseAmount.GreaterThan(currentBalance) {
		us.log.Error("Insufficient funds", "id", id, "currentBalance", currentBalance)
		return decimal.Zero, ErrInsufficientFunds
	}

	updatedBalance, err := us.balanceManager.DecreaseBalance(ctx, id, decreaseAmount)
	if err != nil {
		us.log.Error("Failed to decrease balance", "id", id, "err", err)
		return decimal.Zero, fmt.Errorf("%s: %w", op, err)
	}

	return updatedBalance, nil
}

package redis

import (
	"Exchange/internal/config"
	"Exchange/internal/domain/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const prefix = "exchange:binance:price"

type Redis struct {
	client *redis.Client
}

func New(redisConfig config.RedisConfig) *Redis {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Host + ":" + strconv.Itoa(redisConfig.Port),
		Password: redisConfig.Password,
		DB:       redisConfig.Db,
	})

	return &Redis{
		client: redisClient,
	}
}

func (s *Redis) SaveOrder(ctx context.Context, order models.Order) error {
	const method = "SaveOrder"
	log := slog.With("method", method)
	parsedLiqPrice, err := strconv.ParseFloat(order.LiquidationPrice.String(), 64)
	if err != nil {
		log.Error("failed to parse liquidation price", "err", err)
		return fmt.Errorf("liq price parse err: %s:%w", "err", err)
	}

	orderPrefix := "orders:"
	if models.Long == order.Type {
		orderPrefix += "long"
	} else if models.Short == order.Type {
		orderPrefix += "short"
	}
	orderPrefix += order.Ticker

	s.client.ZAdd(ctx, orderPrefix, &redis.Z{
		Score: parsedLiqPrice, Member: order.Id,
	})
	log.Info("saved order to redis-sorted-set", "id", order.Id)
	return nil
}

func (s *Redis) RemoveOrder(ctx context.Context, id string) error {
	const method = "RemoveOrder"
	log := slog.With("method", method)
	s.client.ZRem(ctx, prefix+id, &redis.Z{
		Member: id,
	})
	log.Info("removed order from redis-sorted-set", "id", id)
	return nil
}

func (s *Redis) SavePrices(ctx context.Context, prices []models.PriceResponse) error {
	log := slog.With("method", "SavePrices")
	pipe := s.client.Pipeline()

	for _, priceResp := range prices {
		key := fmt.Sprintf("%s:%s", prefix, priceResp.Symbol)
		value, _ := json.Marshal(priceResp.Price)
		pipe.Set(ctx, key, value, 10*time.Minute)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Error("failed to save prices", "err", err)
		return fmt.Errorf("failed to save prices: %w", err)
	}

	// log.Info("saved prices successfully")
	return nil
}

func (s *Redis) GetLiqOrders(ctx context.Context, ticker, price string) ([]uuid.UUID, error) {
	const method = "GetLiqOrders"
	log := slog.With("method", method)
	parsedLiqPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Error("failed to parse liq price", "err", err)
		return nil, fmt.Errorf("%s:%w", method, err)
	}

	const longPrefix = "orders:long:"
	const shortPrefix = "orders:short:"
	const minInf = "-inf"
	const maxInf = "+inf"
	maxScore := strconv.FormatFloat(parsedLiqPrice, 'f', -1, 64)
	longOrders, err := s.client.ZRangeByScore(ctx, longPrefix+ticker, &redis.ZRangeBy{
		Min: minInf, Max: maxScore,
	}).Result()
	if err != nil {
		log.Error("failed to get long orders for liq by Zrange", "err", err)
		return nil, fmt.Errorf("%s:%s:%w", method, "long", err)
	}

	minScore := maxScore
	shortOrders, err := s.client.ZRangeByScore(ctx, shortPrefix+ticker, &redis.ZRangeBy{
		Min: minScore, Max: maxInf,
	}).Result()

	allOrders := append(longOrders, shortOrders...)
	result := make([]uuid.UUID, 0, len(allOrders))
	for _, idStr := range allOrders {
		id, err := uuid.Parse(idStr)
		if err != nil {
			log.Error("failed to parse UUID", "id", idStr, "err", err)
			continue
		}
		result = append(result, id)
	}

	return result, nil
}

func (s *Redis) GetAllPrices(ctx context.Context) ([]models.PriceResponse, error) {
	log := slog.With("method", "GetAllPrices")
	data, err := s.client.HGetAll(ctx, prefix).Result()
	if err != nil {
		log.Error("failed to get prices", "err", err)
		return nil, fmt.Errorf("failed to get all prices: %w", err)
	}

	prices := make([]models.PriceResponse, 0, len(data))
	for _, jsonData := range data {
		var price models.PriceResponse
		if err := json.Unmarshal([]byte(jsonData), &price); err == nil {
			log.Debug("priceResponse", price)
			prices = append(prices, price)
		}
	}
	return prices, nil
}

func (s *Redis) GetPrice(ctx context.Context, ticker string) (string, error) {
	log := slog.With("method", "GetPrice")

	tickerRedis := ticker
	if strings.Contains(ticker, "/") {
		parts := strings.Split(ticker, "/")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid ticker: %s", ticker)
		}
		tickerRedis = parts[0] + parts[1]
		log.Debug("ticker modified", "ticker", ticker)
	}

	data, err := s.client.Get(ctx, prefix+":"+tickerRedis).Result()
	if err != nil {
		log.Error("failed to get price", "err", err)
		return "", fmt.Errorf("failed to get prices: %w", err)
	}
	var price string
	err = json.Unmarshal([]byte(data), &price)
	if err != nil {
		log.Error("failed to unmarshal price", "data", data, "err", err)
		return "", fmt.Errorf("failed to unmarshal prices: %w", err)
	}

	log.Debug("successfully get price from redis", "price", price)
	return price, nil
}

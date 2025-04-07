package redis

import (
	"Exchange/internal/config"
	"Exchange/internal/domain/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log/slog"
	"strconv"
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

	log.Info("saved prices successfully")
	return nil

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
	data, err := s.client.Get(ctx, prefix+":"+ticker).Result()
	if err != nil {
		log.Error("failed to get prices", "err", err)
		return "", fmt.Errorf("failed to get prices: %w", err)
	}
	var price string
	err = json.Unmarshal([]byte(data), &price)
	if err != nil {
		log.Error("failed to unmarshal price", "data", data, "err", err)
		return "", fmt.Errorf("failed to unmarshal prices: %w", err)
	}

	return price, nil
}

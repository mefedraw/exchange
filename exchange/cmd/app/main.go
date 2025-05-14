package main

import (
	"Exchange/internal/config"
	"Exchange/internal/domain/models"
	"Exchange/internal/http_client"
	"Exchange/internal/liquidation"
	"Exchange/internal/services/order"
	"Exchange/internal/services/trade"
	user "Exchange/internal/services/user"
	"Exchange/internal/storage/postgres"
	"Exchange/internal/storage/redis"
	handler "Exchange/transport"
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/nats-io/nats.go"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	envDevMac = "dev_mac"
	envDevWin = "dev_win"
	envProd   = "prod"
)

func main() {
	// TODO: init config

	cfg := config.MustLoad()

	// TODO: init logger

	log := setupLogger(cfg.Env)

	log.Info("starting application",
		slog.String("env", cfg.Env),
		slog.Any("cfg", cfg),
	)

	// TODO: переделать по уму коннект к бд
	var connString string
	if cfg.Env == envDevMac {
		log.Info("connecting to postgres via dev_mac")
		connString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.PostgresCfgMac.Username,
			cfg.PostgresCfgMac.Password,
			cfg.PostgresCfgMac.Host,
			cfg.PostgresCfgMac.Port,
			cfg.PostgresCfgMac.Database)
	} else if cfg.Env == envDevWin {
		log.Info("connecting to postgres via dev_win")
		connString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.PostgresCfgWin.Username,
			cfg.PostgresCfgWin.Password,
			cfg.PostgresCfgWin.Host,
			cfg.PostgresCfgWin.Port,
			cfg.PostgresCfgWin.Database)
	}

	storage, err := postgres.New(connString)
	if err != nil {
		log.Error("failed to connect to postgres")
	}

	priceClient := http_client.New(*cfg, *log)
	redisClient := redis.New(cfg.RedisCfg)

	// TODO: init NATS CONN
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Error("failed to connect to nats", "error", err)
		panic(err)
	}
	log.Info("connected to nats broker", "url", nats.DefaultURL)
	js, err := nc.JetStream()
	if err != nil {
		log.Error("failed to connect to nats", "error", err)
		panic(err)
	}
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "PRICES-STREAM",
		Subjects: []string{"prices.*"},
	})
	if err != nil {
		log.Error("failed to connect to nats", "error", err)
		panic(err)
	}

	ctx := context.Background()
	// todo: GET PRICES LOOP
	go func() {
		for {
			prices, _ := priceClient.GetPrice()
			_ = redisClient.SavePrices(ctx, prices)
			const topicPart = "prices."
			for _, priceResp := range prices {
				// todo: get it from cfg
				topic := topicPart + priceResp.Symbol
				_, err = js.Publish(topic, []byte(priceResp.Price))
				if err != nil {
					slog.Error("failed to publish price", "topic", topic, "priceResp", priceResp, "err", err)
				}
			}
			const testTicker = "TESTUSDT"
			testPrice, _ := redisClient.GetPrice(ctx, testTicker)
			testPriceResp := models.PriceResponse{
				Symbol: testTicker,
				Price:  testPrice,
			}
			_, err = js.Publish(topicPart+testTicker, []byte(testPriceResp.Price))
			if err != nil {
				slog.Error("failed to publish price", "topic", topicPart+testTicker, "priceResp", testPriceResp, "err", err)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// TODO: init chi router
	validate := validator.New()

	userService := user.New(*log, storage, storage)
	orderService := order.New(*log, storage, storage, storage)
	tradeService := trade.New(log, *orderService, *redisClient)

	// TODO: init Liquidator
	liquidator, err := liquidation.NewLiquidator(nc, orderService)
	liquidator.Process()

	userHandler := handler.NewUserHandler(log, userService, validate)
	tradeHandler := handler.NewTradeHandler(log, tradeService, validate)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "300")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	})
	r.Mount("/user", userHandler.Routes())
	r.Mount("/trade", tradeHandler.Routes())

	port := ":8080"
	log.Info("Starting server on " + port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Error("Server failed", "error", err)
	}

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envDevMac:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDevWin:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

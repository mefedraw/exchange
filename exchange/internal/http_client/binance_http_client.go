package http_client

import (
	"Exchange/internal/config"
	"Exchange/internal/domain/models"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type BinanceHTTPClient struct {
	baseURL  string
	endpoint string
	streams  []string
	log      slog.Logger
	client   *http.Client
}

func New(cfg config.Config, log slog.Logger) *BinanceHTTPClient {
	return &BinanceHTTPClient{
		baseURL:  cfg.BinanceConfig.BaseURL,
		endpoint: cfg.BinanceConfig.Endpoint,
		streams:  cfg.BinanceConfig.Streams,
		log:      log,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (pr *BinanceHTTPClient) GetPrice() ([]models.PriceResponse, error) {
	log := pr.log.With("method", "GetPrice")

	reqUrl := fmt.Sprintf("%s%s%s", pr.baseURL, pr.endpoint, pr.addParamsToUrl())

	// log.Debug("making request to Binance API", "url", reqUrl)
	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Error("failed to create request", "error", err)
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	resp, err := pr.client.Do(req)
	if err != nil {
		log.Error("failed to make request", "error", err)
		return nil, fmt.Errorf("could not make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error("unexpected status code",
			"status", resp.StatusCode,
			"response", string(body))
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	priceResp := []models.PriceResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		log.Error("failed to decode response", "error", err)
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// log.Debug("successfully received price")

	return priceResp, nil
}

func (pr *BinanceHTTPClient) addParamsToUrl() string {
	params := "?symbols=["
	for i, stream := range pr.streams {
		params = fmt.Sprintf("%s\"%s\"", params, stream)
		if i != len(pr.streams)-1 {
			params = fmt.Sprintf("%s,", params)
		}
	}
	params = fmt.Sprintf("%s]", params)

	return params
}

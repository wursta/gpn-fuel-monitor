package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// StationResponse — ответ API: {"data": [...]}
type StationResponse struct {
	Data []Fuel `json:"data"`
}

// Fuel — элемент массива data в ответе API.
type Fuel struct {
	ID      int     `json:"id"`
	Product Product `json:"product"`
	Price   Price   `json:"price"`
	Rest    Rest    `json:"rest"`
}

// Product — информация о продукте (виде топлива).
type Product struct {
	Title      string `json:"title"`
	ShortTitle string `json:"shortTitle"`
}

// Price — информация о цене.
type Price struct {
	Currency string  `json:"currency"`
	Price    float64 `json:"price"`
	Since    string  `json:"since"`
}

// Rest — информация о наличии и доставке.
type Rest struct {
	Avail    bool   `json:"avail"`
	Since    string `json:"since"`
	Delivery string `json:"delivery"`
}

// FetchStationStatus выполняет GET-запрос к API и возвращает список топлив.
// Использует контекст по умолчанию (без таймаута отмены).
func FetchStationStatus(cfg *Config) ([]Fuel, error) {
	return fetchStationStatusWithContext(context.Background(), cfg)
}

// fetchStationStatusWithContext выполняет GET-запрос к API с поддержкой контекста.
func fetchStationStatusWithContext(ctx context.Context, cfg *Config) ([]Fuel, error) {
	url := fmt.Sprintf("https://gpnbonus.ru/api/stations/%d", cfg.StationID)

	client := newHTTPClient(cfg)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Устанавливаем заголовки из конфига
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var stationResp StationResponse
	if err := json.Unmarshal(body, &stationResp); err != nil {
		return nil, fmt.Errorf("parse JSON response: %w", err)
	}

	return stationResp.Data, nil
}

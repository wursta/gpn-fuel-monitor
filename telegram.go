package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// telegramMessage — тело запроса к Telegram Bot API.
type telegramMessage struct {
	ChatID  string `json:"chat_id"`
	Text    string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendTelegramNotification отправляет сообщение в Telegram через прокси.
// Возвращает ошибку при неудаче.
func SendTelegramNotification(cfg *Config, token, chatID, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	body := telegramMessage{
		ChatID:    chatID,
		Text:      message,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal telegram message: %w", err)
	}

	client := newHTTPClient(cfg)
	resp, err := client.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP request to Telegram API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Telegram API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Проверяем ответ Telegram
	var tgResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return fmt.Errorf("parse telegram response: %w", err)
	}
	if !tgResp.OK {
		return fmt.Errorf("telegram API returned error: %s", tgResp.Description)
	}

	return nil
}

// SendWithRetry отправляет сообщение с повторными попытками.
// maxRetries — максимальное число попыток, delaySec — пауза между ними.
func SendWithRetry(cfg *Config, token, chatID, message string, maxRetries int, delaySec int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("telegram: attempt %d/%d", attempt, maxRetries)

		err := SendTelegramNotification(cfg, token, chatID, message)
		if err == nil {
			log.Printf("telegram: message sent successfully on attempt %d", attempt)
			return nil
		}

		lastErr = err
		log.Printf("telegram: attempt %d failed: %v", attempt, err)

		if attempt < maxRetries {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}
	}

	return fmt.Errorf("telegram: all %d attempts failed, last error: %w", maxRetries, lastErr)
}

// SendSystemNotification отправляет системное сообщение (старт/остановка).
// Использует ускоренный retry: 3 попытки × 5 секунд.
func SendSystemNotification(cfg *Config, token, chatID, message string) error {
	return SendWithRetry(cfg, token, chatID, message, 3, 5)
}

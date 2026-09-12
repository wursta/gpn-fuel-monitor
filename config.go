package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config — структура конфигурации приложения.
type Config struct {
	StationID          int               `yaml:"station_id"`
	Fuels              map[string]int    `yaml:"fuels"`
	CheckInterval      int               `yaml:"check_interval_seconds"`
	TelegramBotToken   string            `yaml:"telegram_bot_token"`
	TelegramChatID     string            `yaml:"telegram_chat_id"`
	Headers            map[string]string `yaml:"headers"`
	StateFile          string            `yaml:"state_file"`
	LogFile            string            `yaml:"log_file"`
	LogRetentionDays   int               `yaml:"log_retention_days"`
	LogMaxSizeMB       int               `yaml:"log_max_size_mb"`
	Proxy              string            `yaml:"proxy"`
}

// LoadConfig загружает конфигурацию из YAML-файла.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %q: %w", path, err)
	}

	// Валидация обязательных полей
	if cfg.StationID <= 0 {
		return nil, fmt.Errorf("invalid station_id: must be positive, got %d", cfg.StationID)
	}
	if len(cfg.Fuels) == 0 {
		return nil, fmt.Errorf("fuels map is empty")
	}
	if cfg.CheckInterval <= 0 {
		return nil, fmt.Errorf("invalid check_interval_seconds: must be positive, got %d", cfg.CheckInterval)
	}

	// Значения по умолчанию
	if cfg.StateFile == "" {
		cfg.StateFile = "state.json"
	}
	if cfg.LogFile == "" {
		cfg.LogFile = "monitor.log"
	}
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	// Дефолтный User-Agent, если не задан
	if _, hasUA := cfg.Headers["User-Agent"]; !hasUA {
		cfg.Headers["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	// Значения по умолчанию для автоочистки логов
	if cfg.LogRetentionDays <= 0 {
		cfg.LogRetentionDays = 7
	}
	if cfg.LogMaxSizeMB <= 0 {
		cfg.LogMaxSizeMB = 10
	}

	return &cfg, nil
}

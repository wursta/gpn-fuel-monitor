package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// createLogger создаёт логгер с двойным выводом: в файл и в stdout.
func createLogger(logFile string) (*slog.Logger, error) {
	// Открываем файл лога
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", logFile, err)
	}

	// MultiWriter для вывода в файл и stdout
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Создаём handler с форматированием и logger
	logger := slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return logger, nil
}

// setupSignalHandler настраивает обработку сигналов SIGINT/SIGTERM.
// Возвращает context, который отменяется при получении сигнала.
func setupSignalHandler() (context.Context, context.CancelFunc, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	stopChan := make(chan struct{})
	go func() {
		select {
		case sig := <-sigChan:
			slog.Info("received signal, initiating graceful shutdown", "signal", sig)
			cancel()
		case <-stopChan:
		}
	}()

	cleanup := func() {
		close(stopChan)
		signal.Stop(sigChan)
	}

	return ctx, cancel, cleanup
}

// findFuelByID ищет топливо по ID в списке.
func findFuelByID(fuels []Fuel, id int) *Fuel {
	for i := range fuels {
		if fuels[i].ID == id {
			return &fuels[i]
		}
	}
	return nil
}

// checkAndNotify выполняет одну итерацию мониторинга.
func checkAndNotify(ctx context.Context, cfg *Config, state *AppState, logger *slog.Logger) {
	logger.Info("checking station status", "station_id", cfg.StationID)

	// Создаём контекст с таймаутом для запроса
	reqCtx, reqCancel := context.WithTimeout(ctx, 30*time.Second)
	defer reqCancel()

	// Выполняем запрос к API
	fuels, err := fetchStationStatusWithContext(reqCtx, cfg)
	if err != nil {
		logger.Error("failed to fetch station status", "error", err)
		return
	}

	logger.Info("fetched fuels data", "count", len(fuels))

	// Проверяем каждое топливо из конфига
	for fuelName, fuelID := range cfg.Fuels {
		fuel := findFuelByID(fuels, fuelID)
		if fuel == nil {
			logger.Warn("fuel not found in API response", "fuel_name", fuelName, "fuel_id", fuelID)
			continue
		}

		// Определяем текущий статус
		currentStatus := getFuelStatus(*fuel)

		// Формируем новое состояние
		newState := &FuelState{
			Avail:    fuel.Rest.Avail,
			Delivery: fuel.Rest.Delivery,
			Since:    fuel.Rest.Since,
			Status:   currentStatus.Status,
		}

		// Проверяем, изменилось ли состояние
		if StatusChanged(state.Fuels[fuelID], newState) {
			oldStatus := "неизвестно"
			if state.Fuels[fuelID] != nil {
				oldStatus = state.Fuels[fuelID].Status
			}

			message := fmt.Sprintf(
				"АЗС %d: <b>%s</b> — %s (было %s), с %s",
				cfg.StationID,
				fuelName,
				currentStatus.Status,
				oldStatus,
				currentStatus.Since,
			)

			// Отправляем уведомление
			if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
				err := SendWithRetry(
					cfg,
					cfg.TelegramBotToken,
					cfg.TelegramChatID,
					message,
					3, // max retries
					10, // delay in seconds
				)
				if err != nil {
					logger.Error("failed to send telegram notification",
						"fuel", fuelName,
						"error", err,
					)
				}
			}

			logger.Info("status changed",
				"fuel", fuelName,
				"old_status", oldStatus,
				"new_status", currentStatus.Status,
				"since", currentStatus.Since,
			)
		}

		// Обновляем состояние
		state.Fuels[fuelID] = newState
	}

	// Сохраняем состояние
	if err := state.SaveState(cfg.StateFile); err != nil {
		logger.Error("failed to save state", "error", err)
	}
}

func main() {
	// Загружаем конфигурацию
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Настраиваем логгер
	logger, err := createLogger(cfg.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}

	// Автоочистка логов при старте
	if err := EnsureLogCleaned(cfg.LogFile, cfg.LogRetentionDays, cfg.LogMaxSizeMB); err != nil {
		logger.Warn("failed to clean old logs", "error", err)
	}

	logger.Info("starting GPN fuel monitor", "station_id", cfg.StationID)

	// Отправляем приветственное сообщение в Telegram
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		msg := fmt.Sprintf("🟢 Монитор запущен\nАЗС %d", cfg.StationID)
		if err := SendSystemNotification(cfg, cfg.TelegramBotToken, cfg.TelegramChatID, msg); err != nil {
			logger.Warn("failed to send startup notification", "error", err)
		}
	}

	// Загружаем состояние
	state, err := LoadState(cfg.StateFile)
	if err != nil {
		logger.Error("failed to load state, starting fresh", "error", err)
		state = &AppState{
			Fuels: make(map[int]*FuelState),
		}
	}

	// Настраиваем graceful shutdown
	ctx, cancel, cleanup := setupSignalHandler()
	defer cleanup()
	defer cancel()

	// Создаём тикер
	ticker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Second)
	defer ticker.Stop()

	// Первый запуск сразу при старте
	checkAndNotify(ctx, cfg, state, logger)

	// Основной цикл
	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down, saving state...")
			if err := state.SaveState(cfg.StateFile); err != nil {
				logger.Error("failed to save state on shutdown", "error", err)
			}

			// Отправляем сообщение об остановке в Telegram
			if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
				msg := fmt.Sprintf("🔴 Монитор остановлен\nАЗС %d", cfg.StationID)
				if err := SendSystemNotification(cfg, cfg.TelegramBotToken, cfg.TelegramChatID, msg); err != nil {
					logger.Warn("failed to send shutdown notification", "error", err)
				}
			}

			logger.Info("shutdown complete")
			return
		case <-ticker.C:
			checkAndNotify(ctx, cfg, state, logger)
		}
	}
}

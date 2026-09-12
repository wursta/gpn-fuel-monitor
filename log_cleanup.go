package main

import (
	"fmt"
	"os"
	"time"
)

// CleanOldLog проверяет и обрезает log-файл, если он старше retentionDays.
// Если файл старше указанного возраста — обрезает его до пустого состояния.
func CleanOldLog(logFile string, retentionDays int) error {
	info, err := os.Stat(logFile)
	if err != nil {
		// Файл не существует — нечего чистить
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat log file %q: %w", logFile, err)
	}

	age := time.Since(info.ModTime())
	retention := time.Duration(retentionDays) * 24 * time.Hour

	if age > retention {
		// Обрезаем файл до нуля
		if err := os.Truncate(logFile, 0); err != nil {
			return fmt.Errorf("truncate log file %q: %w", logFile, err)
		}
		fmt.Printf("[LOG] очищен старый лог-файл %q (возраст: %.1f дней)\n", logFile, age.Hours()/24)
	}

	return nil
}

// RotateLogIfTooLarge проверяет и ротирует log-файл, если он превышает maxSizeMB.
// При превышении размера файл переименовывается в .old, создаётся новый пустой.
func RotateLogIfTooLarge(logFile string, maxSizeMB int) error {
	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat log file %q: %w", logFile, err)
	}

	maxSize := int64(maxSizeMB) * 1024 * 1024
	if info.Size() <= maxSize {
		return nil
	}

	// Ротация: переименовываем текущий файл
	oldFile := logFile + ".old"
	if err := os.Rename(logFile, oldFile); err != nil {
		return fmt.Errorf("rename log file to %q: %w", oldFile, err)
	}

	// Создаём новый пустой файл
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		// Возвращаем старый файл обратно при ошибке
		os.Rename(oldFile, logFile)
		return fmt.Errorf("create new log file %q: %w", logFile, err)
	}
	f.Close()

	fmt.Printf("[LOG] ротация лога: %q → %q (размер: %.1f МБ)\n",
		logFile, oldFile, float64(info.Size())/(1024*1024))

	return nil
}

// EnsureLogCleaned применяет все проверки к log-файлу.
// Вызывается при старте приложения.
func EnsureLogCleaned(logFile string, retentionDays, maxSizeMB int) error {
	if err := CleanOldLog(logFile, retentionDays); err != nil {
		fmt.Printf("[LOG] warning: failed to clean old log: %v\n", err)
	}

	if err := RotateLogIfTooLarge(logFile, maxSizeMB); err != nil {
		fmt.Printf("[LOG] warning: failed to rotate large log: %v\n", err)
	}

	return nil
}

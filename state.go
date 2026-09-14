package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// FuelState — состояние конкретного топлива.
type FuelState struct {
	Avail    bool   `json:"avail"`
	Delivery string `json:"delivery"`
	Since    string `json:"since"`
	Status   string `json:"status"`
}

// AppState — полное состояние приложения (все топлива).
type AppState struct {
	Fuels map[int]*FuelState `json:"fuels"`
}

// LoadState загружает состояние из JSON-файла.
// Если файл не существует, создаёт пустое состояние.
func LoadState(path string) (*AppState, error) {
	state := &AppState{
		Fuels: make(map[int]*FuelState),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует — возвращаем пустое состояние
			return state, nil
		}
		return nil, fmt.Errorf("read state file %q: %w", path, err)
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse state file %q: %w", path, err)
	}

	if state.Fuels == nil {
		state.Fuels = make(map[int]*FuelState)
	}

	return state, nil
}

// SaveState сохраняет состояние в JSON-файл.
func (s *AppState) SaveState(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write state file %q: %w", path, err)
	}

	return nil
}

// StatusChanged сравнивает два состояния топлива.
// Возвращает true, если изменился кортеж (avail, delivery).
// Поле since игнорируется — это метка актуальности данных, а не часть статуса.
func StatusChanged(old, new *FuelState) bool {
	if old == nil {
		return true // новое топливо — считаем изменением
	}
	return old.Avail != new.Avail || old.Delivery != new.Delivery
}

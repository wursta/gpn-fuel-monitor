# AGENTS.md — GPN Fuel Monitor

## Роль агента

Ты — разработчик Go-приложений. Работай в проекте `gpn-fuel-monitor` — мониторинге наличия бензина на АЗС «Газпромнефть» через внутренний API с уведомлениями в Telegram.

## Стек

- **Язык:** Go 1.21+
- **Конфиг:** `gopkg.in/yaml.v3`
- **Логирование:** `log/slog` (стандартная библиотека)
- **HTTP:** стандартная библиотека (`net/http`)
- **Зависимости:** только `yaml.v3`, всё остальное — stdlib

## Структура проекта

```
gpn-fuel-monitor/
├── main.go             # Точка входа, главный цикл, graceful shutdown
├── config.go           # Загрузка и валидация config.yaml
├── api.go              # GET к gpnbonus.ru, парсинг JSON
├── status.go           # getFuelStatus() → "в наличии"/"нет"/"в пути"/"неизвестно"
├── state.go            # Чтение/запись state.json, StatusChanged()
├── telegram.go         # Telegram Bot API, SendWithRetry(), SendSystemNotification()
├── log_cleanup.go      # Автоочистка логов (по age и size)
├── http_client.go      # newHTTPClient(cfg) — HTTP-клиент с опциональным прокси
├── config.yaml         # Конфигурация
├── go.mod
└── README.md
```

## Архитектура

### Главный цикл (main.go)

```
Загрузка конфига → Логгер → Автоочистка логов → Приветствие в Telegram
  → Загрузка состояния → Тикер → checkAndNotify()
  → Loop: ticker.C → checkAndNotify() / ctx.Done() → Shutdown-сообщение в Telegram
```

### checkAndNotify()

1. GET `https://gpnbonus.ru/api/stations/{station_id}` с заголовками из конфига
2. Парсинг JSON → `[]Fuel`
3. Для каждого топлива из конфига:
   - Найти по ID в ответе API
   - `getFuelStatus(f)` → статус
   - `StatusChanged(old, new)` → сравнение по кортежу `(avail, delivery, since)`
   - Если изменилось → `SendWithRetry(cfg, token, chatID, message, 3, 10)`
   - Обновить `state.Fuels[fuelID]`
4. `state.SaveState(cfg.StateFile)`

### HTTP-клиент (http_client.go)

```go
func newHTTPClient(cfg *Config) *http.Client {
    transport := &http.Transport{}
    if cfg.Proxy != "" {
        proxyURL, _ := url.Parse(cfg.Proxy)
        transport.Proxy = http.ProxyURL(proxyURL)
    }
    return &http.Client{Timeout: 30s, Transport: transport}
}
```

- `proxy: ""` → прямое подключение
- `proxy: "http://..."` → через прокси
- Используется во всех HTTP-запросах: API АЗС и Telegram

### Telegram (telegram.go)

- `SendTelegramNotification(cfg, token, chatID, message)` — один запрос
- `SendWithRetry(cfg, token, chatID, message, retries, delaySec)` — retry-обёртка
- `SendSystemNotification(cfg, token, chatID, message)` — системные сообщения (3×5s)
- Формат сообщения статуса: `АЗС {id}: {fuel_name} — {status} (было {old_status}), с {since}`
- Формат системного сообщения: `🟢 Монитор запущен\nАЗС {id}` / `🔴 Монитор остановлен\nАЗС {id}`

### Логирование (log_cleanup.go)

Вызывается один раз при старте через `EnsureLogCleaned()`:

- `CleanOldLog()` — если файл старше `log_retention_days` → `os.Truncate(logFile, 0)`
- `RotateLogIfTooLarge()` — если файл больше `log_max_size_mb` → rename `.old` + новый файл

### Состояние (state.go)

- `state.json` хранит `map[fuelID]FuelState{avail, delivery, since, status}`
- `StatusChanged(old, new)` — возвращает `true` если изменился `(avail, delivery, since)`
- При старте загружается, если файла нет — пустая мапа

## Правила работы с кодом

1. **Модульность.** Каждый файл отвечает за одну задачу. Не смешивай логику.
2. **Безопасность.** Скрипт не падает при единичных сбоях API — лог, пропустить итерацию.
3. **Контекст.** Все HTTP-запросы принимают `context.Context`.
4. **Прокси опционален.** `cfg.Proxy == ""` → без прокси. Никогда не делай прокси обязательным.
5. **Русский язык.** Сообщения в лог, статусы, Telegram-сообщения — на русском.
6. **Сравнение по кортежу.** Никогда не сравнивай только статус-строку — сравни `(avail, delivery, since)`.
7. **Graceful shutdown.** SIGINT/SIGTERM → сохранить состояние → отправить shutdown-уведомление → exit.

## Конфигурация (config.yaml)

```yaml
station_id: 1109
fuels:
  "АИ-95": 12
  "G-Drive 95": 421
check_interval_seconds: 300
telegram_bot_token: ""
telegram_chat_id: ""
headers:
  User-Agent: "Mozilla/5.0 ..."
state_file: "state.json"
log_file: "monitor.log"
log_retention_days: 7
log_max_size_mb: 10
proxy: ""
```

### ID топлив

| Топливо | ID |
|---------|-----|
| АИ-95 | 12 |
| G-Drive 95 | 421 |

Добавить топливо = добавить строку в `fuels`.

### Логика статуса

| Условия | Статус |
|---------|--------|
| `avail == true` | в наличии |
| `avail == false && delivery == "no"` | нет |
| `avail == false && delivery == "yes"` | в пути |
| иначе | неизвестно |

## Команды

```bash
go mod tidy          # установить зависимости
go build -o monitor . # собрать
./monitor            # запустить
go vet ./...         # статический анализ
```

## Типичные задачи

### Добавить новое топливо
Добавить строку в `fuels` в `config.yaml`. Код менять не нужно.

### Изменить интервал проверки
Изменить `check_interval_seconds` в `config.yaml`.

### Отключить Telegram
Оставить `telegram_bot_token` и `telegram_chat_id` пустыми.

### Изменить размер/перезапуск логгера
Изменить `log_retention_days` и `log_max_size_mb` в `config.yaml`.

### Добавить новый API-эндпоинт
Создать новый файл (например `other_api.go`), использовать `newHTTPClient(cfg)` для запросов.

### Изменить retry-логику
Модифицировать `SendWithRetry()` в `telegram.go`. Все вызовы принимают `cfg *Config`.

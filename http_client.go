package main

import (
	"net/http"
	"net/url"
	"time"
)

// newHTTPClient создаёт HTTP-клиент с поддержкой прокси из конфига.
// Если proxy пустой — используется стандартный клиент без прокси.
func newHTTPClient(cfg *Config) *http.Client {
	transport := &http.Transport{}

	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			// Если прокси невалидный — используем клиент без прокси
			return &http.Client{Timeout: 30 * time.Second}
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

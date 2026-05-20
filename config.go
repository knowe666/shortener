package config

import (
	"flag"
	"fmt"
	"os"
)

// Config хранит все конфигурационные параметры приложения
type Config struct {
	ServerAddress string // адрес запуска HTTP-сервера (флаг -a)
	BaseURL       string // базовый адрес для сокращённых URL (флаг -b)
}

// NewConfig инициализирует конфигурацию из флагов командной строки
func NewConfig() (*Config, error) {
	serverAddress := flag.String("a", "localhost:8080", "адрес запуска HTTP-сервера")
	baseURL := flag.String("b", "http://localhost:8080", "базовый адрес результирующего сокращённого URL")
	flag.Parse()

	if serverAddressEnv := os.Getenv("SERVER_ADDRESS"); serverAddressEnv != "" {
		*serverAddress = serverAddressEnv
	}
	if baseURLEnv := os.Getenv("BASE_URL"); baseURLEnv != "" {
		*baseURL = baseURLEnv
	}

	// Валидация параметров
	if *serverAddress == "" {
		return nil, fmt.Errorf("server address cannot be empty")
	}

	if *baseURL == "" {
		return nil, fmt.Errorf("base URL cannot be empty")
	}

	return &Config{
		ServerAddress: *serverAddress,
		BaseURL:       *baseURL,
	}, nil
}

// GetShortURL формирует полный короткий URL по ID
func (c *Config) GetShortURL(shortID string) string {
	// Убираем trailing slash из BaseURL, если он есть
	baseURL := c.BaseURL
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	return fmt.Sprintf("%s/%s", baseURL, shortID)
}

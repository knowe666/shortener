package config

import (
	"flag"
	"fmt"
	"os"
)

// Config хранит все настройки приложения
type Config struct {
	ServerAddress string // адрес запуска HTTP-сервера (флаг -a)
	BaseURL       string // базовый адрес результирующего сокращённого URL (флаг -b)
}

// NewConfig создаёт новую конфигурацию из аргументов командной строки
func NewConfig() (*Config, error) {
	// Определяем флаги со значениями по умолчанию
	var (
		serverAddr = flag.String("a", "localhost:8080", "адрес запуска HTTP-сервера")
		baseURL    = flag.String("b", "http://localhost:8080", "базовый адрес результирующего сокращённого URL")
	)

	// Разбираем аргументы командной строки
	flag.Parse()

	// Создаём и возвращаем конфиг
	cfg := &Config{
		ServerAddress: *serverAddr,
		BaseURL:       *baseURL,
	}

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// NewConfigWithArgs создаёт конфигурацию с пользовательскими аргументами (для тестирования)
func NewConfigWithArgs(args []string) (*Config, error) {
	// Сохраняем оригинальные аргументы
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Устанавливаем тестовые аргументы
	os.Args = args

	return NewConfig()
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.ServerAddress == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	return nil
}

// String возвращает строковое представление конфигурации
func (c *Config) String() string {
	return fmt.Sprintf("Config{ServerAddress: %s, BaseURL: %s}",
		c.ServerAddress, c.BaseURL)
}

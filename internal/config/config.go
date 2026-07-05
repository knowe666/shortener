package config

import (
	"flag"
	"fmt"
	"os"
)

// Config хранит все конфигурационные параметры приложения
type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string // Добавляем DSN для БД
}

// NewConfig инициализирует конфигурацию из флагов командной строки
func NewConfig() (*Config, error) {
	serverAddress := flag.String("a", "localhost:8080", "адрес запуска HTTP-сервера")
	baseURL := flag.String("b", "http://localhost:8080", "базовый адрес результирующего сокращённого URL")
	filePath := flag.String("f", "", "путь к файлу для хранения URL (JSON)")
	databaseDSN := flag.String("d", "", "DSN для подключения к PostgreSQL")
	flag.Parse()
	if serverAddressEnv := os.Getenv("SERVER_ADDRESS"); serverAddressEnv != "" {
		*serverAddress = serverAddressEnv
	}
	if baseURLEnv := os.Getenv("BASE_URL"); baseURLEnv != "" {
		*baseURL = baseURLEnv
	}
	if filePathEnv := os.Getenv("FILE_STORAGE_PATH"); filePathEnv != "" {
		*filePath = filePathEnv
	}
	if databaseDSNEnv := os.Getenv("DATABASE_DSN"); databaseDSNEnv != "" {
		*databaseDSN = databaseDSNEnv
	}

	// Если DSN указан, используем PostgreSQL, иначе пробуем файл
	if *databaseDSN == "" && *filePath == "" {
		// Если ни DSN, ни путь к файлу не указаны, используем значение по умолчанию для файла
		*filePath = "storage.json"
	}

	cfg := &Config{
		ServerAddress:   *serverAddress,
		BaseURL:         *baseURL,
		FileStoragePath: *filePath,
		DatabaseDSN:     *databaseDSN,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ServerAddress == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}
	if c.FileStoragePath == "" {
		return fmt.Errorf("file storage path cannot be empty")
	}
	return nil
}

// HasDatabase возвращает true, если настроен DSN
func (c *Config) HasDatabase() bool {
	return c.DatabaseDSN != ""
}

// HasFileStorage возвращает true, если настроен путь к файлу
func (c *Config) HasFileStorage() bool {
	return c.FileStoragePath != ""
}

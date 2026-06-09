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
}

// NewConfig инициализирует конфигурацию из флагов командной строки
func NewConfig() (*Config, error) {
	serverAddress := flag.String("a", "localhost:8080", "адрес запуска HTTP-сервера")
	baseURL := flag.String("b", "http://localhost:8080", "базовый адрес результирующего сокращённого URL")
	filePath := flag.String("f", "", "путь к файлу для хранения URL (JSON)")
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

	// Значение по умолчанию для файла, если не указано
	if *filePath == "" {
		*filePath = "storage.json"
	}

	cfg := &Config{
		ServerAddress:   *serverAddress,
		BaseURL:         *baseURL,
		FileStoragePath: *filePath,
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

// // GetShortURL формирует полный короткий URL по ID
// func (c *Config) GetShortURL(shortID string) string {
// 	// Убираем trailing slash из BaseURL, если он есть
// 	baseURL := c.BaseURL
// 	if baseURL[len(baseURL)-1] == '/' {
// 		baseURL = baseURL[:len(baseURL)-1]
// 	}
// 	return fmt.Sprintf("%s/%s", baseURL, shortID)
// }

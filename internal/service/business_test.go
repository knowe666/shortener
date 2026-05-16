package business

import (
	"fmt"
	"testing"
)

// Mock репозитория для тестирования бизнес-логики
type MockURLRepository struct {
	urls      map[string]string
	cache     map[string]string
	saveError error
	getError  error
}

func NewMockURLRepository() *MockURLRepository {
	return &MockURLRepository{
		urls:  make(map[string]string),
		cache: make(map[string]string),
	}
}

func (m *MockURLRepository) Save(shortID, originalURL string) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.urls[shortID] = originalURL
	m.cache[originalURL] = shortID
	return nil
}

func (m *MockURLRepository) Get(shortID string) (string, error) {
	if m.getError != nil {
		return "", m.getError
	}
	url, exists := m.urls[shortID]
	if !exists {
		return "", fmt.Errorf("not found")
	}
	return url, nil
}

func TestURLShortenerService_CreateShortURL_Duplicate(t *testing.T) {
	repo := NewMockURLRepository()
	service := NewURLShortenerService(repo, "http://localhost:8080")

	originalURL := "https://example.com"

	// Первое создание
	first, err := service.CreateShortURL(originalURL)
	if err != nil {
		t.Fatalf("First creation failed: %v", err)
	}

	// Второе создание с тем же URL
	second, err := service.CreateShortURL(originalURL)
	if err != nil {
		t.Fatalf("Second creation failed: %v", err)
	}

	// Должна вернуться та же ссылка
	if first != second {
		t.Errorf("Expected same short URL, got %s and %s", first, second)
	}
}

func TestURLShortenerService_CreateShortURL(t *testing.T) {
	tests := []struct {
		name         string
		originalURL  string
		baseURL      string
		setupMock    func(*MockURLRepository)
		wantErr      bool
		wantContains string
	}{
		{
			name:         "Invalid URL without scheme",
			originalURL:  "example.com",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *MockURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
		{
			name:         "Empty URL",
			originalURL:  "",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *MockURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockURLRepository()
			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewURLShortenerService(mockRepo, tt.baseURL)
			got, err := service.CreateShortURL(tt.originalURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShortURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != "" {
				if len(got) < len(tt.wantContains) {
					t.Errorf("CreateShortURL() = %v, want contains %v", got, tt.wantContains)
				}
			}
		})
	}
}

// Тест генерации уникальных ID
func TestGenerateShortID(t *testing.T) {
	// Проверяем, что ID генерируются
	id1, err := generateShortID()
	if err != nil {
		t.Errorf("generateShortID() failed: %v", err)
	}

	if len(id1) != 8 {
		t.Errorf("Generated ID length = %d, want 8", len(id1))
	}

	// Проверяем уникальность
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, _ := generateShortID()
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

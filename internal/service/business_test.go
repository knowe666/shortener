package business

import (
	"sync"
	"testing"
	"time"

	"github.com/knowe666/shortener/internal/repository"
)

func TestURLShortenerService_CreateShortURL_Duplicate(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	service := NewURLShortenerService(repo, "http://localhost:8080")

	originalURL := "https://example.com"

	// Первое создание
	first, err := service.CreateShortURL(originalURL, "user1")
	if err != nil {
		t.Fatalf("First creation failed: %v", err)
	}

	// Второе создание с тем же URL
	second, err := service.CreateShortURL(originalURL, "user1")
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
		setupMock    func(*repository.InMemoryURLRepository)
		wantErr      bool
		wantContains string
	}{
		{
			name:         "Valid URL creation",
			originalURL:  "https://example.com",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *repository.InMemoryURLRepository) {},
			wantErr:      false,
			wantContains: "http://localhost:8080/",
		},
		{
			name:         "Invalid URL without scheme",
			originalURL:  "example.com",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *repository.InMemoryURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
		{
			name:         "Empty URL",
			originalURL:  "",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *repository.InMemoryURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewInMemoryURLRepository()
			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewURLShortenerService(mockRepo, tt.baseURL)
			got, err := service.CreateShortURL(tt.originalURL, "user1")

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

func TestDeleteBatcher_FlushesBatchedDeletes(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	service := NewURLShortenerService(repo, "http://localhost:8080")
	if _, err := service.CreateShortURL("https://example.com", "user1"); err != nil {
		t.Fatalf("failed to seed URL: %v", err)
	}

	batcher := NewDeleteBatcher(service, 2, 10*time.Millisecond)
	batcher.Start()
	defer batcher.Stop()

	var mu sync.Mutex
	calls := 0
	batcher.DeleteFunc = func(userID string, shortIDs []string) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return service.DeleteUserURLs(userID, shortIDs)
	}

	batcher.Enqueue("user1", []string{"a1b2c3d4"})
	batcher.Enqueue("user1", []string{"a1b2c3d4"})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	got := calls
	mu.Unlock()

	if got == 0 {
		t.Fatal("expected batcher to flush batched deletes")
	}
}

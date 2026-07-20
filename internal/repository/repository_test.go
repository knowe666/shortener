package repository

import (
	"testing"
)

func TestInMemoryURLRepository_Save(t *testing.T) {
	repo := NewInMemoryURLRepository()

	tests := []struct {
		name        string
		shortID     string
		originalURL string
		wantErr     bool
	}{
		{
			name:        "Valid save",
			shortID:     "abc123",
			originalURL: "https://example.com",
			wantErr:     false,
		},
		{
			name:        "Save with empty shortID",
			shortID:     "",
			originalURL: "https://example.com",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Save(tt.shortID, tt.originalURL, "user1")
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Проверяем, что сохранилось
			if !tt.wantErr {
				saved, err := repo.Get(tt.shortID)
				if err != nil {
					t.Errorf("Failed to find saved URL: %v", err)
				}
				if saved != tt.originalURL {
					t.Errorf("Saved URL = %v, want %v", saved, tt.originalURL)
				}
			}
		})
	}
}

// Тест конкурентного доступа
func TestInMemoryURLRepository_Concurrent(t *testing.T) {
	repo := NewInMemoryURLRepository()

	// Запускаем несколько горутин для записи
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(id int) {
			shortID := string(rune(id))
			repo.Save(shortID, "https://example.com", "user1")
			done <- true
		}(i)
	}

	// Ждём завершения всех горутин
	for i := 0; i < 100; i++ {
		<-done
	}

	// Проверяем, что данные сохранились
	if len(repo.urls) != 100 {
		t.Errorf("Expected 100 items, got %d", len(repo.urls))
	}
}

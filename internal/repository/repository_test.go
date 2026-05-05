package repository

import (
	"sync"
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
			wantErr:     false,
		},
		{
			name:        "Save duplicate shortID",
			shortID:     "duplicate",
			originalURL: "https://first.com",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Save(tt.shortID, tt.originalURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Проверяем, что сохранилось
			if !tt.wantErr {
				saved, err := repo.FindByShortID(tt.shortID)
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

func TestInMemoryURLRepository_FindByShortID(t *testing.T) {
	repo := NewInMemoryURLRepository()

	// Подготовка данных
	repo.Save("test123", "https://google.com")
	repo.Save("test456", "https://github.com")

	tests := []struct {
		name    string
		shortID string
		want    string
		wantErr bool
	}{
		{
			name:    "Existing short ID",
			shortID: "test123",
			want:    "https://google.com",
			wantErr: false,
		},
		{
			name:    "Existing short ID second",
			shortID: "test456",
			want:    "https://github.com",
			wantErr: false,
		},
		{
			name:    "Non-existing short ID",
			shortID: "nonexistent",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Empty short ID",
			shortID: "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByShortID(tt.shortID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByShortID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindByShortID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryURLRepository_FindByOriginalURL(t *testing.T) {
	repo := NewInMemoryURLRepository()

	// Подготовка данных
	repo.Save("short1", "https://example1.com")
	repo.Save("short2", "https://example2.com")

	tests := []struct {
		name        string
		originalURL string
		want        string
		wantErr     bool
	}{
		{
			name:        "Existing original URL",
			originalURL: "https://example1.com",
			want:        "short1",
			wantErr:     false,
		},
		{
			name:        "Existing second URL",
			originalURL: "https://example2.com",
			want:        "short2",
			wantErr:     false,
		},
		{
			name:        "Non-existing original URL",
			originalURL: "https://nonexistent.com",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "Empty URL",
			originalURL: "",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByOriginalURL(tt.originalURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByOriginalURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindByOriginalURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryURLRepository_Exists(t *testing.T) {
	repo := NewInMemoryURLRepository()
	repo.Save("existing", "https://example.com")

	tests := []struct {
		name    string
		shortID string
		want    bool
	}{
		{
			name:    "Existing short ID",
			shortID: "existing",
			want:    true,
		},
		{
			name:    "Non-existing short ID",
			shortID: "notexist",
			want:    false,
		},
		{
			name:    "Empty short ID",
			shortID: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repo.Exists(tt.shortID); got != tt.want {
				t.Errorf("Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Тест конкурентного доступа
func TestInMemoryURLRepository_Concurrent(t *testing.T) {
	repo := NewInMemoryURLRepository()

	// Запускаем несколько горутин для записи
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			shortID := string(rune(id + 65)) // A, B, C, ...
			repo.Save(shortID, "https://example.com")
		}(i)
	}

	wg.Wait()

	// Проверяем, что данные сохранились
	if len(repo.urls) != 100 {
		t.Errorf("Expected 100 items, got %d", len(repo.urls))
	}
}

// Тест перезаписи существующего ключа
func TestInMemoryURLRepository_Overwrite(t *testing.T) {
	repo := NewInMemoryURLRepository()

	// Сохраняем первый раз
	err := repo.Save("test", "https://first.com")
	if err != nil {
		t.Errorf("First save failed: %v", err)
	}

	// Сохраняем с тем же ID, но другим URL
	err = repo.Save("test", "https://second.com")
	if err != nil {
		t.Errorf("Second save failed: %v", err)
	}

	// Проверяем, что значение обновилось
	original, err := repo.FindByShortID("test")
	if err != nil {
		t.Errorf("Find failed: %v", err)
	}

	if original != "https://second.com" {
		t.Errorf("Expected https://second.com, got %s", original)
	}
}

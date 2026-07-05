package repository

import (
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestPostgresURLRepository(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("Skipping PostgreSQL tests: DATABASE_DSN not set")
	}

	repo, err := NewPostgresURLRepository(dsn)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Очищаем таблицу перед тестом
	_, err = repo.db.Exec("DELETE FROM urls")
	if err != nil {
		t.Fatalf("Failed to clean table: %v", err)
	}

	// Тест Save и Get
	testShortID := "test123"
	testURL := "https://test.com"

	err = repo.Save(testShortID, testURL)
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// Проверяем Get
	got, err := repo.Get(testShortID)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got != testURL {
		t.Errorf("Get() = %v, want %v", got, testURL)
	}

	// Тест дубликата
	err = repo.Save(testShortID, "https://another.com")
	if err != ErrDuplicateID {
		t.Errorf("Save duplicate = %v, want %v", err, ErrDuplicateID)
	}

	// Тест GetByOriginalURL
	shortID, err := repo.GetByOriginalURL(testURL)
	if err != nil {
		t.Errorf("GetByOriginalURL() error = %v", err)
	}
	if shortID != testShortID {
		t.Errorf("GetByOriginalURL() = %v, want %v", shortID, testShortID)
	}
}

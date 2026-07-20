package repository

import (
	"errors"
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

	// Тест дубликата URL
	anotherShortID := "test456"
	err = repo.Save(anotherShortID, testURL)
	if err == nil {
		t.Error("Save duplicate original URL should return error")
	}

	var dupErr *ErrDuplicateOriginalURL
	if errors.As(err, &dupErr) {
		if dupErr.ShortID != testShortID {
			t.Errorf("Expected short_id %s, got %s", testShortID, dupErr.ShortID)
		}
	} else {
		t.Errorf("Expected ErrDuplicateOriginalURL, got %v", err)
	}

	// Тест дубликата short_id
	err = repo.Save(testShortID, "https://another.com")
	if err != DuplicateIDError {
		t.Errorf("Save duplicate short_id = %v, want %v", err, DuplicateIDError)
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

func TestPostgresURLRepository_UniqueConstraint(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("Skipping PostgreSQL tests: DATABASE_DSN not set")
	}

	repo, err := NewPostgresURLRepository(dsn)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Очищаем таблицу
	_, err = repo.db.Exec("DELETE FROM urls")
	if err != nil {
		t.Fatalf("Failed to clean table: %v", err)
	}

	testURL := "https://unique-test.com"
	shortID1 := "unique1"
	shortID2 := "unique2"

	// Первая вставка должна пройти успешно
	err = repo.Save(shortID1, testURL)
	if err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	// Вторая вставка с тем же URL должна вернуть ErrDuplicateOriginalURL
	err = repo.Save(shortID2, testURL)
	if err == nil {
		t.Fatal("Second save should fail with duplicate error")
	}

	var dupErr *ErrDuplicateOriginalURL
	if !errors.As(err, &dupErr) {
		t.Fatalf("Expected ErrDuplicateOriginalURL, got %v", err)
	}

	if dupErr.ShortID != shortID1 {
		t.Errorf("Expected existing short_id %s, got %s", shortID1, dupErr.ShortID)
	}
}

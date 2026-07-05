package repository

import (
	"os"
	"testing"
)

func TestPostgresURLRepository(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set, skipping PostgreSQL tests")
	}

	repo, err := NewPostgresURLRepository(dsn)
	if err != nil {
		t.Fatalf("Failed to create Postgres repository: %v", err)
	}
	defer repo.Close()

	// Тест Save и Get
	shortID := "test123"
	originalURL := "https://test.com"

	err = repo.Save(shortID, originalURL)
	if err != nil {
		t.Errorf("Save failed: %v", err)
	}

	got, err := repo.Get(shortID)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if got != originalURL {
		t.Errorf("Get returned %s, want %s", got, originalURL)
	}

	// Тест Ping
	if err := repo.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Тест дубликата
	err = repo.Save(shortID, "https://another.com")
	if err == nil {
		t.Error("Expected duplicate error, got nil")
	}
}

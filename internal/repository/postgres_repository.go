package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	_ "github.com/lib/pq" // PostgreSQL драйвер
)

// PostgresURLRepository реализует URLRepository с хранением в PostgreSQL
type PostgresURLRepository struct {
	db *sql.DB
	mu sync.Mutex
}

// NewPostgresURLRepository создаёт новый экземпляр репозитория с подключением к PostgreSQL
func NewPostgresURLRepository(dsn string) (*PostgresURLRepository, error) {
	if dsn == "" {
		return nil, errors.New("DSN cannot be empty")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Создаем таблицу, если её нет
	if err := createTable(db); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &PostgresURLRepository{
		db: db,
	}, nil
}

// createTable создает таблицу для хранения URL
func createTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_id VARCHAR(8) UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_short_id ON urls(short_id);
	CREATE INDEX IF NOT EXISTS idx_original_url ON urls(original_url);
	`

	_, err := db.Exec(query)
	return err
}

// Save сохраняет короткую ссылку
func (r *PostgresURLRepository) Save(shortID, originalURL string) error {
	if shortID == "" {
		return ErrEmptyID
	}
	if originalURL == "" {
		return errors.New("original URL cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	query := `INSERT INTO urls (short_id, original_url) VALUES ($1, $2)`
	_, err := r.db.Exec(query, shortID, originalURL)
	if err != nil {
		// Проверяем, не нарушение ли уникальности
		if isDuplicateError(err) {
			return ErrDuplicateID
		}
		return fmt.Errorf("failed to save URL: %w", err)
	}
	return nil
}

// Get возвращает оригинальный URL по короткому ID
func (r *PostgresURLRepository) Get(shortID string) (string, error) {
	if shortID == "" {
		return "", ErrEmptyID
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	query := `SELECT original_url FROM urls WHERE short_id = $1`
	var originalURL string
	err := r.db.QueryRow(query, shortID).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFoundID
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}
	return originalURL, nil
}

// Ping проверяет соединение с базой данных
func (r *PostgresURLRepository) Ping() error {
	return r.db.Ping()
}

// Close закрывает соединение с базой данных
func (r *PostgresURLRepository) Close() error {
	return r.db.Close()
}

// isDuplicateError проверяет, является ли ошибка нарушением уникальности
func isDuplicateError(err error) bool {
	// PostgreSQL error code для уникального ограничения - 23505
	return err != nil && (err.Error() == `pq: duplicate key value violates unique constraint "urls_short_id_key"` ||
		err.Error() == `ERROR: duplicate key value violates unique constraint "urls_short_id_key" (SQLSTATE 23505)`)
}

package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// PostgresURLRepository реализует URLRepository для PostgreSQL
type PostgresURLRepository struct {
	db *sqlx.DB
}

// URLRecord представляет запись в БД
type URLRecordDB struct {
	ID          int       `db:"id"`
	ShortID     string    `db:"short_id"`
	OriginalURL string    `db:"original_url"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// NewPostgresURLRepository создаёт новый экземпляр репозитория с подключением к PostgreSQL
func NewPostgresURLRepository(dsn string) (*PostgresURLRepository, error) {
	if dsn == "" {
		return nil, errors.New("DSN cannot be empty")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	repo := &PostgresURLRepository{db: db}

	// Проверяем наличие таблиц и создаем если их нет
	if err := repo.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return repo, nil
}

// initSchema создает таблицу если она не существует
func (r *PostgresURLRepository) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_id VARCHAR(20) UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_short_id ON urls(short_id);
	CREATE INDEX IF NOT EXISTS idx_original_url ON urls(original_url);
	`

	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// Save сохраняет короткую ссылку
func (r *PostgresURLRepository) Save(shortID, originalURL string) error {
	if shortID == "" {
		return ErrEmptyID
	}
	if originalURL == "" {
		return errors.New("original URL cannot be empty")
	}

	query := `
		INSERT INTO urls (short_id, original_url)
		VALUES ($1, $2)
		ON CONFLICT (short_id) DO NOTHING
		RETURNING id
	`

	var id int
	err := r.db.QueryRowx(query, shortID, originalURL).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			// Проверяем, существует ли уже такая запись
			var exists bool
			checkQuery := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_id = $1)`
			if err := r.db.QueryRowx(checkQuery, shortID).Scan(&exists); err != nil {
				return fmt.Errorf("failed to check existing record: %w", err)
			}
			if exists {
				return ErrDuplicateID
			}
			return fmt.Errorf("failed to insert record: %w", err)
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

	query := `SELECT original_url FROM urls WHERE short_id = $1`
	var originalURL string
	err := r.db.Get(&originalURL, query, shortID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFoundID
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}
	return originalURL, nil
}

// GetByOriginalURL находит запись по оригинальному URL (для проверки дубликатов)
func (r *PostgresURLRepository) GetByOriginalURL(originalURL string) (string, error) {
	if originalURL == "" {
		return "", errors.New("original URL cannot be empty")
	}

	query := `SELECT short_id FROM urls WHERE original_url = $1 LIMIT 1`
	var shortID string
	err := r.db.Get(&shortID, query, originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFoundID
		}
		return "", fmt.Errorf("failed to get URL by original: %w", err)
	}

	return shortID, nil
}

// Close закрывает соединение с базой данных
func (r *PostgresURLRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// GetAll возвращает все записи (для тестирования)
func (r *PostgresURLRepository) GetAll() (map[string]string, error) {
	query := `SELECT short_id, original_url FROM urls`
	rows, err := r.db.Queryx(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all URLs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var shortID, originalURL string
		if err := rows.Scan(&shortID, &originalURL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result[shortID] = originalURL
	}

	return result, nil
}

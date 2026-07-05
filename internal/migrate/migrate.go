package migrate

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations выполняет миграции базы данных
func RunMigrations(dsn string) error {
	if dsn == "" {
		return errors.New("DSN cannot be empty")
	}

	// Открываем соединение с БД
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Создаем источник миграций из встроенных файлов
	srcDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	// Создаем драйвер базы данных
	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Создаем экземпляр мигратора
	m, err := migrate.NewWithInstance(
		"iofs", srcDriver,
		"postgres", dbDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Выполняем миграции
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Migrations applied successfully")
	return nil
}

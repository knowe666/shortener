package transport

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// URLService интерфейс для связи со слоем бизнес-логики
type URLService interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
}

// SetupRouter настраивает и возвращает маршрутизатор
func SetupRouter(handler *URLHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(loggingmiddleware)
	r.Use(gzipHandle)
	r.Use(middleware.Recoverer)
	r.Post("/", handler.HandlePost)
	r.Post("/api/shorten", handler.HandleAPIShorten)
	r.Get("/{id}", handler.HandleGet)
	r.Get("/ping", handler.HandlePing) // Добавляем эндпоинт ping

	return r
}

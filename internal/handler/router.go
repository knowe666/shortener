package transport

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type UserURLData struct {
	ShortURL    string
	OriginalURL string
}

// URLService интерфейс для связи со слоем бизнес-логики
type URLService interface {
	CreateShortURL(originalURL, userID string) (string, error)
	GetOriginalURL(shortID string) (string, error)
	GetUserURLs(userID string) ([]UserURLData, error)
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
	r.Get("/ping", handler.HandlePing)
	r.Get("/api/user/urls", handler.HandleUserURLs)

	return r
}

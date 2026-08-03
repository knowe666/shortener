package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/knowe666/shortener/internal/auth"
	"github.com/knowe666/shortener/internal/repository"
)

// URLService интерфейс для связи со слоем бизнес-логики
type URLService interface {
	CreateShortURL(originalURL, userID string) (string, error)
	GetOriginalURL(shortID string) (string, error)
	GetUserURLs(userID string) ([]repository.URLData, error)
	DeleteUserURLs(userID string, shortIDs []string) error
}

func AuthMiddleware(authenticator auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := authenticator.GetUserIDFromCookie(r)
			if err != nil {
				userID = auth.GenerateUserID()
				authenticator.SetUserCookie(w, userID)
			}

			ctx := auth.SetUserIDToContext(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

	r.Route("/api/user/urls", func(r chi.Router) {
		r.Use(AuthMiddleware(handler.authenticator))
		r.Get("/", handler.HandleUserURLs)
		r.Delete("/", handler.HandleDeleteUserURLs)
	})

	return r
}

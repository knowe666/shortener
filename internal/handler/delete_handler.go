package transport

import (
	"encoding/json"
	"net/http"

	"github.com/knowe666/shortener/internal/auth"
	"go.uber.org/zap"
)

// HandleDeleteUserURLs обрабатывает DELETE /api/user/urls
// Принимает список ID для асинхронного удаления
// Возвращает 202 Accepted
func (h *URLHandler) HandleDeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == "" {
		h.logger.Warn("Unauthorized access to delete URLs")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Проверяем Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Декодируем список ID
	var shortIDs []string
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&shortIDs); err != nil {
		h.logger.Error("Failed to decode JSON request", zap.Error(err))
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Проверяем, что список не пустой
	if len(shortIDs) == 0 {
		http.Error(w, "Empty short IDs list", http.StatusBadRequest)
		return
	}

	h.logger.Info("Deleting user URLs", zap.String("user_id", userID), zap.Int("count", len(shortIDs)))

	if h.deleteBatcher != nil {
		if err := h.deleteBatcher.Enqueue(userID, shortIDs); err != nil {
			h.logger.Error("Failed to enqueue delete request", zap.String("user_id", userID), zap.Error(err))
			http.Error(w, "Failed to enqueue delete request", http.StatusServiceUnavailable)
			return
		}
	} else {
		// Fallback для тестов: оставляем старую семантику только если батчер не передан.
		go func() {
			if err := h.service.DeleteUserURLs(userID, shortIDs); err != nil {
				h.logger.Error("Failed to delete URLs", zap.String("user_id", userID), zap.Error(err))
			} else {
				h.logger.Info("Successfully deleted URLs", zap.String("user_id", userID), zap.Int("count", len(shortIDs)))
			}
		}()
	}

	// Возвращаем 202 Accepted - запрос принят, обработка асинхронная
	w.WriteHeader(http.StatusAccepted)
}

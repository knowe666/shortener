package transport

import (
	"encoding/json"
	"net/http"

	"github.com/knowe666/shortener/internal/auth"
	"go.uber.org/zap"
)

func (h *URLHandler) HandleUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == "" {
		h.logger.Warn("Unauthorized access to user URLs")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Info("Getting user URLs", zap.String("user_id", userID))

	urls, err := h.service.GetUserURLs(userID)
	if err != nil {
		h.logger.Error("Failed to get user URLs", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(urls); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	cookieName   = "user_id"
	cookieMaxAge = 24 * 60 * 60 // 24 часа
)

var (
	ErrInvalidCookie = errors.New("invalid or expired cookie")
	ErrNoCookie      = errors.New("no cookie found")
	secretKey        []byte
)

// InitAuth инициализирует секретный ключ для подписи
func InitAuth(secret string) {
	secretKey = []byte(secret)
	if len(secretKey) == 0 {
		// Если секрет не задан, используем случайный
		secretKey = []byte("default-secret-key-change-me-in-production")
	}
}

// GenerateUserID генерирует новый уникальный ID пользователя
func GenerateUserID() string {
	return uuid.New().String()
}

// signUserID создает подпись для userID
func signUserID(userID string) string {
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(userID))
	return hex.EncodeToString(mac.Sum(nil))
}

// verifySignature проверяет подпись userID
func verifySignature(userID, signature string) bool {
	expected := signUserID(userID)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// SetUserCookie устанавливает подписанную куку с userID
func SetUserCookie(w http.ResponseWriter, userID string) {
	signature := signUserID(userID)
	cookieValue := base64.URLEncoding.EncodeToString([]byte(userID + ":" + signature))

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    cookieValue,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   false, // В продакшене должен быть true
		SameSite: http.SameSiteLaxMode,
	})
}

// GetUserIDFromCookie извлекает и проверяет userID из куки
func GetUserIDFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", ErrNoCookie
		}
		return "", fmt.Errorf("failed to get cookie: %w", err)
	}

	// Декодируем значение
	decoded, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", ErrInvalidCookie
	}

	// Разделяем userID и подпись
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", ErrInvalidCookie
	}

	userID := parts[0]
	signature := parts[1]

	// Проверяем подпись
	if !verifySignature(userID, signature) {
		return "", ErrInvalidCookie
	}

	return userID, nil
}

// GetOrCreateUserID возвращает существующий userID из куки или создает новый
func GetOrCreateUserID(w http.ResponseWriter, r *http.Request) string {
	userID, err := GetUserIDFromCookie(r)
	if err == nil {
		return userID
	}

	// Генерируем новый ID
	newUserID := GenerateUserID()
	SetUserCookie(w, newUserID)
	return newUserID
}

// Middleware для проверки аутентификации (требует наличия валидной куки)
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromCookie(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Добавляем userID в контекст запроса для дальнейшего использования
		ctx := SetUserIDToContext(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

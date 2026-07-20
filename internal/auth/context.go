package auth

import "context"

type contextKey string

const userIDContextKey contextKey = "userID"

// SetUserIDToContext добавляет userID в контекст
func SetUserIDToContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// GetUserIDFromContext извлекает userID из контекста
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

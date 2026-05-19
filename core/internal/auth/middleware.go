package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// ctxKey — приватный тип ключа для context.
type ctxKey struct{}

var userKey = ctxKey{}

// CurrentUser извлекает userID из context. Возвращает (zero, false) если нет.
func CurrentUser(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(userKey).(*Claims)
	if !ok {
		return uuid.Nil, false
	}
	return v.UserID, true
}

// CurrentClaims возвращает полные claims из context.
func CurrentClaims(ctx context.Context) (*Claims, bool) {
	v, ok := ctx.Value(userKey).(*Claims)
	return v, ok
}

// Middleware проверяет Authorization: Bearer <token> и кладёт Claims в context.
// При невалидном/отсутствующем токене — 401.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" {
			http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"bad Authorization format"}`, http.StatusUnauthorized)
			return
		}
		claims, err := ParseToken(parts[1])
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

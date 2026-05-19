package api

import (
	"net/http"
	"strings"
)

// CORS возвращает middleware, разрешающее запросы с указанных origin'ов.
// Список origin'ов — CSV в CORE_CORS_ORIGINS (пустой — выключено).
// Спец-значение "*" разрешает любой Origin (для совместимости с публичным API).
//
// Поддерживает preflight OPTIONS, заголовки Authorization и Content-Type,
// все методы используемые роутером (GET, POST, OPTIONS).
func CORS(originsCSV string) func(http.Handler) http.Handler {
	allowed := parseOrigins(originsCSV)
	allowAny := len(allowed) == 1 && allowed[0] == "*"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAny || originAllowed(origin, allowed)) {
				if allowAny {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseOrigins(csv string) []string {
	out := []string{}
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}

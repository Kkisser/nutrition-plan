package api

import (
	"net/http"

	"nutrition-core/internal/auth"
)

// Routes регистрирует обработчики на mux.
//
// Открытые эндпоинты: /health, /auth/{register,login,verify}.
// Защищённые (требуют JWT): /plan, /pricing, /catalog, /auth/me.
//
// authLimiter (nil-able) применяется к POST /auth/{register,login}
// для защиты от перебора паролей и регистрационного спама.
func Routes(mux *http.ServeMux, h *Handler, authLimiter *RateLimiter) {
	// Открытые
	mux.HandleFunc("GET /health", h.Health)

	register := http.HandlerFunc(h.PostAuthRegister)
	login := http.HandlerFunc(h.PostAuthLogin)
	if authLimiter != nil {
		mux.Handle("POST /auth/register", authLimiter.Middleware(register))
		mux.Handle("POST /auth/login", authLimiter.Middleware(login))
	} else {
		mux.Handle("POST /auth/register", register)
		mux.Handle("POST /auth/login", login)
	}
	mux.HandleFunc("POST /auth/verify", h.PostAuthVerify)

	// Защищённые — оборачиваем в auth middleware
	mux.Handle("POST /plan", auth.Middleware(http.HandlerFunc(h.PostPlan)))
	mux.Handle("POST /pricing", auth.Middleware(http.HandlerFunc(h.PostPricing)))
	mux.Handle("GET /catalog", auth.Middleware(http.HandlerFunc(h.GetCatalog)))
	mux.Handle("GET /auth/me", auth.Middleware(http.HandlerFunc(h.GetAuthMe)))
}

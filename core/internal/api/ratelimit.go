package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter — простой in-memory token-bucket лимитер по IP.
// Используется для защиты эндпоинтов /auth/{register,login} от
// перебора паролей и регистрационного спама. Параметры берутся из
// ENV (CORE_AUTH_RATE_RPM, дефолт 10 запросов/мин).
//
// Реализация: per-IP счётчик с окном w. По истечении окна счётчик
// сбрасывается. Это даёт грубый, но достаточный для не-распределённого
// dev/single-host прода контроль.
type RateLimiter struct {
	max    int           // макс. запросов в окне
	window time.Duration // окно
	mu     sync.Mutex
	hits   map[string]*bucket
}

type bucket struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter создаёт лимитер; max=0 отключает (Allow всегда true).
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	if max <= 0 {
		return &RateLimiter{max: 0}
	}
	rl := &RateLimiter{
		max:    max,
		window: window,
		hits:   make(map[string]*bucket),
	}
	go rl.gcLoop()
	return rl
}

// Allow возвращает (ok, retryAfter). Если ok=false, клиент должен ждать
// retryAfter секунд.
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	if rl.max == 0 {
		return true, 0
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.hits[key]
	if !ok || now.After(b.resetAt) {
		rl.hits[key] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true, 0
	}
	if b.count >= rl.max {
		return false, time.Until(b.resetAt)
	}
	b.count++
	return true, 0
}

func (rl *RateLimiter) gcLoop() {
	t := time.NewTicker(rl.window * 2)
	defer t.Stop()
	for range t.C {
		rl.mu.Lock()
		now := time.Now()
		for k, b := range rl.hits {
			if now.After(b.resetAt) {
				delete(rl.hits, k)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware оборачивает обработчик с лимитом по IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ok, retry := rl.Allow(ip); !ok {
			secs := int(retry.Seconds())
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", itoa(secs))
			http.Error(w, "слишком много запросов, попробуйте позже", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func itoa(i int) string {
	// без strconv для микро-зависимости — нужно только для Retry-After.
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func clientIP(r *http.Request) string {
	// Доверять X-Forwarded-For имеет смысл только за proxy. Для dev/single-host
	// проще брать r.RemoteAddr. Если фронт за nginx — задать заголовок и
	// разобрать здесь первым нехостовым значением.
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		if i := strings.Index(xf, ","); i > 0 {
			return strings.TrimSpace(xf[:i])
		}
		return strings.TrimSpace(xf)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

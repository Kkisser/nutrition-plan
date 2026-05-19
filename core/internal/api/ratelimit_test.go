package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Disabled(t *testing.T) {
	rl := NewRateLimiter(0, time.Minute)
	if ok, _ := rl.Allow("1.2.3.4"); !ok {
		t.Fatal("disabled limiter must always allow")
	}
}

func TestRateLimiter_AllowsUpToMax(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		ok, _ := rl.Allow("1.2.3.4")
		if !ok {
			t.Fatalf("request %d denied early", i+1)
		}
	}
	ok, retry := rl.Allow("1.2.3.4")
	if ok {
		t.Fatal("4th request must be denied")
	}
	if retry <= 0 {
		t.Errorf("retry-after must be positive, got %s", retry)
	}
}

func TestRateLimiter_PerKey(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	rl.Allow("a")
	if ok, _ := rl.Allow("b"); !ok {
		t.Fatal("limit must be per-key, b should pass independently of a")
	}
	if ok, _ := rl.Allow("a"); ok {
		t.Fatal("a should be over limit")
	}
}

func TestRateLimiter_Middleware_TooManyRequests(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	called := 0
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "1.2.3.4:5555"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if i == 0 && rr.Code != 200 {
			t.Fatalf("first request: %d", rr.Code)
		}
		if i == 1 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("second request: %d", rr.Code)
		}
	}
	if called != 1 {
		t.Fatalf("handler called %d times, expected 1", called)
	}
}

func TestRateLimiter_ClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	if got := clientIP(req); got != "203.0.113.5" {
		t.Errorf("clientIP = %q, want 203.0.113.5", got)
	}
}

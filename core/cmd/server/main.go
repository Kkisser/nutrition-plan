// nutrition-core server — формирование персонализированного плана питания.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"nutrition-core/internal/api"
	"nutrition-core/internal/auth"
	"nutrition-core/internal/config"
	"nutrition-core/internal/db"
	"nutrition-core/internal/mailer"
	"nutrition-core/internal/pricing"
	pgrepo "nutrition-core/internal/repository/pg"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	logger := newLogger()
	slog.SetDefault(logger)

	addr := envOrDefault("CORE_HTTP_ADDR", ":8080")

	cfg, err := config.LoadPenalty()
	if err != nil {
		return err
	}

	dsn, err := db.DSNFromEnv()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	repo := pgrepo.New(pool)

	// pricing — опциональная подсистема. Если PRICE_SERVICE_URL не задан,
	// эндпоинт /pricing вернёт 503; план и список покупок строятся как обычно.
	var pricingClient *pricing.Client
	if u := os.Getenv("PRICE_SERVICE_URL"); u != "" {
		pricingClient = pricing.New(u)
		logger.Info("pricing enabled", slog.String("url", u))
	} else {
		logger.Info("pricing disabled", slog.String("reason", "PRICE_SERVICE_URL not set"))
	}

	users := auth.NewStore(pool)

	mailProvider := strings.ToLower(envOrDefault("CORE_MAIL_PROVIDER", "log"))
	var m mailer.Mailer
	switch mailProvider {
	case "log":
		m = &mailer.LogMailer{Logger: logger}
		logger.Info("mailer configured", slog.String("provider", "log"))
	default:
		logger.Warn("unknown CORE_MAIL_PROVIDER, falling back to log",
			slog.String("requested", mailProvider))
		m = &mailer.LogMailer{Logger: logger}
	}

	// В dev оставляем confirm_token в response для удобства e2e curl-сниппетов.
	// В prod (CORE_EXPOSE_AUTH_TOKEN=false) пользователь получает токен только
	// через mailer — это поведение, ожидаемое от реального SMTP.
	exposeToken := strings.ToLower(envOrDefault("CORE_EXPOSE_AUTH_TOKEN", "true")) == "true"

	handler := api.NewHandler(repo, cfg, pricingClient, users, m, exposeToken)

	mux := http.NewServeMux()
	var authLimiter *api.RateLimiter
	if rpm := envIntOrDefault("CORE_AUTH_RATE_RPM", 10); rpm > 0 {
		authLimiter = api.NewRateLimiter(rpm, time.Minute)
		logger.Info("auth rate limit configured", slog.Int("rpm_per_ip", rpm))
	}
	api.Routes(mux, handler, authLimiter)

	var rootHandler http.Handler = api.RequestLogger(logger)(mux)
	if origins := os.Getenv("CORE_CORS_ORIGINS"); origins != "" {
		rootHandler = api.CORS(origins)(rootHandler)
		logger.Info("CORS enabled", slog.String("origins", origins))
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           rootHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down")
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("listening",
		slog.String("addr", addr),
		slog.Float64("corridor", cfg.CorridorRel),
		slog.Float64("w1", cfg.W1),
		slog.Float64("w2", cfg.W2),
		slog.Float64("w3", cfg.W3),
		slog.Int("k", cfg.K),
	)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// newLogger возвращает slog.Logger. По умолчанию текстовый формат для удобства
// чтения логов в dev; CORE_LOG_FORMAT=json — JSONHandler для продакшена.
// CORE_LOG_LEVEL — info/debug/warn/error (по умолчанию info).
func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("CORE_LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	if strings.ToLower(os.Getenv("CORE_LOG_FORMAT")) == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// Package mailer отправляет транзакционные письма приложения.
//
// В dev по умолчанию используется LogMailer — пишет письмо в slog как
// структурированную запись. Реальный SMTP-провайдер подключается тем же
// интерфейсом, переключение через ENV CORE_MAIL_PROVIDER.
package mailer

import (
	"context"
	"log/slog"
)

// Mailer — единственный способ отправить транзакционное письмо. Конкретная
// имплементация выбирается на старте сервера; вызывающий код не знает, идёт
// ли письмо в SMTP, в лог или в очередь.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// Message — параметры одного письма. Тело текстовое; HTML не используется
// до выхода в prod (PROBLEM: некоторые SMTP-провайдеры markdown в plain
// тексте не парсят).
type Message struct {
	To      string
	Subject string
	Body    string
}

// LogMailer пишет письмо в slog на уровне INFO. Используется по умолчанию
// в dev: на защите можно показать `tail -f` лога core и увидеть «отправленное»
// письмо с токеном подтверждения.
type LogMailer struct {
	Logger *slog.Logger
}

// Send удовлетворяет Mailer.
func (m *LogMailer) Send(ctx context.Context, msg Message) error {
	logger := m.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "email sent (mock)",
		slog.String("provider", "log"),
		slog.String("to", msg.To),
		slog.String("subject", msg.Subject),
		slog.String("body", msg.Body),
	)
	return nil
}

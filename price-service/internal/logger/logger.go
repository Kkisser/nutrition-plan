package logger

import (
	"log"
)

type Logger struct {
	debug bool
}

func New(debug bool) Logger {
	return Logger{debug: debug}
}

func (l Logger) Infof(format string, args ...any) {
	log.Printf(format, args...)
}

func (l Logger) Debugf(format string, args ...any) {
	if l.debug {
		log.Printf(format, args...)
	}
}

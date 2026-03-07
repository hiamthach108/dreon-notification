package logger

import "go.uber.org/zap"

type Level string

const (
	DebugLv Level = "debug"
	InfoLv  Level = "info"
	WarnLv  Level = "warn"
	ErrorLv Level = "error"
)

type ILogger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Fatal(msg string, fields ...any)
	With(fields ...any) ILogger
	GetZapLogger() *zap.Logger
}

type Field struct {
	Key string
	Val any
}

type IDecorator interface {
	Decorate(ILogger) ILogger
}

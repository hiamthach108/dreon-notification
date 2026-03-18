package events

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
)

// zapLoggerAdapter adapts logger.ILogger to watermill.LoggerAdapter.
type zapLoggerAdapter struct {
	log logger.ILogger
}

// NewLoggerAdapter returns a watermill.LoggerAdapter that uses the app logger.
func NewLoggerAdapter(l logger.ILogger) watermill.LoggerAdapter {
	return &zapLoggerAdapter{log: l}
}

func (a *zapLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	args := fieldsToArgs(fields)
	if err != nil {
		args = append(args, "error", err)
	}
	a.log.Error(msg, args...)
}

func (a *zapLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	a.log.Info(msg, fieldsToArgs(fields)...)
}

func (a *zapLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	a.log.Debug(msg, fieldsToArgs(fields)...)
}

func (a *zapLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
	a.log.Debug(msg, fieldsToArgs(fields)...)
}

func (a *zapLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &zapLoggerAdapter{log: a.log.With(fieldsToArgs(fields)...)}
}

func fieldsToArgs(fields watermill.LogFields) []any {
	if len(fields) == 0 {
		return nil
	}
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}

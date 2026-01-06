package logger

import (
    "fmt"
    "io"
    "log/slog"
)

// Logger wraps slog.Logger with helper methods used across the codebase.
type Logger struct {
    l *slog.Logger
}

// New constructs a Logger. If the provided slog.Logger is nil, output is discarded.
func New(l *slog.Logger) *Logger {
    if l == nil {
        l = slog.New(slog.NewJSONHandler(io.Discard, nil))
    }
    return &Logger{l: l}
}

// With attaches key/value pairs to the logger, returning a new derived logger.
func (lg *Logger) With(args ...any) *Logger {
    return &Logger{l: lg.base().With(args...)}
}

func (lg *Logger) base() *slog.Logger {
    if lg == nil || lg.l == nil {
        return slog.New(slog.NewJSONHandler(io.Discard, nil))
    }
    return lg.l
}

func (lg *Logger) Printf(format string, args ...any) {
    lg.base().Info(fmt.Sprintf(format, args...))
}

func (lg *Logger) Println(args ...any) {
    lg.base().Info(fmt.Sprintln(args...))
}

func (lg *Logger) Print(args ...any) {
    lg.base().Info(fmt.Sprint(args...))
}

func (lg *Logger) Errorf(format string, args ...any) {
    lg.base().Error(fmt.Sprintf(format, args...))
}

func (lg *Logger) Debugf(format string, args ...any) {
    lg.base().Debug(fmt.Sprintf(format, args...))
}

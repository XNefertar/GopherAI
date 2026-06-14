package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	defaultLogger = slog.New(handler)
}

// With 创建带额外上下文字段的子 logger。
// 用法：l := logger.With("userName", userName, "sessionID", sessionID)
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

// Info 输出 info 级别结构化日志
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Error 输出 error 级别结构化日志
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Warn 输出 warn 级别结构化日志
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Debug 输出 debug 级别结构化日志
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

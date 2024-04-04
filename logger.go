package framework

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type LibLogger struct {
	stringer fmt.Stringer
	w        io.WriteCloser
	minLevel slog.Level
	pretty   bool
}

func NewLibLogger(stringer fmt.Stringer, logname string) (*slog.Logger, error) {
	err := os.MkdirAll(LogDir(), 0700)
	if err != nil {
		return nil, fmt.Errorf("error while creating log directory: %s: %w", LogDir(), err)
	}

	filename := filepath.Join(LogDir(), logname)
	logfile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0700)
	if err != nil {
		return nil, fmt.Errorf("error while creating log file: %s: %w", filename, err)
	}

	minLevel := slog.LevelInfo
	levelStr := os.Getenv("JARBLES_LOG_LEVEL")
	if levelStr != "" {
		err := minLevel.UnmarshalText([]byte(levelStr))
		if err != nil {
			minLevel = slog.LevelInfo
		}
	}

	pretty := true
	prettyStr := os.Getenv("JARBLES_LOG_PRETTY")
	if prettyStr == "false" {
		pretty = false
	}

	return slog.New(&LibLogger{stringer: stringer, w: logfile, minLevel: minLevel, pretty: pretty}), nil
}

func (l LibLogger) Enabled(context context.Context, level slog.Level) bool {
	return level >= l.minLevel
}

func (l LibLogger) Handle(context context.Context, record slog.Record) error {
	message := record.Message

	line := ""
	if l.pretty {
		attrs := make([]string, 0)
		record.Attrs(func(attr slog.Attr) bool {
			attrs = append(attrs, fmt.Sprintf("- %v: %v", attr.Key, attr.Value))
			return true
		})

		timestamp := record.Time.Format(time.Kitchen)
		line += fmt.Sprintf("\n%v %v %v\n", timestamp, levelAbbrev(record.Level), message)
		for _, attr := range attrs {
			line += fmt.Sprintf("  %v\n", attr)
		}
	} else {
		record.Attrs(func(attr slog.Attr) bool {
			message += fmt.Sprintf(" %v", attr)
			return true
		})

		timestamp := record.Time.Format(time.Kitchen)
		line = fmt.Sprintf("[%v] %s %v %v", record.Level, l.stringer.String(), timestamp, message)
	}

	_, err := fmt.Fprintf(l.w, "%s\n", line)
	return err
}

func levelAbbrev(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DBG"
	case slog.LevelInfo:
		return "INF"
	case slog.LevelWarn:
		return "WRN"
	case slog.LevelError:
		return "ERR"
	default:
		return "???"
	}
}

func (l LibLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	panic("unimplemented")
}

func (l LibLogger) WithGroup(name string) slog.Handler {
	panic("unimplemented")
}

func (l LibLogger) Close() error {
	return l.w.Close()
}

//goland:noinspection GoUnusedExportedFunction
func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	logger.Log(ctx, level, msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logger.LogAttrs(ctx, level, msg, attrs...)
}

//goland:noinspection GoUnusedExportedFunction
func LogDebug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogDebugContext(ctx context.Context, msg string, args ...any) {
	logger.DebugContext(ctx, msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogInfo(msg string, args ...any) {
	logger.Info(msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogInfoContext(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogWarn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogWarnContext(ctx context.Context, msg string, args ...any) {
	logger.WarnContext(ctx, msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogError(msg string, args ...any) {
	logger.Error(msg, args...)
}

//goland:noinspection GoUnusedExportedFunction
func LogErrorContext(ctx context.Context, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, args...)
}

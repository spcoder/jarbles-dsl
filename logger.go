package jarbles_framework

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type LibLogger struct {
	stringer fmt.Stringer
	w        io.WriteCloser
	minLevel slog.Level
	trunc    int
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

	trunc := 0
	truncStr := os.Getenv("JARBLES_LOG_TRUNCATE")
	if truncStr != "" {
		tl, err := strconv.Atoi(truncStr)
		if err == nil {
			trunc = 120
		}
		trunc = tl
	}

	return slog.New(&LibLogger{stringer: stringer, w: logfile, trunc: trunc, minLevel: minLevel}), nil
}

func (l LibLogger) Enabled(context context.Context, level slog.Level) bool {
	return level >= l.minLevel
}

func (l LibLogger) Handle(context context.Context, record slog.Record) error {
	message := record.Message

	record.Attrs(func(attr slog.Attr) bool {
		message += fmt.Sprintf(" %v", attr)
		return true
	})

	timestamp := record.Time.Format("15:04:05")
	line := fmt.Sprintf("[%v] %s %v %v", record.Level, l.stringer.String(), timestamp, message)

	if l.trunc > 0 {
		// truncate assumes that the user wants everything on a single line
		// first, truncate by deleting everything after the first newline
		nlp := strings.Index(line, "\n")
		if nlp != -1 {
			line = line[:nlp] + "---"
		}

		// next, truncate by length
		if len(line) > l.trunc {
			line = line[:l.trunc] + "..."
		}
	}

	_, err := fmt.Fprintf(l.w, "%s\n", line)
	return err
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

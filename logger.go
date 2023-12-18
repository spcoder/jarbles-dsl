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
	a        *Assistant
	w        io.WriteCloser
	minLevel slog.Level
	trunc    int
}

func NewLibLogger(a *Assistant) (*slog.Logger, error) {
	err := os.MkdirAll(LogDir(), 0700)
	if err != nil {
		return nil, fmt.Errorf("error while creating log directory: %s: %w", LogDir(), err)
	}

	filename := filepath.Join(LogDir(), "assistants.log")
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

	return slog.New(&LibLogger{a: a, w: logfile, trunc: trunc, minLevel: minLevel}), nil
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
	line := fmt.Sprintf("[%v] (%s) {%s} %v %v", record.Level, l.a.description.Id, l.a.description.Model, timestamp, message)

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

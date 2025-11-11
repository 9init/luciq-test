package logging

import (
	"fmt"
	"go-chat/internal/config"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type Logger struct {
	appName     string
	writer      io.Writer
	extraPrefix string
}

func NewLogger(cfg *config.Config) *Logger {
	appName := cfg.AppName
	logDir := cfg.LogPath

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	logFileName := fmt.Sprintf("%s/%s_%s.log",
		logDir, appName, time.Now().Format("20060102"))

	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	multiWriter := io.MultiWriter(os.Stdout, file)

	return &Logger{
		appName: appName,
		writer:  multiWriter,
	}
}

func (l *Logger) Log(level string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	pc, file, line, ok := runtime.Caller(2)
	funcName := "unknown"
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName = filepath.Base(fn.Name())
		}
		file = filepath.Base(file)
	}

	timestamp := time.Now().Format(time.RFC3339)
	prefix := fmt.Sprintf("[%s %s %s:%d %s] ", level, timestamp, file, line, funcName)

	if l.extraPrefix != "" {
		msg = fmt.Sprintf("[%s] %s", l.extraPrefix, msg)
	}

	_, _ = l.writer.Write(append([]byte(prefix), []byte(msg+"\n")...))
}

// Convenience shortcuts
func (l *Logger) Info(format string, args ...any) {
	l.Log("INFO", format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.Log("ERROR", format, args...)
}

func (l *Logger) Writer() io.Writer {
	return l.writer
}

func (l *Logger) Write(p []byte) (n int, err error) {
	l.Info("%s", string(p))
	return len(p), nil
}

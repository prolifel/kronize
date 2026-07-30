package log

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type RotateWriter struct {
	path    string
	maxSize int64
	mu      sync.Mutex
	file    *os.File
	size    int64
}

func NewRotateWriter(path string, maxSize int64) *RotateWriter {
	return &RotateWriter{path: path, maxSize: maxSize}
}

func (w *RotateWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	if w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotateWriter) open() error {
	dir := filepath.Dir(w.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("log dir: %w", err)
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	w.file = f
	w.size = info.Size()
	return nil
}

func (w *RotateWriter) rotate() error {
	if w.file != nil {
		w.file.Close()
		w.file = nil
	}
	// rename current log to .1, shift older backups
	for i := 9; i >= 0; i-- {
		old := fmt.Sprintf("%s.%d", w.path, i)
		next := fmt.Sprintf("%s.%d", w.path, i+1)
		if _, err := os.Stat(old); err == nil {
			os.Rename(old, next)
		}
	}
	os.Rename(w.path, w.path+".1")
	return w.open()
}

func (w *RotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Init() {
	path := os.Getenv("LOG_FILE")
	if path == "" {
		path = "/data/kronize.log"
	}
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	writer := NewRotateWriter(path, 10*1024*1024) // 10MB

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// redirect standard log to the same writer (unstructured text)
	log.SetOutput(writer)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

package logfile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Writer tees JSONL lines to a log file.
type Writer struct {
	file *os.File
}

// New creates a new log writer under the given logs directory.
// The filename is based on the current timestamp.
func New(logsDir string) (*Writer, error) {
	if err := os.MkdirAll(logsDir, 0o750); err != nil {
		return nil, fmt.Errorf("creating logs dir: %w", err)
	}

	name := time.Now().Format("20060102-150405") + ".jsonl"
	path := filepath.Join(logsDir, name)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating log file: %w", err)
	}

	return &Writer{file: f}, nil
}

// Path returns the path to the log file.
func (w *Writer) Path() string {
	return w.file.Name()
}

// Write implements io.Writer, writing raw bytes to the log file.
func (w *Writer) Write(p []byte) (int, error) {
	return w.file.Write(p)
}

// Close closes the log file.
func (w *Writer) Close() error {
	return w.file.Close()
}

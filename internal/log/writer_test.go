package logfile

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_CreatesLogsDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "logs")

	w, err := New(dir)
	require.NoError(t, err)
	defer w.Close() //nolint:errcheck // deferred close in test, error not actionable

	_, err = os.Stat(dir)
	assert.NoError(t, err, "logs directory should have been created")
}

func TestPath_MatchesTimestampPattern(t *testing.T) {
	w, err := New(t.TempDir())
	require.NoError(t, err)
	defer w.Close() //nolint:errcheck // deferred close in test, error not actionable

	pattern := regexp.MustCompile(`\d{8}-\d{6}\.jsonl$`)
	assert.Regexp(t, pattern, w.Path())
}

func TestWrite_BytesReadableAfterClose(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir)
	require.NoError(t, err)

	payload := []byte(`{"type":"result"}` + "\n")
	n, err := w.Write(payload)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)

	require.NoError(t, w.Close())

	got, err := os.ReadFile(w.Path())
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestClose_NoError(t *testing.T) {
	w, err := New(t.TempDir())
	require.NoError(t, err)

	assert.NoError(t, w.Close())
}

func TestNew_UnwritableDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission check is not meaningful")
	}

	// Create a directory that is not writable.
	parent := t.TempDir()
	locked := filepath.Join(parent, "locked")
	require.NoError(t, os.MkdirAll(locked, 0o500))

	_, err := New(filepath.Join(locked, "logs"))
	assert.Error(t, err)
}

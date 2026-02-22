package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		// valid
		{"specs/feature", false},
		{"custom-specs", false},
		{"a/b/c", false},
		{".", false},

		// absolute paths
		{"/etc/passwd", true},
		{"/absolute/path", true},

		// traversal
		{"..", true},
		{"../outside", true},
		{"a/../../outside", true},
		{"specs/../../../etc", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := validateRelativePath("specs", tt.path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--specs")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

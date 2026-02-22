package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoSlug(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "https with .git",
			input: "https://github.com/owner/repo.git",
			want:  "owner/repo",
		},
		{
			name:  "https without .git",
			input: "https://github.com/owner/repo",
			want:  "owner/repo",
		},
		{
			name:  "https with trailing slash",
			input: "https://github.com/owner/repo/",
			want:  "owner/repo",
		},
		{
			name:  "ssh with .git",
			input: "git@github.com:owner/repo.git",
			want:  "owner/repo",
		},
		{
			name:  "ssh without .git",
			input: "git@github.com:owner/repo",
			want:  "owner/repo",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not-a-url",
			wantErr: true,
		},
		{
			name:    "ssh missing colon path",
			input:   "git@github.com",
			wantErr: true,
		},
		{
			name:    "https host only",
			input:   "https://github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RepoSlug(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

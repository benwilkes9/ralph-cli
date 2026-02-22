package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/auth-flow", "feature-auth-flow"},
		{"main", "main"},
		{"already-clean", "already-clean"},
		{"feat/nested/deep/branch", "feat-nested-deep-branch"},
		{"-leading-hyphen", "leading-hyphen"},
		{"trailing-hyphen-", "trailing-hyphen"},
		{"-both-sides-", "both-sides"},
		{"special!@#chars", "specialchars"},
		{"dots.are.ok", "dots.are.ok"},
		{"under_scores", "under_scores"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranch(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	defaultProtected := []string{"main", "master"}

	tests := []struct {
		branch    string
		protected []string
		want      bool
	}{
		{"main", defaultProtected, true},
		{"master", defaultProtected, true},
		{"Main", defaultProtected, true},
		{"MASTER", defaultProtected, true},
		{"develop", defaultProtected, false},
		{"feature/auth", defaultProtected, false},
		{"main-feature", defaultProtected, false},
		{"develop", []string{"develop", "staging"}, true},
		{"staging", []string{"develop", "staging"}, true},
		{"main", []string{"develop", "staging"}, false},
		{"main", nil, false},
		{"main", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := IsProtectedBranch(tt.branch, tt.protected)
			assert.Equal(t, tt.want, got)
		})
	}
}

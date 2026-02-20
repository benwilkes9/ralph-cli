package git

import "testing"

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
			if got != tt.want {
				t.Errorf("SanitizeBranch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   bool
	}{
		{"main", true},
		{"master", true},
		{"develop", false},
		{"feature/auth", false},
		{"main-feature", false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := IsProtectedBranch(tt.branch)
			if got != tt.want {
				t.Errorf("IsProtectedBranch(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

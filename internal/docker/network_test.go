package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedDomains(t *testing.T) {
	tests := []struct {
		name   string
		extras []string
		want   []string
	}{
		{
			name:   "nil extras returns defaults",
			extras: nil,
			want:   DefaultAllowedDomains,
		},
		{
			name:   "empty extras returns defaults",
			extras: []string{},
			want:   DefaultAllowedDomains,
		},
		{
			name:   "extras appended to defaults",
			extras: []string{"pypi.org", "files.pythonhosted.org"},
			want: append(
				append([]string{}, DefaultAllowedDomains...),
				"pypi.org", "files.pythonhosted.org",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllowedDomains(tt.extras)
			assert.Equal(t, tt.want, got)
		})
	}
}

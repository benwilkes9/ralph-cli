package docker

// DefaultAllowedDomains is the base set of domains allowed through the
// container's network firewall. These are required for Claude Code operation,
// git push/pull, and npm updates.
var DefaultAllowedDomains = []string{
	"api.anthropic.com",
	"claude.ai",
	"github.com",
	"api.github.com",
	"registry.npmjs.org",
}

// AllowedDomains merges the default allowed domains with user-configured extras,
// deduplicating any overlap.
func AllowedDomains(extras []string) []string {
	seen := make(map[string]bool, len(DefaultAllowedDomains)+len(extras))
	all := make([]string, 0, len(DefaultAllowedDomains)+len(extras))
	for _, d := range append(DefaultAllowedDomains, extras...) {
		if !seen[d] {
			seen[d] = true
			all = append(all, d)
		}
	}
	return all
}

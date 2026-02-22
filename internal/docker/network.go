package docker

// DefaultAllowedDomains is the base set of domains allowed through the
// container's network firewall. These are required for Claude Code operation,
// git push/pull, and npm updates.
var DefaultAllowedDomains = []string{
	"api.anthropic.com",
	"github.com",
	"api.github.com",
	"registry.npmjs.org",
}

// AllowedDomains merges the default allowed domains with user-configured extras.
func AllowedDomains(extras []string) []string {
	all := make([]string, 0, len(DefaultAllowedDomains)+len(extras))
	all = append(all, DefaultAllowedDomains...)
	all = append(all, extras...)
	return all
}

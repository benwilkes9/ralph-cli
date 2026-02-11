package scaffold

import (
	"io"
	"strings"

	"github.com/charmbracelet/huh"
)

// PromptOptions configures input/output for interactive prompts.
type PromptOptions struct {
	In         io.Reader
	Out        io.Writer
	Accessible bool
}

// RunPrompts asks the user to confirm or override detected values.
func RunPrompts(info *ProjectInfo, opts *PromptOptions) error {
	var envVars string
	info.CreateSpecs = true // default to creating specs dir

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Run command (how to start the app)").
				Value(&info.RunCmd),
			huh.NewInput().
				Title("Project goal (one sentence)").
				Value(&info.Goal),
			huh.NewConfirm().
				Title("Create .ralph/specs/ directory?").
				Affirmative("Yes").
				Negative("No").
				Value(&info.CreateSpecs),
			huh.NewInput().
				Title("Additional env vars (comma-separated, e.g. DATABASE_URL,REDIS_URL)").
				Value(&envVars),
		),
	).WithAccessible(opts.Accessible)

	if opts.In != nil {
		form = form.WithInput(opts.In)
	}
	if opts.Out != nil {
		form = form.WithOutput(opts.Out)
	}

	if err := form.Run(); err != nil {
		return err //nolint:wrapcheck // propagate huh errors directly
	}

	if envVars != "" {
		for _, v := range strings.Split(envVars, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				info.EnvVars = append(info.EnvVars, v)
			}
		}
	}

	return nil
}

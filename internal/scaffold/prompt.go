package scaffold

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

// PromptOptions configures input/output for interactive prompts.
type PromptOptions struct {
	In         io.Reader
	Out        io.Writer
	Accessible bool
	Branch     string // current git branch; used to show concrete examples in prompts
}

// RunPrompts asks the user to confirm or override detected values.
func RunPrompts(info *ProjectInfo, opts *PromptOptions) error {
	var runChoice, goalChoice, specsChoice string

	form := huh.NewForm(
		// Group 1: Select run command
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(runCmdTitle(info)).
				Options(runCmdOptions(info)...).
				Value(&runChoice),
		),
		// Group 2: Custom run command (shown only if "Type something." selected)
		huh.NewGroup(
			huh.NewInput().
				Title("Run command").
				Value(&info.RunCmd),
		).WithHideFunc(func() bool { return runChoice != customSentinel }),

		// Group 3: Select goal
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What is the ultimate goal for this project? (one sentence describing what it should become)").
				Options(goalOptions(info)...).
				Value(&goalChoice),
		),
		// Group 4: Custom goal (shown only if "Type something." selected)
		huh.NewGroup(
			huh.NewInput().
				Title("Project goal").
				Value(&info.Goal),
		).WithHideFunc(func() bool { return goalChoice != customSentinel }),

		// Group 5: Select specs directory
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Where will your specs live? (stored as <dir>/<branch>/)").
				Options(specsDirOptions(opts.Branch)...).
				Value(&specsChoice),
		),
		// Group 6: Custom specs dir (shown only if "Type something." selected)
		huh.NewGroup(
			huh.NewInput().
				Title("Specs directory").
				Value(&info.SpecsDir),
		).WithHideFunc(func() bool { return specsChoice != customSentinel }),
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

	if runChoice != customSentinel {
		info.RunCmd = runChoice
	}
	if goalChoice != customSentinel {
		info.Goal = goalChoice
	}
	if specsChoice != customSentinel {
		info.SpecsDir = specsChoice
	}

	if err := validateSpecsDir(info.SpecsDir); err != nil {
		return err
	}

	return nil
}

// validateSpecsDir rejects absolute paths and paths that escape the repo root.
func validateSpecsDir(dir string) error {
	if filepath.IsAbs(dir) {
		return fmt.Errorf("specs directory must be a relative path, got %q", dir)
	}
	clean := filepath.Clean(dir)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("specs directory must stay within the repository, got %q", dir)
	}
	return nil
}

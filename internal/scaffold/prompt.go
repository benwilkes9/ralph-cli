package scaffold

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// PromptOptions configures input/output for interactive prompts.
type PromptOptions struct {
	In  io.Reader
	Out io.Writer
}

// RunPrompts asks the user to confirm or override detected values.
func RunPrompts(info *ProjectInfo, opts *PromptOptions) error {
	scanner := bufio.NewScanner(opts.In)

	info.RunCmd = promptWithDefault(scanner, opts.Out,
		"Run command (how to start the app)", info.RunCmd)

	info.Goal = promptWithDefault(scanner, opts.Out,
		"Project goal (one sentence)", info.Goal)

	info.CreateSpecs = promptYesNo(scanner, opts.Out,
		"Create .ralph/specs/ directory?", true)

	extra := promptWithDefault(scanner, opts.Out,
		"Additional env vars (comma-separated, e.g. DATABASE_URL,REDIS_URL)", "")
	if extra != "" {
		for _, v := range strings.Split(extra, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				info.EnvVars = append(info.EnvVars, v)
			}
		}
	}

	return nil
}

//nolint:errcheck // display-only writes to terminal
func promptWithDefault(scanner *bufio.Scanner, w io.Writer, prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprintf(w, "%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Fprintf(w, "%s: ", prompt)
	}
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			return input
		}
	}
	return defaultVal
}

//nolint:errcheck // display-only writes to terminal
func promptYesNo(scanner *bufio.Scanner, w io.Writer, prompt string, defaultYes bool) bool {
	hint := "Y/n"
	if !defaultYes {
		hint = "y/N"
	}
	fmt.Fprintf(w, "%s [%s]: ", prompt, hint)
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
	return defaultYes
}

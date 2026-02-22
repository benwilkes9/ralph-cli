package docker

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

const osDarwin = "darwin"

// RunOptions configures a docker run invocation.
type RunOptions struct {
	ImageTag       string
	Mode           string // "plan" or "build"
	MaxIter        int
	Branch         string
	ProjectDir     string // host project root for bind mount
	PlanFile       string
	SpecsDir       string
	AllowedDomains []string // merged default + extra
	DepsDir        string   // relative path for dep volume overlay (e.g. "node_modules"), empty = none
	ProjectName    string   // for volume naming
}

// Run executes docker run with the given options, attaching stdin/stdout/stderr.
func Run(opts *RunOptions) error {
	return runWithRunner(defaultRunner{}, opts)
}

func runWithRunner(runner CommandRunner, opts *RunOptions) error {
	args := []string{
		"run", "--rm", "-it",
		"--security-opt", "no-new-privileges",
		"--cap-add", "NET_ADMIN",
		"-e", "ANTHROPIC_API_KEY",
		"-e", "GITHUB_PAT",
		"-e", "BRANCH=" + opts.Branch,
		"-e", "PLAN_FILE=" + opts.PlanFile,
		"-e", "SPECS_DIR=" + opts.SpecsDir,
		"-e", "ALLOWED_DOMAINS=" + strings.Join(opts.AllowedDomains, ","),
		"-v", bindMount(opts.ProjectDir, "/workspace/repo"),
	}

	if opts.DepsDir != "" {
		args = append(args,
			"-v", depsVolume(opts.ProjectName)+":/workspace/repo/"+opts.DepsDir,
			"-e", "DEPS_DIR="+opts.DepsDir,
		)
	}

	args = append(args,
		opts.ImageTag,
		"--",
		opts.Mode,
		strconv.Itoa(opts.MaxIter),
	)

	if err := runner.Run("docker", args...); err != nil {
		return fmt.Errorf("docker run: %w", err)
	}
	return nil
}

func bindMount(hostDir, containerDir string) string {
	m := hostDir + ":" + containerDir
	if runtime.GOOS == osDarwin {
		m += ":delegated"
	}
	return m
}

func depsVolume(projectName string) string {
	return "ralph-deps-" + projectName
}

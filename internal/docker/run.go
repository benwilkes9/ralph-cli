package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RunOptions configures a docker run invocation.
type RunOptions struct {
	ImageTag       string
	Mode           string // "plan" or "build"
	MaxIter        int
	Branch         string
	ProjectDir     string // host project root for bind mount
	PlanFile       string
	SpecsDir       string
	AllowedDomains []string   // merged default + extra
	DepsDir        string     // relative path for dep volume overlay (e.g. "node_modules"), empty = none
	ProjectName    string     // for volume naming
	Auth           AuthMethod // which credential to pass into the container
	AdditionalDirs []string   // host paths to additional repos
}

// Run executes docker run with the given options, attaching stdin/stdout/stderr.
func Run(opts *RunOptions) error {
	return runWithRunner(defaultRunner{}, opts)
}

func runWithRunner(runner CommandRunner, opts *RunOptions) error {
	authEnv := "ANTHROPIC_API_KEY"
	if opts.Auth == AuthOAuth {
		authEnv = "CLAUDE_CODE_OAUTH_TOKEN"
	}

	args := []string{
		"run", "--rm", "-it",
		"--security-opt", "no-new-privileges",
		"--cap-add", "NET_ADMIN",
		"-e", authEnv,
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

	for _, dir := range opts.AdditionalDirs {
		args = append(args, "-v", bindMount(dir, "/workspace/"+filepath.Base(dir)))
	}
	if len(opts.AdditionalDirs) > 0 {
		cPaths := make([]string, 0, len(opts.AdditionalDirs))
		for _, dir := range opts.AdditionalDirs {
			cPaths = append(cPaths, "/workspace/"+filepath.Base(dir))
		}
		args = append(args, "-e", "ADDITIONAL_DIRS="+strings.Join(cPaths, ","))
	}

	// If a cross-compiled Linux binary exists alongside the host binary,
	// mount it into the container to override the registry-installed version.
	// This lets local (unpublished) changes take effect inside Docker.
	if linuxBin := findLinuxBinary(); linuxBin != "" {
		args = append(args, "-v", linuxBin+":/usr/local/bin/ralph:ro")
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
	return hostDir + ":" + containerDir
}

func depsVolume(projectName string) string {
	return "ralph-deps-" + projectName
}

// findLinuxBinary returns the path to a cross-compiled Linux ralph binary
// if one exists alongside the currently running executable (e.g. ralph-linux
// next to ralph). Returns "" if not found.
func findLinuxBinary() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	linuxBin := exe + "-linux"
	if _, err := os.Stat(linuxBin); err == nil {
		return linuxBin
	}
	return ""
}

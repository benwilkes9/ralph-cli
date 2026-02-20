package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/benwilkes9/ralph-cli/internal/config"
	"github.com/benwilkes9/ralph-cli/internal/docker"
	"github.com/benwilkes9/ralph-cli/internal/git"
	"github.com/benwilkes9/ralph-cli/internal/loop"
	"github.com/benwilkes9/ralph-cli/internal/scaffold"
	"github.com/benwilkes9/ralph-cli/internal/state"
	"github.com/benwilkes9/ralph-cli/internal/status"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:          "ralph",
		Short:        "Autonomous plan/build iteration using Claude Code",
		Version:      version,
		SilenceUsage: true,
	}

	root.AddCommand(initCmd())
	root.AddCommand(planCmd())
	root.AddCommand(applyCmd())
	root.AddCommand(statusCmd())
	root.AddCommand(loopCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold .ralph/ in current repo",
		RunE: func(_ *cobra.Command, _ []string) error {
			repoRoot, err := git.RepoRoot()
			if err != nil {
				return fmt.Errorf("finding repo root: %w", err)
			}

			info := scaffold.Detect(repoRoot)

			if err := scaffold.RunPrompts(info, &scaffold.PromptOptions{
				In:  os.Stdin,
				Out: os.Stdout,
			}); err != nil {
				return fmt.Errorf("running prompts: %w", err)
			}

			result, err := scaffold.Generate(repoRoot, info)
			if err != nil {
				return fmt.Errorf("generating scaffold: %w", err)
			}

			scaffold.PrintSummary(os.Stdout, result)
			return nil
		},
	}
}

// runParams holds resolved parameters shared by planCmd and applyCmd.
type runParams struct {
	maxVal   int
	branch   string
	planFile string
	specsDir string
	repoRoot string
}

// resolveRunParams extracts flags, resolves the branch, checks protection,
// sanitizes the branch name, loads config, and computes paths.
func resolveRunParams(cmd *cobra.Command) (*runParams, error) {
	maxVal, err := cmd.Flags().GetInt("max")
	if err != nil {
		return nil, fmt.Errorf("reading --max flag: %w", err)
	}
	specsDir, err := cmd.Flags().GetString("specs")
	if err != nil {
		return nil, fmt.Errorf("reading --specs flag: %w", err)
	}

	repoRoot, err := git.RepoRoot()
	if err != nil {
		return nil, fmt.Errorf("finding repo root: %w", err)
	}
	cfg, err := config.Load(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	branch, err := git.Branch()
	if err != nil {
		return nil, fmt.Errorf("getting current branch: %w", err)
	}
	if git.IsProtectedBranch(branch, cfg.ProtectedBranches) {
		return nil, fmt.Errorf("ralph %s must be run on a feature branch, not %q", cmd.Name(), branch)
	}

	sanitized := git.SanitizeBranch(branch)
	if specsDir == "" {
		specsDir = "specs/" + sanitized
	}
	planFile := cfg.PlanPathForBranch(sanitized)

	return &runParams{
		maxVal:   maxVal,
		branch:   branch,
		planFile: planFile,
		specsDir: specsDir,
		repoRoot: repoRoot,
	}, nil
}

func planCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Run planning loop (generates branch-specific plan)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := resolveRunParams(cmd)
			if err != nil {
				return err
			}

			// Ensure specs and plans directories exist.
			if err := os.MkdirAll(filepath.Join(p.repoRoot, p.specsDir), 0o750); err != nil {
				return fmt.Errorf("creating specs dir: %w", err)
			}
			if err := os.MkdirAll(filepath.Join(p.repoRoot, filepath.Dir(p.planFile)), 0o750); err != nil {
				return fmt.Errorf("creating plans dir: %w", err)
			}

			return docker.BuildAndRun("plan", p.maxVal, p.branch, p.planFile, p.specsDir)
		},
	}
	cmd.Flags().IntP("max", "n", 0, "maximum iterations (0 = use config default)")
	cmd.Flags().String("specs", "", "specs directory (default: specs/{branch})")
	return cmd
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run build loop (implements tasks one at a time)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := resolveRunParams(cmd)
			if err != nil {
				return err
			}

			planPath := filepath.Join(p.repoRoot, p.planFile)
			if _, err := os.Stat(planPath); os.IsNotExist(err) {
				return fmt.Errorf("plan file %q not found; run \"ralph plan\" first", p.planFile)
			}

			return docker.BuildAndRun("build", p.maxVal, p.branch, p.planFile, p.specsDir)
		},
	}
	cmd.Flags().IntP("max", "n", 0, "maximum iterations (0 = use config default)")
	cmd.Flags().String("specs", "", "specs directory (default: specs/{branch})")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Progress summary — tasks done, costs, pass/fail",
		RunE: func(_ *cobra.Command, _ []string) error {
			repoRoot, err := git.RepoRoot()
			if err != nil {
				return fmt.Errorf("finding repo root: %w", err)
			}

			cfg, err := config.Load(repoRoot)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			branch, err := git.Branch()
			if err != nil {
				return fmt.Errorf("getting branch: %w", err)
			}

			planPath := cfg.PlanPathForBranch(git.SanitizeBranch(branch))

			tasks, err := status.ParsePlan(planPath)
			if err != nil {
				return fmt.Errorf("parsing plan: %w", err)
			}

			runs, err := status.ParseLogs(".ralph/logs")
			if err != nil {
				return fmt.Errorf("parsing logs: %w", err)
			}

			st, err := state.Load(filepath.Join(repoRoot, state.DefaultPath))
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			status.Render(os.Stdout, cfg.Project, branch, tasks, runs, st.LastRun())
			return nil
		},
	}
}

// loopCmd is the hidden _loop command invoked inside Docker containers.
// Usage: ralph _loop <plan|build> [max_iterations]
func loopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_loop",
		Short:  "Internal: run iteration loop directly (used inside containers)",
		Hidden: true,
		Args:   cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			var mode loop.Mode
			switch args[0] {
			case "plan":
				mode = loop.ModePlan
			case "build":
				mode = loop.ModeBuild
			default:
				return fmt.Errorf("unknown mode: %s (expected plan or build)", args[0])
			}

			var maxIter int
			if len(args) > 1 {
				v, err := strconv.Atoi(args[1])
				if err != nil {
					return fmt.Errorf("invalid max_iterations: %w", err)
				}
				maxIter = v
			}

			return runLoop(mode, maxIter)
		},
	}
	return cmd
}

func runLoop(mode loop.Mode, maxFlag int) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	var phase config.PhaseConfig
	if mode == loop.ModePlan {
		phase = cfg.Phases.Plan
	} else {
		phase = cfg.Phases.Build
	}

	maxIterations := phase.MaxIterations
	if maxFlag > 0 {
		maxIterations = maxFlag
	}

	branch, err := git.Branch()
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	// Plan file and specs dir are passed from the host via env vars.
	planFile := os.Getenv("PLAN_FILE")
	if planFile == "" {
		planFile = cfg.PlanPathForBranch(git.SanitizeBranch(branch))
	}
	specsDir := os.Getenv("SPECS_DIR")
	if specsDir == "" {
		specsDir = "specs/" + git.SanitizeBranch(branch)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	opts := &loop.Options{
		Mode:          mode,
		PromptFile:    phase.Prompt,
		MaxIterations: maxIterations,
		FreshContext:  phase.FreshContext,
		LogsDir:       "logs",
		Branch:        branch,
		StateFile:     state.DefaultPath,
		PlanFile:      planFile,
		SpecsDir:      specsDir,
	}

	loopErr := loop.Run(ctx, opts, os.Stdout)
	stop()

	if ctx.Err() != nil {
		os.Exit(130)
	}

	if loopErr != nil {
		return fmt.Errorf("running loop: %w", loopErr)
	}
	return nil
}

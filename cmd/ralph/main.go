package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/benmyles/ralph-cli/internal/config"
	"github.com/benmyles/ralph-cli/internal/git"
	"github.com/benmyles/ralph-cli/internal/loop"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "ralph",
		Short:   "Autonomous plan/build iteration using Claude Code",
		Version: version,
	}

	root.AddCommand(initCmd())
	root.AddCommand(planCmd())
	root.AddCommand(applyCmd())
	root.AddCommand(statusCmd())

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
			fmt.Println("ralph init: not yet implemented")
			return nil
		},
	}
}

func planCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Run planning loop (generates .ralph/IMPLEMENTATION_PLAN.md)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			maxVal, err := cmd.Flags().GetInt("max")
			if err != nil {
				return fmt.Errorf("reading --max flag: %w", err)
			}
			return runLoop(loop.ModePlan, maxVal)
		},
	}
	cmd.Flags().IntP("max", "n", 0, "maximum iterations (0 = use config default)")
	return cmd
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run build loop (implements tasks one at a time)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			maxVal, err := cmd.Flags().GetInt("max")
			if err != nil {
				return fmt.Errorf("reading --max flag: %w", err)
			}
			return runLoop(loop.ModeBuild, maxVal)
		},
	}
	cmd.Flags().IntP("max", "n", 0, "maximum iterations (0 = use config default)")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Progress summary — tasks done, costs, pass/fail",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("ralph status: not yet implemented")
			return nil
		},
	}
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	opts := &loop.Options{
		Mode:          mode,
		PromptFile:    phase.Prompt,
		MaxIterations: maxIterations,
		FreshContext:  phase.FreshContext,
		LogsDir:       "logs",
		Branch:        branch,
	}

	loopErr := loop.Run(ctx, opts, os.Stdout)
	stop()

	if ctx.Err() != nil {
		os.Exit(130)
	}

	return loopErr
}

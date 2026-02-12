package scaffold

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

const customSentinel = "_custom_"

// option creates a huh select option with a multiline key showing
// the value and a description underneath.
func option(value, description string) huh.Option[string] {
	key := fmt.Sprintf("%s\n    %s", value, description)
	return huh.NewOption(key, value)
}

// customOption returns the "Type something." sentinel option.
func customOption() huh.Option[string] {
	return huh.NewOption("Type something.", customSentinel)
}

// runCmdOptions returns suggested run commands based on detected project info.
func runCmdOptions(info *ProjectInfo) []huh.Option[string] {
	var opts []huh.Option[string]

	name := info.ProjectName

	switch info.PackageManager { //nolint:exhaustive // unknown falls through to custom-only
	case PmUV:
		opts = append(opts,
			option(fmt.Sprintf("uv run uvicorn %s.main:app", name), "Standard uvicorn startup for FastAPI"),
			option(fmt.Sprintf("uv run python -m %s", name), "Run as a Python module"),
		)
	case PmPoetry:
		opts = append(opts,
			option(fmt.Sprintf("poetry run uvicorn %s.main:app", name), "Standard uvicorn startup for FastAPI"),
			option(fmt.Sprintf("poetry run python -m %s", name), "Run as a Python module"),
		)
	case PmNPM:
		opts = append(opts,
			option("npm start", "Run the start script from package.json"),
			option("npm run dev", "Run the dev script from package.json"),
		)
	case PmYarn:
		opts = append(opts,
			option("yarn start", "Run the start script from package.json"),
			option("yarn dev", "Run the dev script from package.json"),
		)
	case PmPNPM:
		opts = append(opts,
			option("pnpm start", "Run the start script from package.json"),
			option("pnpm dev", "Run the dev script from package.json"),
		)
	case PmGo:
		opts = append(opts,
			option(fmt.Sprintf("go run ./cmd/%s", name), "Run the main package"),
			option("go run .", "Run from the project root"),
		)
	case PmCargo:
		opts = append(opts,
			option("cargo run", "Build and run the default binary"),
		)
	}

	opts = append(opts, customOption())
	return opts
}

// goalOptions returns suggested project goals based on detected language.
func goalOptions(info *ProjectInfo) []huh.Option[string] {
	var opts []huh.Option[string]

	switch info.Language { //nolint:exhaustive // unknown falls through to generic
	case LangPython:
		opts = append(opts,
			option("Production-ready REST API", "A complete, well-tested REST API with FastAPI and async SQLite"),
		)
	case LangNode:
		opts = append(opts,
			option("Production-ready web application", "A full-stack web app with TypeScript and comprehensive tests"),
		)
	case LangGo:
		opts = append(opts,
			option("Production-ready CLI tool", "A robust CLI with comprehensive tests and documentation"),
		)
	case LangRust:
		opts = append(opts,
			option("Production-ready system tool", "A performant system utility with comprehensive tests"),
		)
	}

	opts = append(opts,
		option("Learning spike / reference", "A reference implementation for exploring patterns"),
		customOption(),
	)
	return opts
}

// runCmdTitle returns a context-aware title for the run command select.
func runCmdTitle(info *ProjectInfo) string {
	example := ""
	switch info.PackageManager { //nolint:exhaustive // unknown gets generic title
	case PmUV:
		example = fmt.Sprintf("uv run uvicorn %s.main:app --reload", info.ProjectName)
	case PmPoetry:
		example = fmt.Sprintf("poetry run uvicorn %s.main:app --reload", info.ProjectName)
	case PmNPM:
		example = "npm start"
	case PmYarn:
		example = "yarn start"
	case PmPNPM:
		example = "pnpm start"
	case PmGo:
		example = fmt.Sprintf("go run ./cmd/%s", info.ProjectName)
	case PmCargo:
		example = "cargo run"
	}

	if example != "" {
		return fmt.Sprintf("How do you start the application? (e.g. %s)", example)
	}
	return "How do you start the application?"
}

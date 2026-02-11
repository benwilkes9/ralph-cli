package scaffold

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templateFS embed.FS

// funcMap provides helper functions available in all templates.
var funcMap = template.FuncMap{
	"str": func(v any) string { return fmt.Sprintf("%v", v) },
}

// templateMapping maps template paths (under templates/) to output paths (relative to repoRoot).
var templateMapping = []struct {
	tmpl   string
	output string
}{
	{"templates/config.yaml.tmpl", ".ralph/config.yaml"},
	{"templates/agents.md.tmpl", "AGENTS.md"},
	{"templates/prompts/plan.md.tmpl", ".ralph/prompts/plan.md"},
	{"templates/prompts/build.md.tmpl", ".ralph/prompts/build.md"},
	{"templates/docker/Dockerfile.tmpl", ".ralph/docker/Dockerfile"},
	{"templates/docker/entrypoint.sh.tmpl", ".ralph/docker/entrypoint.sh"},
	{"templates/docker/dockerignore.tmpl", ".ralph/docker/.dockerignore"},
	{"templates/env.example.tmpl", ".env.example"},
}

// gitignoreEntries are lines to append to .gitignore idempotently.
var gitignoreEntries = []string{
	".ralph/logs/",
	".ralph/state.json",
	".env",
}

// GenerateResult tracks which files were created or skipped.
type GenerateResult struct {
	Created []string
	Skipped []string
}

// Generate renders all templates into the repo, skipping existing files.
func Generate(repoRoot string, info *ProjectInfo) (*GenerateResult, error) {
	result := &GenerateResult{}

	for _, mapping := range templateMapping {
		outputPath := filepath.Join(repoRoot, mapping.output)

		if fileExists(outputPath) {
			result.Skipped = append(result.Skipped, mapping.output)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", mapping.output, err)
		}

		tmplContent, err := templateFS.ReadFile(mapping.tmpl)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", mapping.tmpl, err)
		}

		tmpl, err := template.New(filepath.Base(mapping.tmpl)).Funcs(funcMap).Parse(string(tmplContent))
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", mapping.tmpl, err)
		}

		if err := renderToFile(outputPath, tmpl, info); err != nil {
			return nil, fmt.Errorf("rendering %s: %w", mapping.output, err)
		}

		// Make entrypoint.sh executable.
		if strings.HasSuffix(mapping.output, "entrypoint.sh") {
			if err := os.Chmod(outputPath, 0o750); err != nil { //nolint:gosec // entrypoint needs exec bit
				return nil, fmt.Errorf("chmod %s: %w", mapping.output, err)
			}
		}

		result.Created = append(result.Created, mapping.output)
	}

	if info.CreateSpecs {
		specsDir := filepath.Join(repoRoot, ".ralph", "specs")
		gitkeep := filepath.Join(specsDir, ".gitkeep")
		if fileExists(gitkeep) {
			result.Skipped = append(result.Skipped, ".ralph/specs/.gitkeep")
		} else {
			if err := os.MkdirAll(specsDir, 0o750); err != nil {
				return nil, fmt.Errorf("creating specs dir: %w", err)
			}
			if err := os.WriteFile(gitkeep, nil, 0o600); err != nil {
				return nil, fmt.Errorf("creating .gitkeep: %w", err)
			}
			result.Created = append(result.Created, ".ralph/specs/.gitkeep")
		}
	}

	if err := appendGitignore(repoRoot, result); err != nil {
		return nil, err
	}

	return result, nil
}

func renderToFile(path string, tmpl *template.Template, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	execErr := tmpl.Execute(f, data)
	closeErr := f.Close()
	if execErr != nil {
		return execErr
	}
	return closeErr
}

func appendGitignore(repoRoot string, result *GenerateResult) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	}

	var toAdd []string
	for _, entry := range gitignoreEntries {
		if !containsLine(existing, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening .gitignore: %w", err)
	}
	defer f.Close() //nolint:errcheck // best-effort close on append

	if existing != "" && !strings.HasSuffix(existing, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("writing to .gitignore: %w", err)
		}
	}

	for _, entry := range toAdd {
		if _, err := fmt.Fprintln(f, entry); err != nil {
			return fmt.Errorf("writing to .gitignore: %w", err)
		}
	}

	result.Created = append(result.Created, ".gitignore (appended)")
	return nil
}

func containsLine(content, line string) bool {
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == strings.TrimSpace(line) {
			return true
		}
	}
	return false
}

// PrintSummary displays which files were created and skipped.
func PrintSummary(w io.Writer, result *GenerateResult) {
	printLine(w, "")
	for _, f := range result.Created {
		printLine(w, "  created  "+f)
	}
	for _, f := range result.Skipped {
		printLine(w, "  exists   "+f)
	}
	printLine(w, "")
	printLine(w, "Next steps:")
	printLine(w, "  1. Edit .env with your API keys")
	printLine(w, "  2. Review .ralph/config.yaml")
	printLine(w, "  3. Run: ralph plan")
}

func printLine(w io.Writer, s string) {
	fmt.Fprintln(w, s) //nolint:errcheck // display-only
}

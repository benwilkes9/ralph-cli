package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_CreatesAllFiles(t *testing.T) {
	dir := t.TempDir()
	info := &ProjectInfo{
		ProjectName:     "test-project",
		Language:        LangPython,
		LanguageVersion: "3.12",
		PackageManager:  PmUV,
		InstallCmd:      "uv sync --all-extras",
		TestCmd:         "uv run pytest",
		TypecheckCmd:    "uv run pyright",
		LintCmd:         "uv run ruff check",
		RunCmd:          "uv run uvicorn app:app",
		Goal:            "Build a REST API",
		BaseImage:       "node:22-bookworm",
	}

	result, err := Generate(dir, info)
	if err != nil {
		t.Fatal(err)
	}

	expectedFiles := []string{
		".ralph/config.yaml",
		"AGENTS.md",
		".ralph/prompts/plan.md",
		".ralph/prompts/build.md",
		".ralph/docker/Dockerfile",
		".ralph/docker/entrypoint.sh",
		".ralph/docker/.dockerignore",
		".env.example",
		"specs/.gitkeep",
		".ralph/plans/.gitkeep",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	if len(result.Created) < len(expectedFiles) {
		t.Errorf("expected at least %d created files, got %d: %v", len(expectedFiles), len(result.Created), result.Created)
	}
}

func TestGenerate_ConfigContent(t *testing.T) {
	dir := t.TempDir()
	info := &ProjectInfo{
		ProjectName:     "myapp",
		Language:        LangGo,
		LanguageVersion: "1.25.7",
		PackageManager:  PmGo,
		InstallCmd:      "go mod download",
		TestCmd:         "go test ./...",
		LintCmd:         "golangci-lint run ./...",
		BaseImage:       "node:22-bookworm",
	}

	_, err := Generate(dir, info)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".ralph", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, `project: "myapp"`) {
		t.Error("config should contain project name")
	}
	if !strings.Contains(s, `test: "go test ./..."`) {
		t.Error("config should contain test command")
	}
	if !strings.Contains(s, `lint: "golangci-lint run ./..."`) {
		t.Error("config should contain lint command")
	}
}

func TestGenerate_SkipsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	info := &ProjectInfo{
		ProjectName:    "test-project",
		Language:       LangPython,
		PackageManager: PmUV,
		InstallCmd:     "uv sync",
		TestCmd:        "uv run pytest",
		BaseImage:      "node:22-bookworm",
	}

	result1, err := Generate(dir, info)
	if err != nil {
		t.Fatal(err)
	}
	if len(result1.Skipped) != 0 {
		t.Errorf("first run should skip nothing, got %v", result1.Skipped)
	}

	result2, err := Generate(dir, info)
	if err != nil {
		t.Fatal(err)
	}

	if len(result2.Created) != 0 {
		t.Errorf("second run should create nothing, got %v", result2.Created)
	}
	if len(result2.Skipped) < 10 {
		t.Errorf("second run should skip at least 10 files, got %d: %v", len(result2.Skipped), result2.Skipped)
	}
}

func TestGenerate_GitignoreIdempotent(t *testing.T) {
	dir := t.TempDir()
	info := &ProjectInfo{
		ProjectName:    "test-project",
		Language:       LangPython,
		PackageManager: PmUV,
		InstallCmd:     "uv sync",
		TestCmd:        "uv run pytest",
		BaseImage:      "node:22-bookworm",
	}

	if _, err := Generate(dir, info); err != nil {
		t.Fatal(err)
	}

	content1, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Generate(dir, info); err != nil {
		t.Fatal(err)
	}
	content2, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(content1, content2) {
		t.Errorf("gitignore should be idempotent:\nfirst:\n%s\nsecond:\n%s", content1, content2)
	}
}

func TestGenerate_GitignoreAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	existing := "node_modules/\n*.pyc\n"
	writeFile(t, dir, ".gitignore", existing)

	info := &ProjectInfo{
		ProjectName:    "test-project",
		Language:       LangNode,
		PackageManager: PmNPM,
		InstallCmd:     "npm install",
		TestCmd:        "npm test",
		BaseImage:      "node:22-bookworm",
	}

	if _, err := Generate(dir, info); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.HasPrefix(s, existing) {
		t.Error("existing .gitignore content should be preserved")
	}
	if !strings.Contains(s, ".ralph/logs/") {
		t.Error("should append .ralph/logs/")
	}
	if !strings.Contains(s, ".env") {
		t.Error("should append .env")
	}
}

func TestGenerate_EntrypointIsExecutable(t *testing.T) {
	dir := t.TempDir()
	info := &ProjectInfo{
		ProjectName:    "test-project",
		Language:       LangPython,
		PackageManager: PmUV,
		InstallCmd:     "uv sync",
		TestCmd:        "uv run pytest",
		BaseImage:      "node:22-bookworm",
	}

	if _, err := Generate(dir, info); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(filepath.Join(dir, ".ralph", "docker", "entrypoint.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&0o111 == 0 {
		t.Error("entrypoint.sh should be executable")
	}
}

func TestGenerate_DockerfileContent(t *testing.T) {
	dir := t.TempDir()
	info := &ProjectInfo{
		ProjectName:     "test-project",
		Language:        LangPython,
		LanguageVersion: "3.12",
		PackageManager:  PmUV,
		InstallCmd:      "uv sync --all-extras",
		TestCmd:         "uv run pytest",
		BaseImage:       "node:22-bookworm",
	}

	if _, err := Generate(dir, info); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".ralph", "docker", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, "FROM node:22-bookworm") {
		t.Error("Dockerfile should use base image")
	}
	if !strings.Contains(s, "uv python install 3.12") {
		t.Error("Dockerfile should install Python version")
	}
	if !strings.Contains(s, "ralph") {
		t.Error("Dockerfile should install ralph CLI")
	}
}

func TestPrintSummary(t *testing.T) {
	result := &GenerateResult{
		Created: []string{".ralph/config.yaml", "AGENTS.md"},
		Skipped: []string{".env.example"},
	}

	var buf bytes.Buffer
	PrintSummary(&buf, result)

	output := buf.String()
	if !strings.Contains(output, "created  .ralph/config.yaml") {
		t.Error("should show created files")
	}
	if !strings.Contains(output, "exists   .env.example") {
		t.Error("should show skipped files")
	}
	if !strings.Contains(output, "Next steps") {
		t.Error("should show next steps")
	}
}

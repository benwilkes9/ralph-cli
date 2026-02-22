package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

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
		assert.FileExists(t, filepath.Join(dir, f))
	}

	assert.GreaterOrEqual(t, len(result.Created), len(expectedFiles))
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
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, ".ralph", "config.yaml"))
	require.NoError(t, err)
	s := string(content)

	assert.Contains(t, s, `project: "myapp"`)
	assert.Contains(t, s, `test: "go test ./..."`)
	assert.Contains(t, s, `lint: "golangci-lint run ./..."`)
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
	require.NoError(t, err)
	assert.Empty(t, result1.Skipped)

	result2, err := Generate(dir, info)
	require.NoError(t, err)

	assert.Empty(t, result2.Created)
	assert.GreaterOrEqual(t, len(result2.Skipped), 10)
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

	_, err := Generate(dir, info)
	require.NoError(t, err)

	content1, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)

	_, err = Generate(dir, info)
	require.NoError(t, err)

	content2, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)

	assert.Equal(t, content1, content2)
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

	_, err := Generate(dir, info)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	s := string(content)

	assert.True(t, strings.HasPrefix(s, existing), "existing .gitignore content should be preserved")
	assert.Contains(t, s, ".ralph/logs/")
	assert.Contains(t, s, ".env")
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

	_, err := Generate(dir, info)
	require.NoError(t, err)

	fi, err := os.Stat(filepath.Join(dir, ".ralph", "docker", "entrypoint.sh"))
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&0o111, "entrypoint.sh should be executable")
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

	_, err := Generate(dir, info)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, ".ralph", "docker", "Dockerfile"))
	require.NoError(t, err)
	s := string(content)

	assert.Contains(t, s, "FROM node:22-bookworm")
	assert.Contains(t, s, "uv python install 3.12")
	assert.Contains(t, s, "ralph")
}

func TestPrintSummary(t *testing.T) {
	result := &GenerateResult{
		Created: []string{".ralph/config.yaml", "AGENTS.md"},
		Skipped: []string{".env.example"},
	}

	var buf bytes.Buffer
	PrintSummary(&buf, result)

	output := buf.String()
	assert.Contains(t, output, "created  .ralph/config.yaml")
	assert.Contains(t, output, "exists   .env.example")
	assert.Contains(t, output, "Next steps")
}

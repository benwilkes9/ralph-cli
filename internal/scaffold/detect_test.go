package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_Python_UV(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")
	writeFile(t, dir, ".python-version", "3.13")
	mkdirAll(t, filepath.Join(dir, "src"))
	mkdirAll(t, filepath.Join(dir, "tests"))

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "python")
	assertEqual(t, "PackageManager", string(info.PackageManager), "uv")
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "3.13")
	assertEqual(t, "InstallCmd", info.InstallCmd, "uv sync --all-extras")
	assertEqual(t, "TestCmd", info.TestCmd, "uv run pytest")
	assertEqual(t, "TypecheckCmd", info.TypecheckCmd, "uv run pyright")
	assertEqual(t, "LintCmd", info.LintCmd, "uv run ruff check")

	if len(info.SourceDirs) == 0 || info.SourceDirs[0] != "src" {
		t.Errorf("expected SourceDirs to contain 'src', got %v", info.SourceDirs)
	}
	if len(info.TestDirs) == 0 || info.TestDirs[0] != "tests" {
		t.Errorf("expected TestDirs to contain 'tests', got %v", info.TestDirs)
	}
}

func TestDetect_Python_Poetry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "poetry.lock", "")

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "python")
	assertEqual(t, "PackageManager", string(info.PackageManager), "poetry")
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "3.12") // default
	assertEqual(t, "InstallCmd", info.InstallCmd, "poetry install")
}

func TestDetect_Go(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/test\n\ngo 1.25.7\n")
	writeFile(t, dir, "go.sum", "")
	mkdirAll(t, filepath.Join(dir, "cmd"))
	mkdirAll(t, filepath.Join(dir, "internal"))
	writeFile(t, dir, "Makefile", "")

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "go")
	assertEqual(t, "PackageManager", string(info.PackageManager), "go")
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "1.25.7")
	assertEqual(t, "InstallCmd", info.InstallCmd, "go mod download")
	assertEqual(t, "TestCmd", info.TestCmd, "go test ./...")
	if !info.HasMakefile {
		t.Error("expected HasMakefile to be true")
	}
	if len(info.SourceDirs) < 2 {
		t.Errorf("expected at least cmd and internal in SourceDirs, got %v", info.SourceDirs)
	}
}

func TestDetect_Node_NPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package-lock.json", "{}")
	writeFile(t, dir, ".nvmrc", "20")

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "node")
	assertEqual(t, "PackageManager", string(info.PackageManager), "npm")
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "20")
	assertEqual(t, "InstallCmd", info.InstallCmd, "npm install")
}

func TestDetect_Node_Yarn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "yarn.lock", "")

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "node")
	assertEqual(t, "PackageManager", string(info.PackageManager), "yarn")
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "22") // default
}

func TestDetect_Node_PNPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-lock.yaml", "")

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "node")
	assertEqual(t, "PackageManager", string(info.PackageManager), "pnpm")
}

func TestDetect_Rust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"")
	writeFile(t, dir, "Cargo.lock", "")
	writeFile(t, dir, "rust-toolchain", "nightly")

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "rust")
	assertEqual(t, "PackageManager", string(info.PackageManager), "cargo")
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "nightly")
	assertEqual(t, "InstallCmd", info.InstallCmd, "cargo build")
	assertEqual(t, "TestCmd", info.TestCmd, "cargo test")
}

func TestDetect_Unknown(t *testing.T) {
	dir := t.TempDir()

	info := Detect(dir)

	assertEqual(t, "Language", string(info.Language), "unknown")
	assertEqual(t, "PackageManager", string(info.PackageManager), "unknown")
	assertEqual(t, "ProjectName", info.ProjectName, filepath.Base(dir))
}

func TestDetect_ProjectName(t *testing.T) {
	dir := t.TempDir()
	info := Detect(dir)
	assertEqual(t, "ProjectName", info.ProjectName, filepath.Base(dir))
}

func TestDetect_BaseImage(t *testing.T) {
	dir := t.TempDir()
	info := Detect(dir)
	assertEqual(t, "BaseImage", info.BaseImage, "node:22-bookworm")
}

func TestDetect_GoVersionParsing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/mymod\n\ngo 1.23.4\n\nrequire (\n)")
	writeFile(t, dir, "go.sum", "")

	info := Detect(dir)
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "1.23.4")
}

func TestDetect_PythonDefaultVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")

	info := Detect(dir)
	assertEqual(t, "LanguageVersion", info.LanguageVersion, "3.12")
}

func TestSourceDirsList(t *testing.T) {
	info := &ProjectInfo{SourceDirs: []string{"src", "lib"}}
	got := info.SourceDirsList()
	if got != "src/, lib/" {
		t.Errorf("SourceDirsList() = %q, want %q", got, "src/, lib/")
	}
}

func TestSourceDirsList_Empty(t *testing.T) {
	info := &ProjectInfo{}
	got := info.SourceDirsList()
	if got != "src/" {
		t.Errorf("SourceDirsList() = %q, want %q", got, "src/")
	}
}

func TestTestDirsList_Empty(t *testing.T) {
	info := &ProjectInfo{}
	got := info.TestDirsList()
	if got != "tests/" {
		t.Errorf("TestDirsList() = %q, want %q", got, "tests/")
	}
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

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

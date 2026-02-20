package scaffold

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// safeVersion matches version strings that are safe to interpolate into shell
// commands and Dockerfile instructions. Allows alphanumeric, dots, hyphens, and
// plus signs (e.g. "1.25.7", "3.12", "nightly", "nightly-2024-01-01").
var safeVersion = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.\-+]*$`)

// Language represents a detected programming language.
type Language string

// Supported languages.
const (
	LangPython  Language = "python"
	LangNode    Language = "node"
	LangGo      Language = "go"
	LangRust    Language = "rust"
	LangUnknown Language = "unknown"
)

// PackageManager represents a detected package manager.
type PackageManager string

// Supported package managers.
const (
	PmUV      PackageManager = "uv"
	PmPoetry  PackageManager = "poetry"
	PmNPM     PackageManager = "npm"
	PmYarn    PackageManager = "yarn"
	PmPNPM    PackageManager = "pnpm"
	PmGo      PackageManager = "go"
	PmCargo   PackageManager = "cargo"
	PmUnknown PackageManager = "unknown"
)

// ProjectInfo holds detected and user-provided project metadata used to render templates.
type ProjectInfo struct {
	ProjectName     string
	Language        Language
	LanguageVersion string
	PackageManager  PackageManager

	InstallCmd   string
	TestCmd      string
	TypecheckCmd string
	LintCmd      string
	RunCmd       string
	Goal         string

	SourceDirs  []string
	TestDirs    []string
	BaseImage   string
	HasMakefile bool
}

// lockFileSignals maps lock/config files to their language and package manager.
var lockFileSignals = []struct {
	file string
	lang Language
	pm   PackageManager
}{
	{"uv.lock", LangPython, PmUV},
	{"poetry.lock", LangPython, PmPoetry},
	{"go.sum", LangGo, PmGo},
	{"go.mod", LangGo, PmGo},
	{"package-lock.json", LangNode, PmNPM},
	{"yarn.lock", LangNode, PmYarn},
	{"pnpm-lock.yaml", LangNode, PmPNPM},
	{"Cargo.lock", LangRust, PmCargo},
	{"Cargo.toml", LangRust, PmCargo},
}

// Detect inspects the repo at repoRoot and returns a ProjectInfo with sensible defaults.
func Detect(repoRoot string) *ProjectInfo {
	info := &ProjectInfo{
		ProjectName:    filepath.Base(repoRoot),
		Language:       LangUnknown,
		PackageManager: PmUnknown,
		BaseImage:      "node:22-bookworm",
	}

	for _, sig := range lockFileSignals {
		if fileExists(filepath.Join(repoRoot, sig.file)) {
			info.Language = sig.lang
			info.PackageManager = sig.pm
			break
		}
	}

	info.LanguageVersion = detectLanguageVersion(repoRoot, info.Language)
	applyEcosystemDefaults(info)
	info.SourceDirs = detectDirs(repoRoot, []string{"src", "lib", "app", "cmd", "internal"})
	info.TestDirs = detectDirs(repoRoot, []string{"tests", "test", "__tests__"})
	info.HasMakefile = fileExists(filepath.Join(repoRoot, "Makefile"))

	return info
}

func detectLanguageVersion(repoRoot string, lang Language) string {
	var v string
	switch lang { //nolint:exhaustive // unknown has no version
	case LangGo:
		v = readGoVersion(filepath.Join(repoRoot, "go.mod"))
	case LangPython:
		if fv := readFirstLine(filepath.Join(repoRoot, ".python-version")); fv != "" {
			v = fv
		} else {
			v = "3.12"
		}
	case LangNode:
		if fv := readFirstLine(filepath.Join(repoRoot, ".nvmrc")); fv != "" {
			v = fv
		} else {
			v = "22"
		}
	case LangRust:
		if fv := readFirstLine(filepath.Join(repoRoot, "rust-toolchain")); fv != "" {
			v = fv
		} else {
			v = "stable"
		}
	default:
		return ""
	}
	return sanitizeVersion(v)
}

// sanitizeVersion returns the version string only if it matches the safe
// pattern. If the version contains shell metacharacters or other unsafe
// content, it returns an empty string to prevent injection.
func sanitizeVersion(v string) string {
	if v == "" {
		return ""
	}
	if !safeVersion.MatchString(v) {
		return ""
	}
	return v
}

func applyEcosystemDefaults(info *ProjectInfo) {
	switch info.PackageManager { //nolint:exhaustive // unknown has no defaults
	case PmUV:
		info.InstallCmd = "uv sync --all-extras"
		info.TestCmd = "uv run pytest"
		info.TypecheckCmd = "uv run pyright"
		info.LintCmd = "uv run ruff check"
	case PmPoetry:
		info.InstallCmd = "poetry install"
		info.TestCmd = "poetry run pytest"
		info.TypecheckCmd = "poetry run pyright"
		info.LintCmd = "poetry run ruff check"
	case PmNPM:
		info.InstallCmd = "npm install"
		info.TestCmd = "npm test"
		info.TypecheckCmd = "npx tsc --noEmit"
		info.LintCmd = "npm run lint"
	case PmYarn:
		info.InstallCmd = "yarn install"
		info.TestCmd = "yarn test"
		info.TypecheckCmd = "yarn tsc --noEmit"
		info.LintCmd = "yarn lint"
	case PmPNPM:
		info.InstallCmd = "pnpm install"
		info.TestCmd = "pnpm test"
		info.TypecheckCmd = "pnpm tsc --noEmit"
		info.LintCmd = "pnpm lint"
	case PmGo:
		info.InstallCmd = "go mod download"
		info.TestCmd = "go test ./..."
		info.TypecheckCmd = ""
		info.LintCmd = "golangci-lint run ./..."
	case PmCargo:
		info.InstallCmd = "cargo build"
		info.TestCmd = "cargo test"
		info.TypecheckCmd = ""
		info.LintCmd = "cargo clippy"
	}
}

func readGoVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimPrefix(line, "go ")
		}
	}
	return ""
}

func readFirstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func detectDirs(repoRoot string, candidates []string) []string {
	var found []string
	for _, dir := range candidates {
		info, err := os.Stat(filepath.Join(repoRoot, dir))
		if err == nil && info.IsDir() {
			found = append(found, dir)
		}
	}
	return found
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SourceDirsList returns source dirs as a comma-separated string for display.
func (p *ProjectInfo) SourceDirsList() string {
	return joinOrDefault(p.SourceDirs, "src/")
}

// TestDirsList returns test dirs as a comma-separated string for display.
func (p *ProjectInfo) TestDirsList() string {
	return joinOrDefault(p.TestDirs, "tests/")
}

func joinOrDefault(items []string, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%s/", strings.TrimSuffix(item, "/"))
	}
	return strings.Join(quoted, ", ")
}

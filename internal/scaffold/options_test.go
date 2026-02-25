package scaffold

import (
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertOptionValue(t *testing.T, opt huh.Option[string], want string) {
	t.Helper()
	assert.Equal(t, want, opt.Value)
}

func TestRunCmdOptions_UV(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmUV}
	opts := runCmdOptions(info)

	// 2 suggestions + custom
	require.Len(t, opts, 3)
	assertOptionValue(t, opts[0], "uv run uvicorn myapp.main:app")
	assertOptionValue(t, opts[1], "uv run python -m myapp")
	assertOptionValue(t, opts[len(opts)-1], customSentinel)
}

func TestRunCmdOptions_Go(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmGo}
	opts := runCmdOptions(info)

	require.Len(t, opts, 3)
	assertOptionValue(t, opts[0], "go run ./cmd/myapp")
	assertOptionValue(t, opts[1], "go run .")
}

func TestRunCmdOptions_NPM(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmNPM}
	opts := runCmdOptions(info)

	require.Len(t, opts, 3)
	assertOptionValue(t, opts[0], "npm start")
	assertOptionValue(t, opts[1], "npm run dev")
}

func TestRunCmdOptions_Unknown(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmUnknown}
	opts := runCmdOptions(info)

	require.Len(t, opts, 1, "expected 1 option (custom only)")
	assertOptionValue(t, opts[0], customSentinel)
}

func TestGoalOptions_Python(t *testing.T) {
	info := &ProjectInfo{Language: LangPython}
	opts := goalOptions(info)

	// 1 language-specific + learning spike + custom
	require.Len(t, opts, 3)
	assertOptionValue(t, opts[len(opts)-1], customSentinel)
}

func TestGoalOptions_Unknown(t *testing.T) {
	info := &ProjectInfo{Language: LangUnknown}
	opts := goalOptions(info)

	// learning spike + custom
	require.Len(t, opts, 2)
	assertOptionValue(t, opts[len(opts)-1], customSentinel)
}

func TestRunCmdTitle_WithExample(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmGo}
	title := runCmdTitle(info)
	assert.Equal(t, "How do you start the application? (e.g. go run ./cmd/myapp)", title)
}

func TestRunCmdTitle_Unknown(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmUnknown}
	title := runCmdTitle(info)
	assert.Equal(t, "How do you start the application?", title)
}

func TestSpecsDirOptions_WithBranch(t *testing.T) {
	opts := specsDirOptions("my-feature")

	require.Len(t, opts, 3)
	assertOptionValue(t, opts[0], "specs")
	assertOptionValue(t, opts[1], "docs/specs")
	assertOptionValue(t, opts[len(opts)-1], customSentinel)

	// Descriptions should contain the concrete branch name.
	assert.Contains(t, opts[0].Key, "specs/my-feature/")
	assert.Contains(t, opts[1].Key, "docs/specs/my-feature/")
}

func TestSpecsDirOptions_NoBranch(t *testing.T) {
	opts := specsDirOptions("")

	require.Len(t, opts, 3)
	assert.Contains(t, opts[0].Key, "specs/<branch>/")
}

func TestAllOptionSetsEndWithCustom(t *testing.T) {
	pms := []PackageManager{PmUV, PmPoetry, PmNPM, PmYarn, PmPNPM, PmGo, PmCargo, PmUnknown}
	for _, pm := range pms {
		opts := runCmdOptions(&ProjectInfo{ProjectName: "x", PackageManager: pm})
		last := opts[len(opts)-1]
		assertOptionValue(t, last, customSentinel)
	}

	langs := []Language{LangPython, LangNode, LangGo, LangRust, LangUnknown}
	for _, lang := range langs {
		opts := goalOptions(&ProjectInfo{Language: lang})
		last := opts[len(opts)-1]
		assertOptionValue(t, last, customSentinel)
	}

	specsOpts := specsDirOptions("")
	assertOptionValue(t, specsOpts[len(specsOpts)-1], customSentinel)
}

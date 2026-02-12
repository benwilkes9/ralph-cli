package scaffold

import (
	"testing"

	"github.com/charmbracelet/huh"
)

func assertOptionValue(t *testing.T, opt huh.Option[string], want string) {
	t.Helper()
	if opt.Value != want {
		t.Errorf("option value = %q, want %q", opt.Value, want)
	}
}

func TestRunCmdOptions_UV(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmUV}
	opts := runCmdOptions(info)

	// 2 suggestions + custom
	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	assertOptionValue(t, opts[0], "uv run uvicorn myapp.main:app")
	assertOptionValue(t, opts[1], "uv run python -m myapp")
	assertOptionValue(t, opts[len(opts)-1], customSentinel)
}

func TestRunCmdOptions_Go(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmGo}
	opts := runCmdOptions(info)

	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	assertOptionValue(t, opts[0], "go run ./cmd/myapp")
	assertOptionValue(t, opts[1], "go run .")
}

func TestRunCmdOptions_NPM(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmNPM}
	opts := runCmdOptions(info)

	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	assertOptionValue(t, opts[0], "npm start")
	assertOptionValue(t, opts[1], "npm run dev")
}

func TestRunCmdOptions_Unknown(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmUnknown}
	opts := runCmdOptions(info)

	if len(opts) != 1 {
		t.Fatalf("expected 1 option (custom only), got %d", len(opts))
	}
	assertOptionValue(t, opts[0], customSentinel)
}

func TestGoalOptions_Python(t *testing.T) {
	info := &ProjectInfo{Language: LangPython}
	opts := goalOptions(info)

	// 1 language-specific + learning spike + custom
	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	assertOptionValue(t, opts[len(opts)-1], customSentinel)
}

func TestGoalOptions_Unknown(t *testing.T) {
	info := &ProjectInfo{Language: LangUnknown}
	opts := goalOptions(info)

	// learning spike + custom
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
	assertOptionValue(t, opts[len(opts)-1], customSentinel)
}

func TestRunCmdTitle_WithExample(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmGo}
	title := runCmdTitle(info)

	if title != "How do you start the application? (e.g. go run ./cmd/myapp)" {
		t.Errorf("unexpected title: %s", title)
	}
}

func TestRunCmdTitle_Unknown(t *testing.T) {
	info := &ProjectInfo{ProjectName: "myapp", PackageManager: PmUnknown}
	title := runCmdTitle(info)

	if title != "How do you start the application?" {
		t.Errorf("unexpected title: %s", title)
	}
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
}

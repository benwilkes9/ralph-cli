package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	assert.NotNil(t, theme)

	// Verify all styles are set (non-zero).
	assert.NotEmpty(t, theme.Body.Render("x"))
	assert.NotEmpty(t, theme.Muted.Render("x"))
	assert.NotEmpty(t, theme.Success.Render("x"))
	assert.NotEmpty(t, theme.Error.Render("x"))
	assert.NotEmpty(t, theme.Warning.Render("x"))
	assert.NotEmpty(t, theme.Cost.Render("x"))
	assert.NotEmpty(t, theme.Info.Render("x"))
}

func TestModeStyle(t *testing.T) {
	theme := DefaultTheme()

	plan := theme.ModeStyle("plan")
	build := theme.ModeStyle("build")

	assert.Equal(t, theme.PlanMode, plan)
	assert.Equal(t, theme.BuildMode, build)
}

func TestIterationStyle(t *testing.T) {
	theme := DefaultTheme()

	assert.Equal(t, theme.IterationPl, theme.IterationStyle("plan"))
	assert.Equal(t, theme.IterationBd, theme.IterationStyle("build"))
}

func TestBanner(t *testing.T) {
	theme := DefaultTheme()
	banner := theme.Banner()

	assert.Contains(t, banner, "____")
	assert.Contains(t, banner, "/ __ \\")
}

func TestFormatError(t *testing.T) {
	theme := DefaultTheme()
	out := theme.FormatError("something went wrong")

	assert.Contains(t, out, "✗")
	assert.Contains(t, out, "something went wrong")
	// Should be wrapped in a rounded border (ErrorBox)
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "╯")
}

func TestHuhTheme(t *testing.T) {
	ht := HuhTheme()
	assert.NotNil(t, ht)
	assert.NotNil(t, ht.Focused)
	assert.NotNil(t, ht.Blurred)
}

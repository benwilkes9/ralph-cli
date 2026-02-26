package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// HuhTheme returns a huh form theme using the Simpsons palette.
func HuhTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Focused field styling
	t.Focused.Title = lipgloss.NewStyle().Bold(true).Foreground(Yellow)
	t.Focused.Description = lipgloss.NewStyle().Foreground(Dim)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(Yellow).SetString("> ")
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(Green)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(White)
	t.Focused.FocusedButton = lipgloss.NewStyle().Bold(true).Foreground(Yellow)
	t.Focused.BlurredButton = lipgloss.NewStyle().Foreground(Dim)
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(Yellow)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(Yellow).SetString("> ")

	// Blurred field styling
	t.Blurred.Title = lipgloss.NewStyle().Foreground(Dim)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(Dim)
	t.Blurred.SelectSelector = lipgloss.NewStyle().Foreground(Dim).SetString("  ")
	t.Blurred.SelectedOption = lipgloss.NewStyle().Foreground(Dim)

	return t
}

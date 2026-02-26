package ui

import "github.com/charmbracelet/lipgloss"

// Simpsons color palette.
const (
	Yellow = lipgloss.Color("#FFD90F") // Simpsons Yellow — primary accent
	Blue   = lipgloss.Color("#4A8BD6") // Ralph's Blue — headers, info
	Green  = lipgloss.Color("#7BC67E") // Krusty Green — success
	Red    = lipgloss.Color("#E84D3D") // Bart's Red — errors
	Purple = lipgloss.Color("#6B4C9A") // Moe's Purple — cost values
	Sky    = lipgloss.Color("#87CEEB") // Springfield Sky — next-steps
	Dim    = lipgloss.Color("#888888") // Muted/secondary text
	White  = lipgloss.Color("#FFFFFF")
)

// Theme holds all Lip Gloss styles used across the CLI.
type Theme struct {
	// Text styles
	Body    lipgloss.Style // primary text (bold white)
	Muted   lipgloss.Style // secondary/dim text
	Success lipgloss.Style // green checkmarks, success
	Error   lipgloss.Style // red errors
	Warning lipgloss.Style // yellow warnings
	Cost    lipgloss.Style // purple cost values
	Info    lipgloss.Style // blue info labels

	// Mode-specific
	PlanMode  lipgloss.Style // blue bold for plan mode
	BuildMode lipgloss.Style // green bold for build mode

	// Structural
	BannerStyle lipgloss.Style // yellow bold for ASCII banner
	Separator   lipgloss.Style // yellow bold horizontal rule
	IterationPl lipgloss.Style // plan iteration box border
	IterationBd lipgloss.Style // build iteration box border

	// Boxes
	ErrorBox   lipgloss.Style // red rounded border for errors
	SummaryBox lipgloss.Style // blue rounded border
	StatusBox  lipgloss.Style // yellow rounded border
	NextSteps  lipgloss.Style // sky blue rounded border

	// Scaffold file status
	FileCreated lipgloss.Style // green for created files
	FileUpdated lipgloss.Style // yellow for updated files
	FileSkipped lipgloss.Style // dim for skipped files

	// Subagent
	SubagentTag lipgloss.Style // blue bold for subagent type
}

// DefaultTheme returns the Simpsons-themed default.
func DefaultTheme() *Theme {
	return &Theme{
		Body:    lipgloss.NewStyle().Bold(true).Foreground(White),
		Muted:   lipgloss.NewStyle().Foreground(Dim),
		Success: lipgloss.NewStyle().Foreground(Green),
		Error:   lipgloss.NewStyle().Bold(true).Foreground(Red),
		Warning: lipgloss.NewStyle().Bold(true).Foreground(Yellow),
		Cost:    lipgloss.NewStyle().Foreground(Purple),
		Info:    lipgloss.NewStyle().Bold(true).Foreground(Blue),

		PlanMode:  lipgloss.NewStyle().Bold(true).Foreground(Blue),
		BuildMode: lipgloss.NewStyle().Bold(true).Foreground(Green),

		BannerStyle: lipgloss.NewStyle().Bold(true).Foreground(Yellow),
		Separator:   lipgloss.NewStyle().Bold(true).Foreground(Yellow),
		IterationPl: lipgloss.NewStyle().Bold(true).Foreground(Blue),
		IterationBd: lipgloss.NewStyle().Bold(true).Foreground(Green),

		ErrorBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Red).
			Padding(0, 2),
		SummaryBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Blue).
			Padding(0, 2),
		StatusBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Yellow).
			Padding(0, 2),
		NextSteps: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Sky).
			Padding(0, 2),

		FileCreated: lipgloss.NewStyle().Foreground(Green),
		FileUpdated: lipgloss.NewStyle().Foreground(Yellow),
		FileSkipped: lipgloss.NewStyle().Foreground(Dim),

		SubagentTag: lipgloss.NewStyle().Bold(true).Foreground(Blue),
	}
}

// FormatError renders an error message inside a red-bordered box with an error icon.
func (t *Theme) FormatError(msg string) string {
	return t.ErrorBox.Render(t.Error.Render("✗ " + msg))
}

// ModeStyle returns the appropriate style for the given loop mode string.
// "plan" matches loop.ModePlan; all other values (including "build") use BuildMode.
func (t *Theme) ModeStyle(mode string) lipgloss.Style {
	if mode == "plan" {
		return t.PlanMode
	}
	return t.BuildMode
}

// IterationStyle returns the iteration box border style for the given mode.
// "plan" matches loop.ModePlan; all other values (including "build") use IterationBd.
func (t *Theme) IterationStyle(mode string) lipgloss.Style {
	if mode == "plan" {
		return t.IterationPl
	}
	return t.IterationBd
}

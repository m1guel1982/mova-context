// tui_style.go — Lip Gloss styles shared by every TUI screen (see
// tui_app.go and the tui_*.go files). One palette, one place to change
// it — no screen defines its own colors, matching Mova's "no duplicar
// lógica" rule applied to presentation instead of business logic.
package main

import "github.com/charmbracelet/lipgloss"

var (
	tuiAccent = lipgloss.Color("62")  // violeta suave — títulos, selección
	tuiMuted  = lipgloss.Color("244") // gris — texto secundario, ayuda
	tuiOK     = lipgloss.Color("42")  // verde — éxito
	tuiWarn   = lipgloss.Color("214") // ámbar — advertencia
	tuiErr    = lipgloss.Color("204") // rojo suave — error
	tuiBorder = lipgloss.Color("240")

	tuiTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(tuiAccent).
			Padding(0, 1)

	tuiHelpStyle = lipgloss.NewStyle().Foreground(tuiMuted)

	tuiOKStyle   = lipgloss.NewStyle().Foreground(tuiOK)
	tuiWarnStyle = lipgloss.NewStyle().Foreground(tuiWarn)
	tuiErrStyle  = lipgloss.NewStyle().Foreground(tuiErr).Bold(true)

	tuiStatusBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)

	tuiBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiBorder).
			Padding(0, 1)

	tuiDocStyle = lipgloss.NewStyle().Margin(1, 2)
)

// tuiHeader renders the small "Mova · <sección>" title bar every screen
// shows at the top, so navigation always feels consistent.
func tuiHeader(section string) string {
	return tuiTitleStyle.Render("Mova · "+section) + "\n"
}

// tuiFooter renders a one-line key-hint bar, consistent across screens.
func tuiFooter(hints string) string {
	return "\n" + tuiHelpStyle.Render(hints)
}

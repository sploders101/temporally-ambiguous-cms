package splash

import (
	"image/color"
	"math"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	width       int
	height      int
	text        string
	cursorColor color.Color
}

func NewSplashScreen(text string, cursorColor color.Color) Model {
	return Model{
		text:        text,
		cursorColor: cursorColor,
	}
}

func (m *Model) SetText(text string) {
	m.text = text
}

func (m *Model) SetSize(width int, height int) {
	m.width = width
	m.height = height
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() tea.View {
	instruction := "Press enter to continue."
	renderedInstruction := lipgloss.NewStyle().
		Foreground(lipgloss.BrightBlack).
		Render(instruction)
	renderedText := lipgloss.NewStyle().
		Foreground(m.cursorColor).
		Render(m.text)
	view := tea.NewView(
		lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			renderedText+"\n\n"+renderedInstruction,
		),
	)
	view.Cursor = tea.NewCursor(
		(m.width/2)+(len(instruction)/2),
		int(math.Round((float64(m.height) / 2))),
	)
	view.Cursor.Color = m.cursorColor
	view.AltScreen = true
	return view
}

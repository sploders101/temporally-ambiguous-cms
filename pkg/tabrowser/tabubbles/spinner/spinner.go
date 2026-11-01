package spinner

import (
	"log/slog"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/bigstraw"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
)

var defaultSpinner [][]string = [][]string{
	{"▰▱▱ ", " ▱▱▱"},
	{"▱▰▱ ", " ▱▱▱"},
	{"▱▱▰ ", " ▱▱▱"},
	{"▱▱▱ ", " ▱▱▱"},
	{"▱▱▱ ", " ▰▱▱"},
	{"▱▱▱ ", " ▱▰▱"},
	{"▱▱▱ ", " ▱▱▰"},
}
var defaultInterval = time.Second / 10

type Model struct {
	frames       [][]string
	interval     time.Duration
	message      string
	tickerCancel chan struct{}
	frameIndex   int
	visible      bool
	generation   int

	containerStyle lipgloss.Style
	spinnerStyle   lipgloss.Style
	messageStyle   lipgloss.Style
}

func New() *Model {
	return &Model{
		frames:   defaultSpinner,
		interval: defaultInterval,
		message:  "Loading...",
	}
}

type ticker struct {
	generation int
}

func (m *Model) SetTheme(theme tabubbles.Theme) {
	m.containerStyle = lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(theme.PrimaryColor, theme.AccentColor)
	m.spinnerStyle = lipgloss.NewStyle().Foreground(theme.SecondaryColor)
	m.messageStyle = lipgloss.NewStyle().Foreground(theme.TextColor)
}

func (spinner *Model) SetVisibility(visible bool, cmds bigstraw.CmdCollector) {
	slog.Debug("Set visibility", "visibility", visible)
	switch {
	case spinner.visible && !visible:
		close(spinner.tickerCancel)
		spinner.generation += 1
	case !spinner.visible && visible:
		spinner.tickerCancel = make(chan struct{})
		cmds.AddNotifySelf(spinner.getTicker())
	}
	spinner.visible = visible
}

func (spinner *Model) GetVisibility() bool {
	return spinner.visible
}

func (spinner *Model) Update(msg tea.Msg, cmds bigstraw.CmdCollector) {
	switch msg := msg.(type) {
	case bigstraw.InitMsg:
		if spinner.visible {
			spinner.tickerCancel = make(chan struct{})
			cmds.AddNotifySelf(spinner.getTicker())
		}
	case ticker:
		if spinner.visible && msg.generation == spinner.generation {
			spinner.frameIndex += 1 // renderers will do their own modulo
			cmds.AddNotifySelf(spinner.getTicker())
		}
	}
}

func (spinner *Model) getTicker() tea.Cmd {
	interval := spinner.interval
	generation := spinner.generation
	cancelChan := spinner.tickerCancel
	return func() tea.Msg {
		select {
		case <-time.After(interval):
			return ticker{
				generation: generation,
			}
		case <-cancelChan:
			return nil
		}
	}
}

func (spinner *Model) View() string {
	if !spinner.visible {
		return ""
	}

	frame := spinner.frames[spinner.frameIndex%len(spinner.frames)]
	message := spinner.spinnerStyle.Render(frame[0]) +
		spinner.messageStyle.Render(spinner.message) +
		spinner.spinnerStyle.Render(frame[1])

	return spinner.containerStyle.BorderForegroundBlendOffset(spinner.frameIndex).Render(message)
}

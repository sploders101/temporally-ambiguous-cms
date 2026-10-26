package spinner

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/bubbles/themes"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller/modules/utilsctrl"
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

func New() Model {
	return Model{
		frames:   defaultSpinner,
		interval: defaultInterval,
		message:  "Loading...",
	}
}

func (spinner Model) Init() tea.Cmd {
	if spinner.visible {
		return controller.CallController(utilsctrl.TickerRequest{
			Cancel:   spinner.tickerCancel,
			Duration: spinner.interval,
			Msg:      spinner.generation,
		})
	}
	return nil
}

func (m *Model) SetTheme(theme themes.Theme) {
	m.containerStyle = lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(theme.PrimaryColor, theme.AccentColor)
	m.spinnerStyle = lipgloss.NewStyle().Foreground(theme.SecondaryColor)
	m.messageStyle = lipgloss.NewStyle().Foreground(theme.TextColor)
}

func (spinner *Model) SetVisibility(visible bool) tea.Cmd {
	switch {
	case spinner.visible && !visible:
		close(spinner.tickerCancel)
		spinner.generation += 1
		spinner.visible = visible
		return nil
	case !spinner.visible && visible:
		spinner.tickerCancel = make(chan struct{})
		spinner.visible = visible
		return controller.CallController(utilsctrl.TickerRequest{
			Cancel:   spinner.tickerCancel,
			Duration: spinner.interval,
			Msg:      spinner.generation,
		})
	default:
		return nil
	}
}

func (spinner Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case controller.ControllerResponse:
		switch msg := msg.Msg.(type) {
		case utilsctrl.TickerResponse:
			if generation, ok := msg.Msg.(int); !ok || generation != spinner.generation {
				return spinner, nil
			}
			spinner.frameIndex += 1 // renderers will do their own modulo
			return spinner, controller.CallController(utilsctrl.TickerRequest{
				Cancel:   spinner.tickerCancel,
				Duration: spinner.interval,
				Msg:      spinner.generation,
			})
		}
	}
	return spinner, nil
}

func (spinner Model) View() string {
	if !spinner.visible {
		return ""
	}

	frame := spinner.frames[spinner.frameIndex%len(spinner.frames)]
	message := spinner.spinnerStyle.Render(frame[0]) +
		spinner.messageStyle.Render(spinner.message) +
		spinner.spinnerStyle.Render(frame[1])

	return spinner.containerStyle.BorderForegroundBlendOffset(spinner.frameIndex).Render(message)
}

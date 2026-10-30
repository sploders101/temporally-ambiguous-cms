package spinner

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller/modules/utilsctrl"
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

func (spinner *Model) Init(cmds tabubbles.CmdCollector) {
	if spinner.visible {
		cmds.Push(controller.CallController(utilsctrl.TickerRequest{
			Cancel:   spinner.tickerCancel,
			Duration: spinner.interval,
			Msg:      spinner.generation,
		}))
	}
}

func (m *Model) SetTheme(theme tabubbles.Theme) {
	m.containerStyle = lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(theme.PrimaryColor, theme.AccentColor)
	m.spinnerStyle = lipgloss.NewStyle().Foreground(theme.SecondaryColor)
	m.messageStyle = lipgloss.NewStyle().Foreground(theme.TextColor)
}

func (spinner *Model) SetVisibility(visible bool, cmds tabubbles.CmdCollector) {
	switch {
	case spinner.visible && !visible:
		close(spinner.tickerCancel)
		spinner.generation += 1
		spinner.visible = visible
	case !spinner.visible && visible:
		spinner.tickerCancel = make(chan struct{})
		spinner.visible = visible
		cmds.Push(controller.CallController(utilsctrl.TickerRequest{
			Cancel:   spinner.tickerCancel,
			Duration: spinner.interval,
			Msg:      spinner.generation,
		}))
	}
}

func (spinner *Model) GetVisibility() bool {
	return spinner.visible
}

func (spinner *Model) Update(msg tea.Msg, cmds tabubbles.CmdCollector) {
	switch msg := msg.(type) {
	case controller.ControllerResponse:
		switch msg := msg.Msg.(type) {
		case utilsctrl.TickerResponse:
			if generation, ok := msg.Msg.(int); !ok || generation != spinner.generation {
				return
			}
			spinner.frameIndex += 1 // renderers will do their own modulo
			cmds.Push(controller.CallController(utilsctrl.TickerRequest{
				Cancel:   spinner.tickerCancel,
				Duration: spinner.interval,
				Msg:      spinner.generation,
			}))
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

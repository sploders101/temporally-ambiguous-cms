package splash

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller/modules/utilsctrl"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
)

const (
	TickCursor int = iota
	TickText
)

const (
	tickDuration  time.Duration = time.Second / 40
	blinkDuration time.Duration = time.Second
	splashText    string        = "Temporally Ambiguous Browser"
	splashTrailer string        = "Press ENTER to continue."
	splashPause   time.Duration = time.Second
)

var splashTicksTotal = len(splashText) + 2 /* \n\n */ + len(splashTrailer)

type Model struct {
	width            int
	height           int
	textOffset       int
	cursorVisibility bool
	splashText       lipgloss.Style
	splashTrailer    lipgloss.Style
	cursor           lipgloss.Style
}

func NewSplashScreen(text string) *Model {
	return &Model{
		cursorVisibility: true,
	}
}

func (model *Model) Init(cmds tabubbles.CmdCollector) {
	cmds.Push(controller.CallController(utilsctrl.TickerRequest{
		Duration: blinkDuration,
		Msg:      TickCursor,
	}))
	cmds.Push(controller.CallController(utilsctrl.TickerRequest{
		Duration: tickDuration,
		Msg:      TickText,
	}))
}

func (model *Model) SetSize(width int, height int) {
	model.width = width
	model.height = height
}

func (model *Model) SetTheme(theme tabubbles.Theme) {
	model.cursor = lipgloss.NewStyle().Background(theme.PrimaryColor)
	model.splashText = lipgloss.NewStyle().Foreground(theme.PrimaryColor).Bold(true)
	model.splashTrailer = lipgloss.NewStyle().Foreground(theme.MutedColor)
}

func (model *Model) Update(msg tea.Msg, cmds tabubbles.CmdCollector) {
	switch msg := msg.(type) {
	case controller.ControllerResponse:
		_, ok, msg := msg.PopIndex()
		if ok {
			return
		}
		switch msg := msg.Msg.(type) {
		case utilsctrl.TickerResponse:
			if msg, ok := msg.Msg.(int); ok {
				switch msg {
				case TickCursor:
					model.cursorVisibility = !model.cursorVisibility
					cmds.Push(controller.CallController(utilsctrl.TickerRequest{
						Duration: blinkDuration,
						Msg:      TickCursor,
					}))
				case TickText:
					model.textOffset += 1
					if model.textOffset == len(splashText) {
						cmds.Push(controller.CallController(utilsctrl.TickerRequest{
							Duration: splashPause,
							Msg:      TickText,
						}))
					} else if model.textOffset < splashTicksTotal {
						cmds.Push(controller.CallController(utilsctrl.TickerRequest{
							Duration: tickDuration,
							Msg:      TickText,
						}))
					}
				}
			}
		}
	}
}

func (model *Model) View() string {
	splashTextEnd := min(len(splashText), model.textOffset)
	var totalStr string
	if model.cursorVisibility && model.textOffset <= len(splashText) {
		totalStr = " "
	}
	totalStr += model.splashText.Render(splashText[:splashTextEnd])

	if model.textOffset > len(splashText) {
		totalStr += "\n\n"[:min(2, model.textOffset-len(splashText))]
		trailerIdx := max(0, model.textOffset-len(splashText)-2)
		if model.cursorVisibility && trailerIdx > 0 {
			totalStr += " "
		}
		totalStr += model.splashTrailer.Render(splashTrailer[:trailerIdx])
	}

	if model.cursorVisibility {
		totalStr += model.cursor.Render(" ")
	} else {
		totalStr += ""
	}

	return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, totalStr)
}

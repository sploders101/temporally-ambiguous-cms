package splash

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/bigstraw"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
)

const (
	TickCursor int = iota
	TickText
)

const (
	tickDuration  time.Duration = time.Second / 40
	blinkDuration time.Duration = time.Second
	splashPause   time.Duration = time.Second
)

type Model struct {
	visible          bool
	generation       int
	width            int
	height           int
	textOffset       int
	cursorVisibility bool
	topText          string
	bottomText       string
	splashTicksTotal int
	splashText       lipgloss.Style
	splashTrailer    lipgloss.Style
	cursor           lipgloss.Style
}

func NewModel(visible bool, topText string, bottomText string) *Model {
	return &Model{
		visible:          true,
		cursorVisibility: true,
		topText:          topText,
		bottomText:       bottomText,
		splashTicksTotal: len(topText) + 2 /* \n\n */ + len(bottomText),
	}
}

type tickCursor struct {
	generation int
}
type tickText struct {
	generation int
}

func cursorTicker(generation int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(blinkDuration)
		return tickCursor{generation}
	}
}
func textTicker(generation int, duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duration)
		return tickText{generation}
	}
}

func (model *Model) SetVisibility(visible bool, cmds bigstraw.CmdCollector) {
	switch {
	case model.visible && !visible:
		model.generation += 1
		model.visible = false
	case !model.visible && visible:
		model.textOffset = 0
		model.cursorVisibility = false
		cmds.AddNotifySelf(cursorTicker(model.generation))
		cmds.AddNotifySelf(textTicker(model.generation, tickDuration))
	}
	model.visible = visible
}

func (model *Model) SetSize(width int, height int, cmds bigstraw.CmdCollector) {
	model.width = width
	model.height = height
}

func (model *Model) SetTheme(theme tabubbles.Theme) {
	model.cursor = lipgloss.NewStyle().Background(theme.PrimaryColor)
	model.splashText = lipgloss.NewStyle().Foreground(theme.PrimaryColor).Bold(true)
	model.splashTrailer = lipgloss.NewStyle().Foreground(theme.MutedColor)
}

func (model *Model) Update(msg tea.Msg, cmds bigstraw.CmdCollector) {
	msg = bigstraw.HandleMsg(msg, cmds, nil)
	switch msg := msg.(type) {
	case bigstraw.InitMsg:
		model.visible = !model.visible
		model.SetVisibility(!model.visible, cmds)
	case tickCursor:
		if msg.generation == model.generation {
			model.cursorVisibility = !model.cursorVisibility
			cmds.AddNotifySelf(cursorTicker(model.generation))
		}
	case tickText:
		if msg.generation == model.generation {
			model.textOffset += 1
			if model.textOffset == len(model.topText) {
				cmds.AddNotifySelf(textTicker(model.generation, splashPause))
			} else if model.textOffset < model.splashTicksTotal {
				cmds.AddNotifySelf(textTicker(model.generation, tickDuration))
			}
		}
	}
}

func (model *Model) View() string {
	if !model.visible {
		return ""
	}
	splashTextEnd := min(len(model.topText), model.textOffset)
	var totalStr string
	if model.cursorVisibility && model.textOffset <= len(model.topText) {
		totalStr = " "
	}
	totalStr += model.splashText.Render(model.topText[:splashTextEnd])

	if model.textOffset > len(model.topText) {
		totalStr += "\n\n"[:min(2, model.textOffset-len(model.topText))]
		trailerIdx := max(0, model.textOffset-len(model.topText)-2)
		if model.cursorVisibility && trailerIdx > 0 {
			totalStr += " "
		}
		totalStr += model.splashTrailer.Render(model.bottomText[:trailerIdx])
	}

	if model.cursorVisibility {
		totalStr += model.cursor.Render(" ")
	} else {
		totalStr += ""
	}

	return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, totalStr)
}

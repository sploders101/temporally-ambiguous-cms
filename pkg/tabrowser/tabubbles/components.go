package tabubbles

import (
	tea "charm.land/bubbletea/v2"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

type Component interface {
	Init(CmdCollector)
	Update(tea.Msg, CmdCollector)
	View() string
}

type Sizable interface {
	SetSize(width, height int)
}

type Themable interface {
	SetTheme(theme Theme)
}

type CmdCollector struct {
	trackingIndex []int
	commands      *[]tea.Cmd
}

func NewCmdCollector() CmdCollector {
	var commands []tea.Cmd
	return CmdCollector{
		commands: &commands,
	}
}

func (collector CmdCollector) WithIndex(idx int) CmdCollector {
	trackingIndex := make([]int, len(collector.trackingIndex)+1)
	copy(trackingIndex, collector.trackingIndex)
	trackingIndex[len(trackingIndex)-1] = idx
	return CmdCollector{
		trackingIndex: trackingIndex,
		commands:      collector.commands,
	}
}

func (collector CmdCollector) Push(cmds ...tea.Cmd) {
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		*collector.commands = append(*collector.commands, func() tea.Msg {
			msg := cmd()
			if msg, ok := msg.(controller.ControllerCommand); ok {
				msg.IndexChain = collector.trackingIndex
				return msg
			}
			return msg
		})
	}
}

func (collector CmdCollector) ToCmd() tea.Cmd {
	return tea.Batch(*collector.commands...)
}

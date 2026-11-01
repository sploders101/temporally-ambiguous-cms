package article

import (
	tea "charm.land/bubbletea/v2"
	"github.com/sploders101/personal-website/pkg/bigstraw"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
)

type Model struct {
}

func New() *Model {
	return &Model{}
}

func (model *Model) SetTheme(theme tabubbles.Theme) {}

func (model *Model) Update(msg tea.Msg, cmds bigstraw.CmdCollector) {}

func (model *Model) View() string {
	return "TESTING"
}

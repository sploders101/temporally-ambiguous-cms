package article

import (
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
)

type Model struct {
}

func New() *Model {
	return &Model{}
}

func (model *Model) Init(cmds tabubbles.CmdCollector) {}

func (model *Model) SetTheme(theme tabubbles.Theme) {}

func (model *Model) Update(cmds tabubbles.CmdCollector) {}

func (model *Model) View() string {
	return "TESTING"
}

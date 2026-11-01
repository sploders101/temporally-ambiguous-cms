package bigstraw

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
)

// This is a thin wrapper to convert a bigstraw component into a
// bubbletea component.
//
// Because it only accepts one child and doesn't contain any
// behavior of its own, it holds no index in message paths, and
// is functionally transparent.
//
// If a standard interface such as `Sizable` is attached to this
// component, its corresponding messages (eg. `tea.WindowSizeMsg`)
// will be intercepted and rerouted to the corresponding method.
// Otherwise, the message will propagate to the standard `Update`
// method.
type AppRoot struct {
	root RootComponent
}

func NewAppRoot(root RootComponent) AppRoot {
	return AppRoot{root}
}

func (app AppRoot) Init() tea.Cmd {
	cmds := NewCmdCollector()
	app.root.Update(InitMsg{}, cmds)
	return cmds.ToCmd()
}

func (app AppRoot) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	slog.Debug("Got msg", "msg", msg)
	cmds := NewCmdCollector()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if root, ok := app.root.(Sizable); ok {
			root.SetSize(msg.Width, msg.Height, cmds)
			return app, cmds.ToCmd()
		}
	}
	app.root.Update(msg, cmds)
	return app, cmds.ToCmd()
}

func (app AppRoot) View() tea.View {
	return app.root.View()
}

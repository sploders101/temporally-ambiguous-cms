package iotracker

import tea "charm.land/bubbletea/v2"

type TrackedIO struct {
	Index int
	Cmd   tea.Msg
}

func TrackIO(index int, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		cmd := cmd()
		return TrackedIO{
			Index: index,
			Cmd:   cmd,
		}
	}
}

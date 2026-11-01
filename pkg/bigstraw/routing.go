package bigstraw

import (
	"maps"
	"slices"

	tea "charm.land/bubbletea/v2"
)

// A message which is intended for a specific component
type RoutedMsg struct {
	trackingIndex []int
	msg           tea.Msg
}

type CmdCollector struct {
	path        []int
	roleIndexes map[string]int
	commands    *[]tea.Cmd
}

func NewCmdCollector() CmdCollector {
	var commands []tea.Cmd
	return CmdCollector{
		commands: &commands,
	}
}

func (collector CmdCollector) DesignateAs(role string) CmdCollector {
	roleIndexes := maps.Clone(collector.roleIndexes)
	if roleIndexes == nil {
		roleIndexes = make(map[string]int)
	}
	roleIndexes[role] = len(collector.path)
	return CmdCollector{
		path:        collector.path,
		roleIndexes: roleIndexes,
		commands:    collector.commands,
	}
}

func (collector CmdCollector) WithIndex(idx int) CmdCollector {
	trackingIndex := make([]int, len(collector.path)+1)
	copy(trackingIndex, collector.path)
	trackingIndex[len(collector.path)] = idx
	return CmdCollector{
		path:        trackingIndex,
		roleIndexes: collector.roleIndexes,
		commands:    collector.commands,
	}
}

func (collector CmdCollector) GetPath() []int {
	return slices.Clone(collector.path)
}

func (collector CmdCollector) WrapNotifySelf(cmd tea.Cmd) tea.Cmd {
	path := collector.path
	if len(path) == 0 {
		return cmd
	}
	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}
		return RoutedMsg{
			trackingIndex: path,
			msg:           msg,
		}
	}
}

func (collector CmdCollector) WrapNotifyRole(role string, cmd tea.Cmd) tea.Cmd {
	if index, ok := collector.roleIndexes[role]; ok {
		path := collector.path[:index]
		if len(path) == 0 {
			return cmd
		}
		return func() tea.Msg {
			msg := cmd()
			if msg == nil {
				return nil
			}
			return RoutedMsg{
				trackingIndex: path,
				msg:           msg,
			}
		}
	}
	return nil
}

// Adds commands to the collector without tracking
func (collector CmdCollector) Add(cmds ...tea.Cmd) {
	*collector.commands = append(*collector.commands, cmds...)
}

func (collector CmdCollector) AddNotifySelf(cmds ...tea.Cmd) {
	wrappedCmds := make([]tea.Cmd, len(cmds))
	for i, cmd := range cmds {
		wrappedCmds[i] = collector.WrapNotifySelf(cmd)
	}
	*collector.commands = append(*collector.commands, wrappedCmds...)
}

func (collector CmdCollector) AddNotifyRole(role string, cmds ...tea.Cmd) {
	wrappedCmds := make([]tea.Cmd, len(cmds))
	for i, cmd := range cmds {
		wrappedCmds[i] = collector.WrapNotifyRole(role, cmd)
	}
	*collector.commands = append(*collector.commands, wrappedCmds...)
}

func (collector CmdCollector) ToCmd() tea.Cmd {
	return tea.Batch(*collector.commands...)
}

// Handles a message, either by routing it to a child component, or by
// unwrapping it and returning.
//
// If the message was not handled, it is returned. Otherwise, the returned message will
// be `nil`. Make sure to use the returned message instead of the original in case the
// message was unwrapped.
func HandleMsg(
	msg tea.Msg,
	cmds CmdCollector,
	components map[int]Component,
) (unwrappedMsg tea.Msg) {
	if _, ok := msg.(InitMsg); ok {
		for idx, component := range components {
			component.Update(msg, cmds.WithIndex(idx))
		}
		return msg
	}
	routedMsg, ok := msg.(RoutedMsg)
	if !ok {
		return msg
	}

	if len(routedMsg.trackingIndex) == 0 {
		return routedMsg.msg
	}

	idx := routedMsg.trackingIndex[0]
	routedMsg.trackingIndex = routedMsg.trackingIndex[1:]
	var newMsg tea.Msg
	if len(routedMsg.trackingIndex) == 0 {
		newMsg = routedMsg.msg
	} else {
		newMsg = routedMsg
	}
	if component, ok := components[idx]; ok {
		component.Update(newMsg, cmds.WithIndex(idx))
		return nil
	}
	return msg
}

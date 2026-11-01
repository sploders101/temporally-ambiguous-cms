package bigstraw

import tea "charm.land/bubbletea/v2"

// The initialization event.
//
// This takes the place of the `Init` function in `tea.Model`, allowing
// consolidation of message routing functionality into the `Update` function
// using `HandleMsg`.
//
// This is the first event that any bigstraw component should receive. This
// event should be propagated to all sub-components, whether manually, or
// automatically via `HandleMsg`. Initialization shall also be defined as
// occurring from leaf to stem, ie. the child comes first, then the parent.
type InitMsg struct {}

// This represents a `bigstraw` root component.
//
// The difference between `Component` and `RootComponent` is the return value
// of the `View` method. The root component has full control over the
// `tea.View` object, while subcomponents do not. The view is considered global
// state, and thus should not be implicitly able to be controlled by
// subcomponents.
type RootComponent interface {
	Update(tea.Msg, CmdCollector)
	View() tea.View
}

// This represents a `bigstraw` component.
//
// The `Component` interface is a consolidated form of `bubbletea`'s `Model`
// interface. It features a `CmdCollector` object which keeps track of the
// component's path within the tree and provides functions for batching
// commands and routing their results.
//
// Unlike `tea.Model`, initialization occurs in the `Update` function using the
// `InitMsg` message type, and is propagated to all children through a special
// case in the `HandleMsg` function.
type Component interface {
	Update(tea.Msg, CmdCollector)
	View() string
}

// Generic interface for components which can be resized by the parent
type Sizable interface {
	SetSize(width, height int, cmds CmdCollector)
}

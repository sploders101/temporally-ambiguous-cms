# Bigstraw

Warning: This has a fair bit of rambling. Just getting my thoughts out for now;
it's not yet ready to be published.

Bigstraw is a framework for consuming `bubbletea` from charm.sh. Bubbletea
doesn't have a component framework. They define the interface between your
application and the renderer that draws it to the terminal. That's about it.
They also use the Elm architecture. While I appreciate the concept behind this,
it makes it hard to create and use components. In essence, every component has
to be aware of every other component, because messages have no routing
mechanism, and all I/O goes through `tea.Cmd`, which dumps every result into
the root component, regardless of who actually needs it.

Charm has done amazing work at building a nice framework, and building
something that can efficiently render immediate-mode frames to the terminal in
a predictable fashion without giving up low-level control is no easy feat. I
really love how easy they've made this... but it's still missing some
scaffolding.


## Components

What Bubbletea is missing is a way to define a component tree. They've done a
little bit with their `bubbles` library, which defines a set of nice, pre-made
components that are ready to use. However, this adherence to an
almost-`tea.Model`-but-not-quite structure makes things confusing and yields
surprising behavior when you go beyond the minimal example cases due to the use
of undocumented message types that are expected to be returned, but don't
actually define any composable way of returning them other than "pass along
anything you don't recognize", which can yield clashes between multiple
instances of the same component, and performance degradation since every
message is an application-wide broadcast.

### `CmdCollector`

First, I think the interface between components can be simplified by adding an
object capable of tracking the current position in the component tree, and
allowing components to append commands to a global list declared in the root
component's handler scope.

This interface could wrap components' commands in order to specifically address
their results back to either the component itself, or a parent as defined by a
"role". Roles are attributes a component can self-identify with as the context
propagates to its children. Child components can then address messages to their
closest parent with the specified role. This allows things like navigation
containers with behavior similar to an iframe, or it can serve as a simple
shortcut for plumbing messages to known related components.

### Collapsing return values

Second, I think we can do better for the function signatures. I think we should
standardize on pointer receivers for the methods and eliminate the return
values. Pointers allow modification of the component directly, and commands can
be tracked in the `CmdCollector` object we just finished discussing. This makes
things like early returns much easier to do without exploding the amount of
boilerplate involved, mostly regarding the reassignment dance:

```go
func (model Model) Update(msg tea.Msg) tea.Cmd {
  myIO := func() tea.Msg {
    // My extra functionality needed for the parent component
  }
  var cmd tea.Cmd
  switch msg.(type) {
  case Component1Msg:
    model.component1, cmd := model.component1.Update(msg)
    return model, tea.Batch(cmd, myIO)
  case Component2Msg:
    model.component2, cmd := model.component2.Update(msg)
    return model, tea.Batch(cmd, myIO)
  case Component3Msg:
    model.component3, cmd := model.component3.Update(msg)
    return model, tea.Batch(cmd, myIO)
  case MyMsg1:
    model.thing1 = msg.Prop
  case MyMsg2:
    model.thing2 = msg.Prop
  }
  return model, myIO
}
```

Instead, we leverage the new `CmdCollector` to build a function called
`HandleMsg` which takes 3 arguments: `tea.Msg`, `CmdCollector`, and
`map[int]Component`, and returns a single `tea.Msg`, which will be `nil` if the
message has already been delegated to a child component. Otherwise, it returns
the original message. This turns that dance into:

```go
func (model Model) Update(msg tea.Msg, cmds bigstraw.CmdCollector) {
  msg = bigstraw.HandleMsg(msg, cmds, map[int]bigstraw.Component{
    Component1: model.component1,
    Component2: model.component2,
    Component3: model.component3,
  })
  switch msg.(type) {
  case nil:
    return
  case MyMsg1:
    model.thing1 = msg.Prop
  case MyMsg2:
    model.thing2 = msg.Prop
  }
  cmds.AddNotifySelf(func() tea.Msg {
    // My extra functionality needed for the parent component
  })
}
```

Note how the component is now only responsible for defining its children, their
index values, and handling its own messages. It's no longer concerned with the
messages of its children, and may easily return early if desired.

### Rolling `Init` into `Update`

This one doesn't make a lot of sense on its own, but consider the problems
we've already solved. We have proper routing for the `Update` method using
`HandleMsg(msg, cmds, children)`, and we have no return value to contend with.
This means that the signature of `Update` is very similar to that of `Init`,
with the only exception being `msg tea.Msg`. If we redefine initialization as
a message, then we can add a special case to `HandleMsg` that properly
initializes all child components first, then initializes the parent with no
extra code. This means as long as component authors properly structure their
`Update` functions, propagation of component initialization is no longer a
concern. The `Update` method is called with `InitMsg{}`, and the component may
handle it if it wishes... or not... and if it doesn't have anything to
initialize, it still propagates to its children for free.

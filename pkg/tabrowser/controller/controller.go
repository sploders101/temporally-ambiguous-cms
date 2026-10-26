package controller

import (
	"context"
	"sync"

	tea "charm.land/bubbletea/v2"
)

type Controller interface {
	Init(ctx context.Context, respChan chan<- ControllerResponse)
	Handle(req ControllerCommand)
}

// A type of `tea.Msg` intended to forward requests to the controller
type ControllerCommand struct {
	IndexChain []int
	Msg        tea.Msg
}

func (command ControllerCommand) AppendIndex(index int) ControllerCommand {
	command.IndexChain = append(command.IndexChain, index)
	return command
}

func TrackRequest(cmd tea.Cmd, index int) tea.Cmd {
	return func() tea.Msg {
		msg := cmd()
		if msg, ok := msg.(ControllerCommand); ok {
			return msg.AppendIndex(index)
		}
		return msg
	}
}

type ControllerResponse struct {
	IndexChain []int
	Msg        tea.Msg
}

// Retrieves indexes from the response in FILO order.
// If no indexes were left, `hadIndex` will be `false`.
// `response` contains the response with the current index removed
func (command ControllerResponse) PopIndex() (index int, hadIndex bool, response ControllerResponse) {
	lastIndex := len(command.IndexChain) - 1
	if lastIndex == -1 {
		return 0, false, command
	}
	index = command.IndexChain[lastIndex]
	command.IndexChain = command.IndexChain[:lastIndex]
	return index, true, command
}

// Used to bubble an event up to the controller.
//
// The root component should handle `ControllerCommand` messages by dispatching them to the
// controller task. This function allows subcomponents to easily bubble up a request through the existing
// event propagation system.
func CallController(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return ControllerCommand{Msg: msg}
	}
}

// This function keeps track of the application state, pushing down updates
func StartControllerLoop(
	ctx context.Context,
	controllers []Controller,
) (chan<- ControllerCommand, <-chan ControllerResponse) {
	requestChan := make(chan ControllerCommand, 1)
	msgChan := make(chan ControllerResponse, 1)

	for _, controller := range controllers {
		controller.Init(ctx, msgChan)
	}

	go func() {
		var wg sync.WaitGroup
		// Quit the application if the handler exits
		defer func() {
			wg.Wait()
			close(requestChan)
			close(msgChan)
		}()

		for {
			select {
			case req, ok := <-requestChan:
				if !ok {
					return
				}
				// Handle request, send response through msgChan to update UI
				wg.Go(func() {
					for _, controller := range controllers {
						controller.Handle(req)
					}
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	return requestChan, msgChan
}

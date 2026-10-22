package controller

import (
	"context"
	"errors"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
)

var (
	ErrUnknownRequest = errors.New("unknown request")
)

type HandlerFunc func(ctx context.Context, req ControllerCommand, msgChan chan<- ControllerResponse) error

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

// Initializes the application. The root component should emit this event as the last step of initialization.
//
// The controller will make a series of requests and emit the initial states of the application. Until these
// events come, all states should be considered "loading".
type GetArticles struct {
	BeforeTime time.Time
	Offset     int
	Limit      int
}
type UpdateArticles struct {
	GetArticles
	Articles []*contentv1.ArticleMetadata
}
type ShowError struct{ err error }

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
func NewController(ctx context.Context, handler HandlerFunc) (chan<- ControllerCommand, <-chan ControllerResponse) {
	requestChan := make(chan ControllerCommand, 1)
	msgChan := make(chan ControllerResponse, 1)

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
					if err := handler(ctx, req, msgChan); err != nil {
						msgChan <- ControllerResponse{Msg: ShowError{err}}
					}
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	return requestChan, msgChan
}

func NewRPCMsgHandler(client contentv1connect.ContentServiceClient) HandlerFunc {
	return func(ctx context.Context, req ControllerCommand, respChan chan<- ControllerResponse) error {
		switch msg := req.Msg.(type) {
		case GetArticles:
			articles, err := client.ListArticles(ctx, contentv1.ListArticlesRequest_builder{
				Offset: int32(msg.Offset),
				Limit:  int32(msg.Limit),
			}.Build())
			if err != nil {
				return err
			}

			respChan <- ControllerResponse{
				IndexChain: req.IndexChain,
				Msg: UpdateArticles{
					GetArticles: msg,
					Articles:    articles.GetArticles(),
				},
			}
		default:
			return ErrUnknownRequest
		}
		return nil
	}
}

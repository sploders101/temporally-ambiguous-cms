package tabrowser

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/bubbles/posts"
	"github.com/sploders101/personal-website/pkg/bubbles/splash"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

const (
	ScreenSplash int = iota
	ScreenPosts
)

const (
	ComponentPosts int = iota
)

type Model struct {
	controllerIn  chan<- controller.ControllerCommand
	controllerOut <-chan controller.ControllerResponse
	currentScreen int
	splash        splash.Model
	posts         posts.Model
}

func NewModel(ctx context.Context, handler controller.HandlerFunc) Model {
	controllerIn, controllerOut := controller.NewController(ctx, handler)
	return Model{
		controllerIn:  controllerIn,
		controllerOut: controllerOut,
		splash:        splash.NewSplashScreen("Temporally Ambiguous Browser", lipgloss.Color("#FF4000")),
		posts:         posts.New(),
	}
}

func (model Model) Init() tea.Cmd {
	return tea.Batch(
		model.listenControllerResponse,
		controller.TrackRequest(model.posts.Init(), ComponentPosts),
	)
}

func (model *Model) SetSize(width int, height int) {
	model.splash.SetSize(width, height)
	model.posts.SetSize(width, height)
}

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return model, tea.Quit
		}
		switch model.currentScreen {
		case ScreenSplash:
			if msg.String() == "enter" {
				model.currentScreen = ScreenPosts
			}
		case ScreenPosts:
			var cmd tea.Cmd
			model.posts, cmd = model.posts.Update(msg)
			return model, cmd
		}
	case tea.WindowSizeMsg:
		model.SetSize(msg.Width, msg.Height)
	case controller.ControllerCommand:
		model.controllerIn <- msg
	case controller.ControllerResponse:
		if index, ok, msg := msg.PopIndex(); ok {
			// The event was for someone else
			var cmd tea.Cmd
			switch index {
			case ComponentPosts:
				model.posts, cmd = model.posts.Update(msg)
			}
			return model, tea.Batch(model.listenControllerResponse, cmd)
		} else {
			// The event was for us
			return model, model.listenControllerResponse
		}
	}
	return model, nil
}

func (model Model) listenControllerResponse() tea.Msg {
	update, ok := <-model.controllerOut
	if !ok {
		return nil
	}
	return update
}

func (model Model) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true

	switch model.currentScreen {
	case ScreenSplash:
		return model.splash.View()
	case ScreenPosts:
		view.SetContent(model.posts.View())
	}

	return view
}

package tabrowser

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/bubbles/postlist"
	"github.com/sploders101/personal-website/pkg/bubbles/splash"
	"github.com/sploders101/personal-website/pkg/bubbles/themes"
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
	width         int
	height        int
	currentScreen int
	splash        splash.Model
	posts         postlist.Model
}

func NewModel(ctx context.Context, controllers []controller.Controller) Model {
	controllerIn, controllerOut := controller.StartControllerLoop(ctx, controllers)
	model := Model{
		controllerIn:  controllerIn,
		controllerOut: controllerOut,
		splash:        splash.NewSplashScreen("Temporally Ambiguous Browser"),
		posts:         postlist.New(),
	}
	model.SetTheme(themes.DefaultTheme)
	return model
}

func (model Model) Init() tea.Cmd {
	return tea.Batch(
		model.listenControllerResponse,
		controller.TrackRequest(model.posts.Init(), ComponentPosts),
	)
}

func (model *Model) SetSize(width int, height int) {
	model.width, model.height = width, height
	model.splash.SetSize(width, height)
	model.posts.SetSize(width, height)
}

func (model *Model) SetTheme(theme themes.Theme) {
	model.splash.SetTheme(theme)
	model.posts.SetTheme(theme)
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
				cmd := model.posts.SetVisibility(true)
				return model, controller.TrackRequest(cmd, ComponentPosts)
			}
		case ScreenPosts:
			var cmd tea.Cmd
			model.posts, cmd = model.posts.Update(msg)
			return model, controller.TrackRequest(cmd, ComponentPosts)
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
				cmd = controller.TrackRequest(cmd, ComponentPosts)
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
	if model.currentScreen == ScreenSplash {
		return model.splash.View()
	}

	compositor := lipgloss.NewCompositor()
	switch model.currentScreen {
	case ScreenPosts:
		compositor.AddLayers(lipgloss.NewLayer(model.posts.View()))
	}

	view := tea.NewView(lipgloss.NewCanvas(model.width, model.height).Compose(compositor).Render())
	view.AltScreen = true

	return view
}

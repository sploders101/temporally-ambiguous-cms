package tabrowser

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller/modules/navigation"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/article"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/postlist"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/spinner"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/splash"
)

type ScreenIndex int

const (
	ScreenSplash ScreenIndex = iota
	ScreenPosts
	ScreenArticle
)

type ComponentIndex int

const (
	ComponentSplash ComponentIndex = iota
	ComponentPosts
	ComponentArticle
	ComponentSpinner
)

type Model struct {
	controllerIn  chan<- controller.ControllerCommand
	controllerOut <-chan controller.ControllerResponse
	width         int
	height        int
	currentScreen ScreenIndex
	splash        *splash.Model
	posts         *postlist.Model
	spinner       *spinner.Model
	article       *article.Model
}

func NewModel(ctx context.Context, controllers []controller.Controller) Model {
	controllerIn, controllerOut := controller.StartControllerLoop(ctx, controllers)
	model := Model{
		controllerIn:  controllerIn,
		controllerOut: controllerOut,
		currentScreen: ScreenSplash,
		splash:        splash.NewSplashScreen("Temporally Ambiguous Browser"),
		posts:         postlist.New(),
		spinner:       spinner.New(),
		article:       article.New(),
	}
	model.SetTheme(tabubbles.DefaultTheme)
	return model
}

func (model Model) Init() tea.Cmd {
	cmds := tabubbles.NewCmdCollector()
	cmds.Push(model.listenControllerResponse)
	model.splash.Init(cmds.WithIndex(int(ComponentSplash)))
	model.posts.Init(cmds.WithIndex(int(ComponentPosts)))
	model.spinner.Init(cmds.WithIndex(int(ComponentSpinner)))
	return cmds.ToCmd()
}

func (model *Model) SetSize(width int, height int) {
	model.width, model.height = width, height
	model.splash.SetSize(width, height)
	model.posts.SetSize(width, height)
}

func (model *Model) SetTheme(theme tabubbles.Theme) {
	model.splash.SetTheme(theme)
	model.posts.SetTheme(theme)
	model.spinner.SetTheme(theme)
	model.article.SetTheme(theme)
}

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := tabubbles.NewCmdCollector()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return model, tea.Quit
		}
		if model.spinner.GetVisibility() {
			return model, nil
		}
		switch model.currentScreen {
		case ScreenSplash:
			if msg.String() == "enter" {
				model.currentScreen = ScreenPosts
				model.posts.SetVisibility(true, cmds.WithIndex(int(ComponentPosts)))
			}
		case ScreenPosts:
			model.posts.Update(msg, cmds.WithIndex(int(ComponentPosts)))
		case ScreenArticle:
			// TODO
		}
	case tea.WindowSizeMsg:
		model.SetSize(msg.Width, msg.Height)
	case controller.ControllerCommand:
		model.controllerIn <- msg
	case controller.ControllerResponse:
		cmds.Push(model.listenControllerResponse)
		if index, ok, msg := msg.PopIndex(); ok {
			// The event was for someone else
			switch ComponentIndex(index) {
			case ComponentSplash:
				model.splash.Update(msg, cmds.WithIndex(int(ComponentSplash)))
			case ComponentPosts:
				model.posts.Update(msg, cmds.WithIndex(int(ComponentPosts)))
			case ComponentArticle:
				// TODO
			case ComponentSpinner:
				model.spinner.Update(msg, cmds.WithIndex(int(ComponentSpinner)))
			}
		} else {
			// The event was for us
			switch msg := msg.Msg.(type) {
			case navigation.DisplaySpinner:
				model.spinner.SetVisibility(msg.Visibility, cmds.WithIndex(int(ComponentSpinner)))
			case navigation.DisplayArticle:
				// TODO
			}
		}
	}
	return model, cmds.ToCmd()
}

func (model Model) listenControllerResponse() tea.Msg {
	update, ok := <-model.controllerOut
	if !ok {
		return nil
	}
	return update
}

func (model Model) View() tea.View {

	compositor := lipgloss.NewCompositor()
	switch model.currentScreen {
	case ScreenSplash:
		compositor.AddLayers(lipgloss.NewLayer(model.splash.View()))
	case ScreenPosts:
		compositor.AddLayers(lipgloss.NewLayer(model.posts.View()))
	case ScreenArticle:
		// TODO
	}

	if model.spinner.GetVisibility() {
		compositor.AddLayers(centerLayer(model.width, model.height, model.spinner.View()))
	}

	view := tea.NewView(lipgloss.NewCanvas(model.width, model.height).Compose(compositor).Render())
	view.AltScreen = true

	return view
}

func centerLayer(width, height int, layer string) *lipgloss.Layer {
	layerWidth, layerHeight := lipgloss.Size(layer)
	return lipgloss.NewLayer(layer).X(width/2 - layerWidth/2).Y(height/2 - layerHeight/2)
}

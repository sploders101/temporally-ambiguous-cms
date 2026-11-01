package tabrowser

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/bigstraw"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/article"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/postlist"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/splash"
)

type ScreenIndex int

const (
	ScreenSplash ScreenIndex = iota
	ScreenPosts
	ScreenArticle
)

const (
	ComponentSplash = iota
	ComponentPosts
	ComponentArticle
)

type Model struct {
	width         int
	height        int
	currentScreen ScreenIndex
	splash        *splash.Model
	posts         *postlist.Model
	article       *article.Model
}

func NewModel(ctx context.Context, client contentv1connect.ContentServiceClient) *Model {
	model := Model{
		currentScreen: ScreenSplash,
		splash: splash.NewModel(
			true,
			"Temporally Ambiguous Browser",
			"Press ENTER to continue.",
		),
		posts:   postlist.New(ctx, client),
		article: article.New(),
	}
	model.SetTheme(tabubbles.DefaultTheme)
	return &model
}

func (model *Model) SetSize(width int, height int, cmds bigstraw.CmdCollector) {
	model.width, model.height = width, height
	model.splash.SetSize(width, height, cmds)
	model.posts.SetSize(width, height, cmds)
}

func (model *Model) SetTheme(theme tabubbles.Theme) {
	model.splash.SetTheme(theme)
	model.posts.SetTheme(theme)
	model.article.SetTheme(theme)
}

func (model *Model) Update(msg tea.Msg, cmds bigstraw.CmdCollector) {
	cmds = cmds.DesignateAs("navigator")
	msg = bigstraw.HandleMsg(msg, cmds, map[int]bigstraw.Component{
		ComponentSplash:  model.splash,
		ComponentPosts:   model.posts,
		ComponentArticle: model.article,
	})
	switch msg := msg.(type) {
	case postlist.NavigateArticle:
	case tea.KeyMsg:
		model.HandleKeypress(msg, cmds)
	}
}

func (model *Model) HandleKeypress(msg tea.KeyMsg, cmds bigstraw.CmdCollector) {
	if msg.String() == "ctrl+c" {
		cmds.Add(tea.Quit)
		return
	}
	switch model.currentScreen {
	case ScreenSplash:
		if msg.String() == "enter" {
			model.currentScreen = ScreenPosts
			model.splash.SetVisibility(false, cmds.WithIndex(ComponentSplash))
			model.posts.SetVisibility(true, cmds.WithIndex(ComponentPosts))
		}
	case ScreenPosts:
		model.posts.Update(msg, cmds.WithIndex(ComponentPosts))
	case ScreenArticle:
		// TODO
	}
}

func (model *Model) View() tea.View {
	compositor := lipgloss.NewCompositor()
	switch model.currentScreen {
	case ScreenSplash:
		compositor.AddLayers(lipgloss.NewLayer(model.splash.View()))
	case ScreenPosts:
		compositor.AddLayers(lipgloss.NewLayer(model.posts.View()))
	case ScreenArticle:
		// TODO
	}

	view := tea.NewView(lipgloss.NewCanvas(model.width, model.height).Compose(compositor).Render())
	view.AltScreen = true

	return view
}

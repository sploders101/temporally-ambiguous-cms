package browser

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/bubbles/iotracker"
	"github.com/sploders101/personal-website/pkg/bubbles/splash"
	"github.com/sploders101/personal-website/pkg/tabrowser/articles"
)

type Component int

const (
	ComponentSplash Component = iota
	ComponentArticleSelect
)

type CMSModel struct {
	client        contentv1connect.ContentServiceClient
	currentView   Component
	splash        splash.Model
	articleSelect articles.Model
}

func NewCMSModel(client contentv1connect.ContentServiceClient) CMSModel {
	return CMSModel{
		client:      client,
		currentView: ComponentSplash,
		splash: splash.NewSplashScreen(
			"Temporally Ambiguous Browser",
			lipgloss.Color("#FF4000"),
		),
		articleSelect: articles.New(client),
	}
}

func (m CMSModel) Init() tea.Cmd {
	return tea.Batch(
		iotracker.TrackIO(int(ComponentArticleSelect), m.articleSelect.Init()),
	)
}

func (m CMSModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.currentView {
		case ComponentSplash:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "enter":
				m.currentView = ComponentArticleSelect
				return m, nil
			}
		case ComponentArticleSelect:
			newArticleSelect, cmd := m.articleSelect.Update(msg)
			m.articleSelect = newArticleSelect
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.splash.SetSize(msg.Width, msg.Height)
		m.articleSelect.SetSize(msg.Width, msg.Height)
	case iotracker.TrackedIO:
		switch Component(msg.Index) {
		case ComponentSplash:
			newSplash, cmd := m.splash.Update(msg.Cmd)
			m.splash = newSplash
			return m, cmd
		case ComponentArticleSelect:
			newSelect, cmd := m.articleSelect.Update(msg.Cmd)
			m.articleSelect = newSelect
			return m, cmd
		}
	}
	return m, nil
}

func (m CMSModel) View() tea.View {
	var view tea.View
	switch m.currentView {
	case ComponentSplash:
		view = m.splash.View()
	case ComponentArticleSelect:
		view = tea.NewView(m.articleSelect.View().Content)
	}
	view.AltScreen = true
	return view
}

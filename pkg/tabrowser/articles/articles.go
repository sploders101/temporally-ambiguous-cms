package articles

import (
	"context"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/bubbles/splash"
)

type errEvent error
type articleUpdateEvent []list.Item

type listableArticle struct {
	a *contentv1.ArticleMetadata
}

func (a listableArticle) FilterValue() string {
	return a.a.GetTitle()
}
func (a listableArticle) Title() string {
	return a.a.GetTitle()
}
func (a listableArticle) Description() string {
	return a.a.GetDescription()
}

type Model struct {
	client     contentv1connect.ContentServiceClient
	errVisible bool
	errView    splash.Model
	listStyle  lipgloss.Style
	list       list.Model
}

func New(client contentv1connect.ContentServiceClient) Model {
	return Model{
		client: client,
		errVisible: false,
		errView: splash.NewSplashScreen("", nil),
		listStyle: lipgloss.NewStyle().Margin(3),
		list:   list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		articles, err := m.client.ListArticles(
			context.TODO(),
			contentv1.ListArticlesRequest_builder{
				Offset: 0,
				Limit:  50,
			}.Build(),
		)
		if err != nil {
			return errEvent(err)
		}

		wrappedArticles := make([]list.Item, len(articles.GetArticles()))
		for i, article := range articles.GetArticles() {
			wrappedArticles[i] = listableArticle{article}
		}

		return articleUpdateEvent(wrappedArticles)
	}
}

func (m *Model) SetSize(width int, height int) {
	m.errView.SetSize(width, height)
	frameWidth, frameHeight := m.listStyle.GetFrameSize()
	m.list.SetSize(width - frameWidth, height - frameHeight)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errEvent:
		m.errVisible = true
		m.errView.SetText(msg.Error())
		return m, nil
	case articleUpdateEvent:
		return m, m.list.SetItems(msg)
	default:
		if m.errVisible {
			if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
				m.errVisible = false
			}
			return m, nil
		} else {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, cmd
		}
	}
}

func (m Model) View() tea.View {
	if m.errVisible {
		return m.errView.View()
	}
	return tea.NewView(m.listStyle.Render(m.list.View()))
}

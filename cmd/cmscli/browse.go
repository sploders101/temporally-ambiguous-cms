package main

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sploders101/personal-website/internal/env"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
)

// Browses the CMS with a TUI
func browseCMS(ctx context.Context, cfg Config) error {
	client := contentv1connect.NewContentServiceClient(
		env.ConnectHttp,
		cfg.Endpoint,
	)

	model := NewCMSModel(client)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

type CMSModel struct {
	client contentv1connect.ContentServiceClient
	width  int
	height int
}

func NewCMSModel(client contentv1connect.ContentServiceClient) CMSModel {
	return CMSModel{
		client: client,
	}
}

func (m CMSModel) Init() tea.Cmd {
	return nil
}

func (m CMSModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m CMSModel) View() tea.View {
	view := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "Hello world!"))
	view.AltScreen = true
	view.Cursor = tea.NewCursor(m.width/2, m.height/2)
	view.Cursor.Color = lipgloss.Color("#FF4000")
	return view
}

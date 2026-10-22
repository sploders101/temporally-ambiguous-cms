package posts

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

type Model struct {
	beforeTime   time.Time
	articlesList []*contentv1.ArticleMetadata
}

func New() Model {
	return Model{
		beforeTime: time.Now(),
	}
}

func (model Model) Init() tea.Cmd {
	return controller.CallController(controller.GetArticles{
		BeforeTime: time.Now(),
		Offset:     0,
		Limit:      50,
	})
}

func (model Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case controller.ControllerResponse:
		if _, ok, msg := msg.PopIndex(); !ok {
			switch msg := msg.Msg.(type) {
			case controller.UpdateArticles:
				model.articlesList = msg.Articles
			}
		}
	}

	return model, nil
}

func (model Model) View() string {
	return fmt.Sprintf("%v", model.articlesList)
}

package posts

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

const DEFAULT_CHUNK_SIZE = 50

type Model struct {
	width  int
	height int

	selectedIndex int

	chunkSize    int
	beforeTime   time.Time
	articlesList []*contentv1.ArticleMetadata

	gap             int
	containerStyle  lipgloss.Style
	unselectedStyle lipgloss.Style
	selectedStyle   lipgloss.Style
}

func New() Model {
	return Model{
		chunkSize:      DEFAULT_CHUNK_SIZE,
		beforeTime:     time.Now(),
		gap:            1,
		containerStyle: lipgloss.NewStyle().Margin(1),
		unselectedStyle: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#404040")),
		selectedStyle: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#FF4000")),
	}
}

func (model *Model) SetSize(width int, height int) {
	model.width = width
	model.height = height
}

func (model *Model) UpdateScroll() {
}

func (model Model) Init() tea.Cmd {
	return controller.CallController(controller.GetArticles{
		BeforeTime: time.Now(),
		Offset:     0,
		Limit:      model.chunkSize,
	})
}

func (model Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			model.selectedIndex -= 1
			if model.selectedIndex < 0 {
				if len(model.articlesList) == 0 {
					model.selectedIndex = 0
				} else {
					model.selectedIndex = len(model.articlesList) - 1
				}
			}
		case "down":
			model.selectedIndex += 1
			if model.selectedIndex >= len(model.articlesList) {
				model.selectedIndex = 0
			}
		}
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
	if len(model.articlesList) <= model.selectedIndex {
		return ""
	}

	layer := lipgloss.NewLayer("")

	frameWidth, frameHeight := model.containerStyle.GetFrameSize()
	containerHeight := model.height - frameHeight/2
	containerWidth := model.width - frameWidth/2
	height := containerHeight * 2
	topLines := containerHeight
	topLinesReal := containerHeight / 2
	bottomLines := containerHeight
	bottomLinesReal := containerHeight/2 + (containerHeight % 2)

	selectedArticle := model.renderArticleItem(model.articlesList[model.selectedIndex], true)
	selectedArticleHeight := lipgloss.Height(selectedArticle)
	selectedArticleHalfHeight := selectedArticleHeight / 2
	selectedArticleTopLinesOffset := selectedArticleHalfHeight + (selectedArticleHeight % 2)
	topLines -= selectedArticleTopLinesOffset
	topLinesReal -= selectedArticleTopLinesOffset
	bottomLines -= selectedArticleHalfHeight
	bottomLinesReal -= selectedArticleHalfHeight
	layer.AddLayers(lipgloss.NewLayer(selectedArticle).Y(topLines))

	index := model.selectedIndex + 1
	for bottomLines > 0 {
		if len(model.articlesList) <= index {
			break
		}
		articleText := model.renderArticleItem(model.articlesList[index], false)
		layer.AddLayers(
			lipgloss.NewLayer(articleText).Y(height - bottomLines + model.gap),
		)
		bottomLinesOffset := lipgloss.Height(articleText) + model.gap
		bottomLines -= bottomLinesOffset
		bottomLinesReal -= bottomLinesOffset
		index += 1
	}

	index = model.selectedIndex - 1
	for topLines > 0 {
		if index < 0 {
			break
		}
		articleText := model.renderArticleItem(model.articlesList[index], false)
		articleHeight := lipgloss.Height(articleText)
		layer.AddLayers(lipgloss.NewLayer(articleText).Y(topLines - articleHeight - model.gap))
		topLinesOffset := articleHeight + model.gap
		topLines -= topLinesOffset
		topLinesReal -= topLinesOffset
		index -= 1
	}

	var offset int
	if topLinesReal > 0 {
		offset = -topLinesReal - (containerHeight/2)
	} else if bottomLinesReal > 0 {
		offset = bottomLinesReal
	} else {
		offset = -(containerHeight / 2)
	}

	canvas := lipgloss.NewCanvas(containerWidth, containerHeight)
	canvas.Compose(
		lipgloss.NewCompositor(
			layer.Y(offset),
			lipgloss.NewLayer(fmt.Sprintf("(%v, %v, %v)", offset, topLinesReal, bottomLinesReal)),
		),
	)

	return model.containerStyle.MaxHeight(containerHeight).
		MaxWidth(containerWidth).
		Render(canvas.Render())
}

func (model Model) getArticleItemHeight(article *contentv1.ArticleMetadata, selected bool) int {
	// TODO: This can be improved. I don't need to render the styles to do this, but I'm not done
	// with article rendering yet, so this consolidates the logic.
	renderedArticle := model.renderArticleItem(article, selected)
	return lipgloss.Height(renderedArticle)
}

func (model Model) renderArticleItem(article *contentv1.ArticleMetadata, selected bool) string {
	frameWidth, _ := model.containerStyle.GetFrameSize()
	var style lipgloss.Style
	if selected {
		style = model.selectedStyle
	} else {
		style = model.unselectedStyle
	}
	cFrameWidth, _ := style.GetFrameSize()
	width := model.width - frameWidth - cFrameWidth

	title := strings.TrimSpace(article.GetTitle())
	title = lipgloss.Wrap(title, width, "")

	desc := strings.TrimSpace(article.GetDescription())
	desc = lipgloss.Wrap(desc, width, "")
	if selected {
		return model.selectedStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left, title, desc),
		)
	} else {
		return model.unselectedStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left, title, desc),
		)
	}
}

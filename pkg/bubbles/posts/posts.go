package posts

import (
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

	frameWidth, frameHeight := model.containerStyle.GetFrameSize()
	containerWidth, containerHeight := model.width-frameWidth, model.height-frameHeight

	heightRemainder := containerHeight % 2
	var articlesAbove []string
	linesAbove := containerHeight / 2
	var articlesBelow []string
	linesBelow := containerHeight/2 + heightRemainder
	var totalContentHeight int

	// Initial centered layout. To be adjusted.
	selectedArticle := model.renderArticleItem(model.articlesList[model.selectedIndex], true)
	selectedArticleHeight := lipgloss.Height(selectedArticle)
	selectedArticleHeightHalf := selectedArticleHeight / 2
	selectedArticleHeightRemainder := selectedArticleHeight % 2
	if heightRemainder == 1 && selectedArticleHeightRemainder == 1 {
		// Prefer downward movement to compensate for double-remainders
		linesAbove -= selectedArticleHeightHalf
		linesBelow -= selectedArticleHeightHalf + selectedArticleHeightRemainder
	} else {
		linesAbove -= selectedArticleHeightHalf + selectedArticleHeightRemainder
		linesBelow -= selectedArticleHeightHalf
	}
	// Offset will determine where we start rendering. This is the Y coordinate for the selected article.
	// All other articles will spread outward from there.
	offset := linesAbove
	totalContentHeight = selectedArticleHeight

	// Using a flip-flop here so we can continue the loop in whatever direction is needed to fill the dead space.
	// I'm not totally sold on this design yet, but I've never really done much layout work before...
	var direction bool
	var downDone bool
	var upDone bool
	for (linesAbove+linesBelow > 0) && !(downDone && upDone) {
		if direction {
			// Go down
			newArticleIndex := model.selectedIndex + 1 + len(articlesBelow)
			if len(model.articlesList) <= newArticleIndex {
				downDone = true
				direction = !direction
				continue
			}
			newArticle := model.renderArticleItem(model.articlesList[newArticleIndex], false)
			newArticleHeight := lipgloss.Height(newArticle)
			totalContentHeight += (newArticleHeight + model.gap)
			linesBelow -= (newArticleHeight + model.gap)
			articlesBelow = append(articlesBelow, newArticle)
		} else {
			// Go up
			newArticleIndex := model.selectedIndex - 1 - len(articlesAbove)
			if newArticleIndex < 0 {
				upDone = true
				direction = !direction
				continue
			}
			newArticle := model.renderArticleItem(model.articlesList[newArticleIndex], false)
			newArticleHeight := lipgloss.Height(newArticle)
			totalContentHeight += (newArticleHeight + model.gap)
			linesAbove -= (newArticleHeight + model.gap)
			articlesAbove = append(articlesAbove, newArticle)
		}
		direction = !direction
	}

	if linesAbove > 0 || totalContentHeight < containerHeight {
		offset -= linesAbove
	} else if linesBelow > 0 {
		offset += linesBelow
	}

	compositor := lipgloss.NewCompositor()

	compositor.AddLayers(lipgloss.NewLayer(selectedArticle).Y(offset))

	aboveOffset := offset
	for _, article := range articlesAbove {
		articleHeight := lipgloss.Height(article)
		compositor.AddLayers(lipgloss.NewLayer(article).Y(aboveOffset - model.gap - articleHeight))
		aboveOffset -= (model.gap + articleHeight)
	}

	belowOffset := offset + selectedArticleHeight
	for _, article := range articlesBelow {
		articleHeight := lipgloss.Height(article)
		compositor.AddLayers(lipgloss.NewLayer(article).Y(belowOffset + model.gap))
		belowOffset += (model.gap + articleHeight)
	}

	return model.containerStyle.Render(
		lipgloss.NewCanvas(containerWidth, containerHeight).Compose(compositor).Render(),
	)
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

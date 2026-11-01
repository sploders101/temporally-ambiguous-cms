package postlist

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/bigstraw"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles"
	"github.com/sploders101/personal-website/pkg/tabrowser/tabubbles/spinner"
	"github.com/sploders101/personal-website/pkg/tabrowser/utils"
)

const DEFAULT_CHUNK_SIZE = 50

const (
	ComponentSpinner int = iota
)

type Model struct {
	ctx    context.Context
	client contentv1connect.ContentServiceClient

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
	postTitleStyle  lipgloss.Style
	postDescStyle   lipgloss.Style

	spinnerVisible bool
	spinner        *spinner.Model
}

type articlesList struct {
	articles []*contentv1.ArticleMetadata
}
type NavigateArticle struct {
	Slug string
}

func New(ctx context.Context, client contentv1connect.ContentServiceClient) *Model {
	model := &Model{
		ctx:            ctx,
		client:         client,
		chunkSize:      DEFAULT_CHUNK_SIZE,
		beforeTime:     time.Now(),
		gap:            1,
		spinnerVisible: true,
		spinner:        spinner.New(),
	}
	return model
}

func (model *Model) SetSize(width int, height int, cmds bigstraw.CmdCollector) {
	model.width, model.height = width, height
}

func (model *Model) SetTheme(theme tabubbles.Theme) {
	model.containerStyle = lipgloss.NewStyle().Margin(1)
	model.unselectedStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(theme.GetUnfocusedPostColor())
	model.selectedStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(theme.GetFocusedPostColor())
	model.postTitleStyle = lipgloss.NewStyle().Underline(true).Foreground(theme.GetPostTitleColor())
	model.postDescStyle = lipgloss.NewStyle().Foreground(theme.GetPostDescriptionColor())
	model.spinner.SetTheme(theme)
}

func (model *Model) SetVisibility(visible bool, cmds bigstraw.CmdCollector) {
	model.spinner.SetVisibility(visible && model.spinnerVisible, cmds.WithIndex(ComponentSpinner))
}

func (model *Model) Update(msg tea.Msg, cmds bigstraw.CmdCollector) {
	msg = bigstraw.HandleMsg(msg, cmds, map[int]bigstraw.Component{
		ComponentSpinner: model.spinner,
	})
	switch msg := msg.(type) {
	case bigstraw.InitMsg:
		cmds.AddNotifySelf(model.fetchArticlesCmd())
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if model.selectedIndex-1 < 0 {
				model.selectedIndex = 0
			} else {
				model.selectedIndex -= 1
			}
		case "down", "j":
			if model.selectedIndex+1 >= len(model.articlesList) {
				model.selectedIndex = len(model.articlesList) - 1
			} else {
				model.selectedIndex += 1
			}
		case "enter":
			if len(model.articlesList) > model.selectedIndex {
				article := model.articlesList[model.selectedIndex]
				cmds.AddNotifyRole("navigator", func() tea.Msg {
					return NavigateArticle{
						Slug: article.GetSlug(),
					}
				})
			}
		}
	case articlesList:
		model.spinnerVisible = false
		model.spinner.SetVisibility(false, cmds.WithIndex(ComponentSpinner))
		model.articlesList = msg.articles
	}
}

func (model *Model) fetchArticlesCmd() tea.Cmd {
	ctx := model.ctx
	client := model.client
	chunkSize := model.chunkSize
	return func() tea.Msg {
		articles, err := client.ListArticles(ctx, contentv1.ListArticlesRequest_builder{
			Offset: 0,
			Limit:  int32(chunkSize),
		}.Build())
		if err != nil {
			// TODO: report error
			return nil
		}
		return articlesList{
			articles: articles.GetArticles(),
		}
	}
}

func (model Model) View() string {
	width := model.width - model.containerStyle.GetHorizontalMargins()
	height := model.height - model.containerStyle.GetVerticalMargins()
	frameWidth, frameHeight := model.containerStyle.GetFrameSize()
	containerWidth, containerHeight := model.width-frameWidth, model.height-frameHeight
	compositor := lipgloss.NewCompositor()

	if len(model.articlesList) > model.selectedIndex {
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
		linesAbove -= selectedArticleHeightHalf + selectedArticleHeight%2
		linesBelow -= selectedArticleHeightHalf
		// Offset will determine where we start rendering. This is the Y coordinate for the selected article.
		// All other articles will spread outward from there.
		offset := linesAbove
		totalContentHeight = selectedArticleHeight

		// Using a flip-flop here so we can continue the loop in whatever direction is needed to fill the dead space.
		// I'm not totally sold on this design yet, but I've never really done much layout work before...
		var direction bool
		var downDone bool
		var upDone bool
		for (linesAbove+linesBelow > 0 || direction) && !(downDone && upDone) {
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

		compositor.AddLayers(lipgloss.NewLayer(selectedArticle).Y(offset))

		aboveOffset := offset
		for _, article := range articlesAbove {
			articleHeight := lipgloss.Height(article)
			compositor.AddLayers(
				lipgloss.NewLayer(article).Y(aboveOffset - model.gap - articleHeight),
			)
			aboveOffset -= (model.gap + articleHeight)
		}

		belowOffset := offset + selectedArticleHeight
		for _, article := range articlesBelow {
			articleHeight := lipgloss.Height(article)
			compositor.AddLayers(lipgloss.NewLayer(article).Y(belowOffset + model.gap))
			belowOffset += (model.gap + articleHeight)
		}
	}

	if model.spinner.GetVisibility() {
		compositor.AddLayers(utils.CenterLayer(width, height, model.spinner.View()))
	}

	return model.containerStyle.
		Width(width).
		Height(height).
		Render(lipgloss.NewCanvas(containerWidth, containerHeight).Compose(compositor).Render())
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
	title = model.postTitleStyle.Render(title)

	desc := strings.TrimSpace(article.GetDescription())
	desc = lipgloss.Wrap(desc, width, "")
	desc = model.postDescStyle.Render(desc)
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

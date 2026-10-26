package browsing

import (
	"context"
	"log/slog"

	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

type ArticleIndexController struct {
	ctx      context.Context
	respChan chan<- controller.ControllerResponse
	client   contentv1connect.ContentServiceClient
}

func NewArticleIndexController(
	client contentv1connect.ContentServiceClient,
) *ArticleIndexController {
	return &ArticleIndexController{client: client}
}

func (aic *ArticleIndexController) Init(
	ctx context.Context,
	respChan chan<- controller.ControllerResponse,
) {
	aic.ctx = ctx
	aic.respChan = respChan
}

func (aic *ArticleIndexController) Handle(req controller.ControllerCommand) {
	switch msg := req.Msg.(type) {
	case GetArticles:
		articles, err := aic.client.ListArticles(aic.ctx, contentv1.ListArticlesRequest_builder{
			Offset: int32(msg.Offset),
			Limit:  int32(msg.Limit),
		}.Build())
		if err != nil {
			slog.Error("Error fetching articles", "error", err)
			return
		}

		aic.respChan <- controller.ControllerResponse{
			IndexChain: req.IndexChain,
			Msg: UpdateArticles{
				GetArticles: msg,
				Articles:    articles.GetArticles(),
			},
		}
	}
}

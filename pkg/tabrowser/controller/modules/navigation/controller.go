package navigation

import (
	"context"
	"sync"

	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

// Controller //

type NavigationController struct {
	ctx      context.Context
	respChan chan<- controller.ControllerResponse
	client   contentv1connect.ContentServiceClient
	navMutex sync.Mutex
}

func NewNavigationController(client contentv1connect.ContentServiceClient) *NavigationController {
	return &NavigationController{client: client}
}

func (nav *NavigationController) Init(
	ctx context.Context,
	respChan chan<- controller.ControllerResponse,
) {
	nav.ctx = ctx
	nav.respChan = respChan
}

func (nav *NavigationController) Handle(req controller.ControllerCommand) {
	switch msg := req.Msg.(type) {
	case NavigateArticle:
		nav.navMutex.Lock()
		defer nav.navMutex.Unlock()
		nav.respChan <- controller.ControllerResponse{
			Msg: DisplaySpinner{
				Visibility: true,
			},
		}
		article, err := nav.client.GetArticle(nav.ctx, contentv1.GetArticleRequest_builder{
			Slug: msg.Slug,
		}.Build())
		if err != nil {
			// TODO: Present to user
		}
		nav.respChan <- controller.ControllerResponse{
			Msg: DisplayArticle{
				Meta:      article.GetArticleMetadata(),
				MdContent: article.GetBody(),
			},
		}
		nav.respChan <- controller.ControllerResponse{
			Msg: DisplaySpinner{
				Visibility: false,
			},
		}
	}
}

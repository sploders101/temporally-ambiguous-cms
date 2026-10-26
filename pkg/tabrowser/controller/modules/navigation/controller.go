package navigation

import (
	"context"

	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

// Controller //

type NavigationController struct {
	ctx      context.Context
	respChan chan<- controller.ControllerResponse
	client   contentv1connect.ContentServiceClient
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
	
}

package utilsctrl

import (
	"context"
	"sync"
	"time"

	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
)

type UtilityController struct {
	ctx      context.Context
	respChan chan<- controller.ControllerResponse
}

func NewUtilityController() *UtilityController {
	return &UtilityController{}
}

func (util *UtilityController) Init(
	ctx context.Context,
	respChan chan<- controller.ControllerResponse,
) {
	util.ctx = ctx
	util.respChan = respChan
}

func (util *UtilityController) Handle(wg *sync.WaitGroup, req controller.ControllerCommand) bool {
	switch msg := req.Msg.(type) {
	case TickerRequest:
		wg.Go(func() { util.HandleTickerRequest(req, msg) })
		return true
	}
	return false
}

func (util *UtilityController) HandleTickerRequest(
	req controller.ControllerCommand,
	msg TickerRequest,
) {
	if msg.Time != (time.Time{}) {
		select {
		case <-util.ctx.Done():
		case <-msg.Cancel:
		case <-time.After(time.Until(msg.Time)):
			util.respChan <- controller.ControllerResponse{
				IndexChain: req.IndexChain,
				Msg:        TickerResponse{Msg: msg.Msg},
			}
		}
	} else {
		select {
		case <-util.ctx.Done():
		case <-msg.Cancel:
		case <-time.After(msg.Duration):
			util.respChan <- controller.ControllerResponse{
				IndexChain: req.IndexChain,
				Msg:        TickerResponse{Msg: msg.Msg},
			}
		}
	}
}

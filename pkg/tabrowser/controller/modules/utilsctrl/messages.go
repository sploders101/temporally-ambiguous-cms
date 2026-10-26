package utilsctrl

import (
	"time"
)

// Direction: controller<-
type TickerRequest struct {
	Cancel chan struct{} // can be used to abort the ticker
	Time time.Time
	Duration time.Duration
	Msg any
}

// Direction: <-controller
type TickerResponse struct {
	Msg any
}

package browsing

import (
	"time"

	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
)

// Direction: controller<-
//
// Requests articles to be fetched for display
type GetArticles struct {
	BeforeTime time.Time
	Offset     int
	Limit      int
}

// Direction: <-controller
//
// Updates articles for display
type UpdateArticles struct {
	GetArticles

	Articles []*contentv1.ArticleMetadata
}

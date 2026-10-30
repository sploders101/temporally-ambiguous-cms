package navigation

import contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"

// Direction: controller<-
//
// Begins navigation to the requested article
type NavigateArticle struct {
	Slug string
}

// Direction: <-controller
//
// Indicates that a request to navigate has been received, and the article is being fetched
type DisplaySpinner struct {
	Visibility bool
}

// Direction: <-controller
//
// The requested article has been fetched and should now be displayed.
type DisplayArticle struct {
	Meta      *contentv1.ArticleMetadata
	MdContent string
}

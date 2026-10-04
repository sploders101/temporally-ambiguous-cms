package ht

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
	"github.com/sploders101/personal-website/internal/markdown"
)

type postTemplateVars struct {
	baseTemplateVars

	Slug         string
	Title        string
	Description  string
	PublishedAt  time.Time
	PostContents template.HTML
}

var ErrPostNotFound = errors.New("post not found")

func getPostTemplateConfig(
	cfg config.ServerConfig,
	req *http.Request,
	db *queries.Queries,
) (postTemplateVars, error) {
	ctx := req.Context()
	baseVars, err := getBasePageConfig(cfg, req)
	if err != nil {
		return postTemplateVars{}, err
	}

	var article queries.Article
	var revision queries.ArticlesRevision
	if slug := req.PathValue("slug"); slug != "" {
		articleContainer, err := db.GetPublishedRevisionBySlug(ctx, slug)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return postTemplateVars{}, ErrPostNotFound
			}
			return postTemplateVars{}, err
		}
		article = articleContainer.Article
		revision = articleContainer.ArticlesRevision
	} else if articleId := req.PathValue("articleId"); articleId != "" {
		articleUuid, err := uuid.Parse(articleId)
		if err != nil {
			return postTemplateVars{}, ErrPostNotFound
		}
		articleContainer, err := db.GetPublishedRevisionByUuid(ctx, articleUuid)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return postTemplateVars{}, ErrPostNotFound
			}
			return postTemplateVars{}, err
		}
		article = articleContainer.Article
		revision = articleContainer.ArticlesRevision
	} else {
		return postTemplateVars{}, ErrPostNotFound
	}

	_, htmlContents, err := markdown.Render(
		[]byte(revision.Body),
		markdown.RenderOptions{
			EnableXHTML: false,
			Highlight: markdown.HighlightOptions{
				InlineStyles: true,
			},
		},
	)
	if err != nil {
		return postTemplateVars{}, fmt.Errorf("failed to render markdown: %w", err)
	}

	return postTemplateVars{
		baseTemplateVars: baseVars,
		Slug:             article.Slug,
		Title:            revision.Title,
		Description:      revision.Description,
		PublishedAt:      revision.PublishedAt.Time,
		PostContents:     template.HTML(htmlContents),
	}, nil
}

type postFeedTemplateVars struct {
	baseTemplateVars

	Posts []postFeedPost
}

type postFeedPost struct {
	Url         string
	Slug        string
	Title       string
	Description string
	CreatedAt   time.Time
}

func getPostFeedTemplateConfig(
	cfg config.ServerConfig,
	req *http.Request,
	db *queries.Queries,
) (postFeedTemplateVars, error) {
	ctx := req.Context()
	baseVars, err := getBasePageConfig(cfg, req)
	if err != nil {
		return postFeedTemplateVars{}, err
	}

	dbPosts, err := db.GetArticleFeed(ctx, queries.GetArticleFeedParams{
		// TODO: Pagination & filtering
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		return postFeedTemplateVars{}, err
	}
	posts := make([]postFeedPost, len(dbPosts))
	for i, dbPost := range dbPosts {
		url, err := url.JoinPath(cfg.BaseUrl, "posts", dbPost.Article.Slug, "/")
		if err != nil {
			return postFeedTemplateVars{}, err
		}
		posts[i] = postFeedPost{
			Url:         url,
			Slug:        dbPost.Article.Slug,
			Title:       dbPost.ArticlesRevision.Title,
			Description: dbPost.ArticlesRevision.Description,
			CreatedAt:   dbPost.ArticlesRevision.CreatedAt,
		}
	}

	return postFeedTemplateVars{
		baseTemplateVars: baseVars,
		Posts:            posts,
	}, nil
}

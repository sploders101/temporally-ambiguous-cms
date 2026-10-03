package postfeeds

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gorilla/feeds"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
)

func GenerateDiscoveryFeed(
	ctx context.Context,
	cfg config.ServerConfig,
	db *queries.Queries,
) (*feeds.Feed, error) {
	feed := &feeds.Feed{
		Title:       cfg.SiteSettings.Title,
		Description: cfg.SiteSettings.Description,
		Link:        &feeds.Link{Href: cfg.BaseUrl},
	}
	if cfg.SiteSettings.Author != (config.SiteAuthor{}) {
		feed.Author = &feeds.Author{
			Name:  cfg.SiteSettings.Author.Name,
			Email: cfg.SiteSettings.Author.Email,
		}
	}
	if cfg.SiteSettings.ImagePath != "" {
		imageUrl, err := url.JoinPath(cfg.BaseUrl, cfg.SiteSettings.ImagePath)
		if err != nil {
			return nil, err
		}
		feed.Image = &feeds.Image{Url: imageUrl}
	}

	postsList, err := db.GetArticleFeed(ctx, queries.GetArticleFeedParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		return nil, err
	}

	feed.Items = make([]*feeds.Item, len(postsList))
	for i, post := range postsList {
		// TODO: embed content for the 5 most recent posts.
		itemUrl, err := url.JoinPath(cfg.BaseUrl, "posts", post.Article.Slug)
		if err != nil {
			return nil, err
		}
		feed.Items[i] = &feeds.Item{
			Title: post.ArticlesRevision.Title,
			// Maybe source this from the db? Not sure. This isn't really designed to be multi-tenant, even if the
			// articles do hold an author field. It's really more for fallback permissions checks...
			Author:      feed.Author,
			Description: post.ArticlesRevision.Description,
			Updated:     post.ArticlesRevision.CreatedAt,
			Created:     post.OriginalPublish,
			Link:        &feeds.Link{Href: itemUrl},
		}
	}

	return feed, nil
}

func ServeRSS(ctx context.Context, cfg config.ServerConfig, db dbapi.Db) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		feed, err := GenerateDiscoveryFeed(ctx, cfg, db.Query())
		if err != nil {
			slog.Error("Failed to generate discovery feed", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "application/rss+xml")
		if err := feed.WriteRss(resp); err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func ServeAtom(ctx context.Context, cfg config.ServerConfig, db dbapi.Db) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		feed, err := GenerateDiscoveryFeed(ctx, cfg, db.Query())
		if err != nil {
			slog.Error("Failed to generate discovery feed", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "application/atom+xml")
		if err := feed.WriteAtom(resp); err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func ServeJSON(ctx context.Context, cfg config.ServerConfig, db dbapi.Db) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		feed, err := GenerateDiscoveryFeed(ctx, cfg, db.Query())
		if err != nil {
			slog.Error("Failed to generate discovery feed", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "application/feed+json")
		if err := feed.WriteJSON(resp); err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

package apiservices

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
	"github.com/sploders101/personal-website/cmd/webserver/storage"
	contentv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1"
)

type ContentService struct {
	config        config.ServerConfig
	db            dbapi.Db
	storageDriver storage.StorageDriver
}

func NewContentService(
	config config.ServerConfig,
	db dbapi.Db,
	storageDriver storage.StorageDriver,
) ContentService {
	return ContentService{
		config:        config,
		db:            db,
		storageDriver: storageDriver,
	}
}

func (content ContentService) ListArticles(
	ctx context.Context,
	req *contentv1.ListArticlesRequest,
) (*contentv1.ListArticlesResponse, error) {
	if req.GetLimit() > 100 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("limit must not be >100"),
		)
	}

	dbArticles, err := content.db.Query().GetArticleFeed(ctx, queries.GetArticleFeedParams{
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	})
	if err != nil {
		return nil, ErrAmbiguousInternal
	}

	articles := make([]*contentv1.ArticleMetadata, len(dbArticles))
	for i, article := range dbArticles {
		articles[i] = contentv1.ArticleMetadata_builder{
			Id:          article.Article.PublicID.String(),
			Slug:        article.Article.Slug,
			Title:       article.ArticlesRevision.Title,
			Author:      content.config.SiteSettings.Author.Name,
			Description: article.ArticlesRevision.Description,
		}.Build()
	}

	return contentv1.ListArticlesResponse_builder{
		Articles: articles,
	}.Build(), nil
}

func (content ContentService) GetArticle(
	ctx context.Context,
	req *contentv1.GetArticleRequest,
) (*contentv1.GetArticleResponse, error) {
	dbArticle, err := content.db.Query().GetPublishedRevisionBySlug(ctx, req.GetSlug())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		slog.Error("Failed to get published revision", "slug", req.GetSlug(), "error", err)
		return nil, ErrAmbiguousInternal
	}

	return contentv1.GetArticleResponse_builder{
		ArticleMetadata: contentv1.ArticleMetadata_builder{
			Id:          dbArticle.Article.PublicID.String(),
			Slug:        dbArticle.Article.Slug,
			Title:       dbArticle.ArticlesRevision.Title,
			Author:      content.config.SiteSettings.Author.Name,
			Description: dbArticle.ArticlesRevision.Description,
		}.Build(),
		Body: dbArticle.ArticlesRevision.Body,
	}.Build(), nil
}

package apiservices

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"path/filepath"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
	"github.com/sploders101/personal-website/cmd/webserver/storage"
	"github.com/sploders101/personal-website/internal/authutils"
	cmsv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/cms/v1"
	"github.com/sploders101/personal-website/internal/markdown"
)

// Note: this entire service should be authenticated with JWTs.
// Handlers rely on accurate user claims being in the context.
type CmsService struct {
	config        config.ServerConfig
	db            dbapi.Db
	storageDriver storage.StorageDriver
}

func NewCmsService(
	config config.ServerConfig,
	db dbapi.Db,
	storageDriver storage.StorageDriver,
) CmsService {
	return CmsService{
		config:        config,
		db:            db,
		storageDriver: storageDriver,
	}
}

func (cms CmsService) Ping(
	ctx context.Context,
	req *cmsv1.PingRequest,
) (*cmsv1.PingResponse, error) {
	return cmsv1.PingResponse_builder{
		Message: req.GetMessage(),
		UserId:  authutils.MustGetClaims(ctx).Subject,
	}.Build(), nil
}

func (cms CmsService) SeedArticle(
	ctx context.Context,
	req *cmsv1.SeedArticleRequest,
) (*cmsv1.SeedArticleResponse, error) {
	source := req.GetMarkdown()
	frontmatter, _, err := markdown.Render([]byte(source), markdown.RenderOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Get user info for permission enforcement
	userClaims := authutils.MustGetClaims(ctx)
	uid, err := uuid.Parse(userClaims.Subject)
	if err != nil {
		return nil, ErrAmbiguousInternal
	}

	// Start db transaction
	tx, err := cms.db.Begin(ctx)
	if err != nil {
		slog.Error("Failed to open database transaction", "error", err)
		return nil, ErrAmbiguousInternal
	}
	defer tx.Rollback()

	// Create article entry (if necessary)
	article, err := tx.Query().CreateArticle(ctx, queries.CreateArticleParams{
		Author: uuid.NullUUID{Valid: true, UUID: uid},
		Slug:   frontmatter.Slug,
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("Failed to create article", "error", err)
			return nil, ErrAmbiguousInternal
		}
		article, err = tx.Query().GetArticleBySlug(ctx, frontmatter.Slug)
		if err != nil {
			slog.Error("Failed to get existing article", "error", err)
			return nil, ErrAmbiguousInternal
		}
	}
	if article.Author.UUID != uid {
		return nil, ErrPermissionDenied
	}

	// Create revision
	revision, err := tx.Query().StageArticleRevision(ctx, queries.StageArticleRevisionParams{
		ArticleID:   article.ID,
		Title:       frontmatter.Title,
		Description: frontmatter.Description,
		Body:        source,
	})
	if err != nil {
		slog.Error("Failed to create article revision", "error", err)
		return nil, ErrAmbiguousInternal
	}

	// Link assets
	for _, asset := range req.GetAssets() {
		// Check for non-local paths. These will break rendering.
		if !filepath.IsLocal(asset.GetFilename()) {
			return nil, connect.NewError(
				connect.CodeInvalidArgument,
				fmt.Errorf("file %q is not local", asset.GetFilename()),
			)
		}
		if err := tx.Query().LinkArticleAsset(ctx, queries.LinkArticleAssetParams{
			RevisionID: revision.ID,
			Sha512Hash: asset.GetSha512Hash(),
			FileName:   path.Clean(asset.GetFilename()),
		}); err != nil {
			slog.Error("Failed to link article asset", "error", err)
			return nil, ErrAmbiguousInternal
		}
	}

	// Build list of missing assets
	var missingAssets []*cmsv1.AssetDescriptor
	dbMissingAssets, err := tx.Query().GetMissingArticleAssets(ctx, revision.ID)
	if err != nil {
		slog.Error("Failed to list missing assets", "error", err)
		return nil, ErrAmbiguousInternal
	}
	for _, dbAsset := range dbMissingAssets {
		for _, asset := range req.GetAssets() {
			if path.Clean(asset.GetFilename()) == dbAsset.FileName {
				missingAssets = append(missingAssets, asset)
			}
		}
	}

	// Commit & respond
	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		return nil, ErrAmbiguousInternal
	}
	return cmsv1.SeedArticleResponse_builder{
		RevisionId:    revision.PublicID.String(),
		MissingAssets: missingAssets,
	}.Build(), nil
}

func (cms CmsService) PushAsset(
	ctx context.Context,
	req *connect.ClientStream[cmsv1.PushAssetRequest],
) (*cmsv1.PushAssetResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !req.Receive() {
		return nil, ErrProtocolViolation
	}
	header := req.Msg()
	if !header.HasDescriptor() {
		return nil, ErrProtocolViolation
	}
	descriptor := header.GetDescriptor()

	// Open database transaction
	tx, err := cms.db.Begin(ctx)
	if err != nil {
		slog.Error("Failed to open database connection", "error", err)
		return nil, ErrAmbiguousInternal
	}
	defer tx.Rollback()

	// Insert the asset into the db
	if err := tx.Query().CreateAsset(ctx, queries.CreateAssetParams{
		Sha512Hash:    descriptor.GetSha512Hash(),
		ContentType:   descriptor.GetContentType(),
		ContentLength: descriptor.GetContentLength(),
	}); err != nil {
		slog.Error("Error creating asset in db", "error", err)
		return nil, ErrAmbiguousInternal
	}

	// Set up adapter for storage driver to read the asset
	type FallbackError struct {
		error
	}
	writerResp := make(chan error, 1)
	assetReader, assetWriter := io.Pipe()
	defer func() { _ = assetReader.Close() }()
	go func() {
		defer func() { _ = assetWriter.Close() }()
		hasher := sha512.New()
		var byteCounter int64
		for req.Receive() {
			msg := req.Msg()
			if !msg.HasChunk() {
				writerResp <- ErrProtocolViolation
				return
			}
			chunk := msg.GetChunk()

			// Check if chunk puts us over our limit
			byteCounter += int64(len(chunk))
			if byteCounter > descriptor.GetContentLength() {
				writerResp <- ErrSizeMismatch
				return
			}

			// Write chunk
			_, _ = hasher.Write(chunk) // Never returns an error according to docs
			if _, err := assetWriter.Write(chunk); err != nil {
				writerResp <- FallbackError{err}
				return
			}
		}
		hash := hasher.Sum(nil)
		if !slices.Equal(descriptor.GetSha512Hash(), hash) {
			writerResp <- ErrHashMismatch
			return
		}
		writerResp <- nil
	}()

	// Upload asset to storage driver
	objectName := "SHA512:" + hex.EncodeToString(descriptor.GetSha512Hash())
	if err := cms.storageDriver.PutFile(
		ctx,
		objectName,
		descriptor.GetContentLength(),
		assetReader,
	); err != nil {
		// Check for more specific error from writer task
		select {
		case err := <-writerResp:
			if err != nil {
				// Don't return fallback errors.
				// They only matter if we don't make it to this block.
				// Linter disabled: The entire scope of this error is known. It will not be wrapped.
				if _, ok := err.(FallbackError); !ok { //nolint:errorlint
					return nil, err
				}
			}
		default:
		}
		// Fall back to driver error
		slog.Error("Error while uploading asset", "error", err)
		return nil, ErrAmbiguousInternal
	}

	// Watch for writer errors
	if err := <-writerResp; err != nil {
		if err := cms.storageDriver.DeleteFile(ctx, objectName); err != nil {
			slog.Warn(
				"Failed to clean up after failed upload",
				"objectName",
				objectName,
				"error",
				err,
			)
		}
		// Handle fallback errors separately
		// Linter disabled: The entire scope of this error is known. It will not be wrapped.
		if err, ok := err.(FallbackError); ok { // nolint:errorlint
			slog.Error("Error writing asset to upload pipe", "error", err.error)
			return nil, ErrAmbiguousInternal
		}
		// Writer will log internally if it should
		return nil, err
	}

	// Commit & respond
	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		if err := cms.storageDriver.DeleteFile(ctx, objectName); err != nil {
			slog.Warn(
				"Failed to clean up after failed upload",
				"objectName",
				objectName,
				"error",
				err,
			)
		}
		return nil, ErrAmbiguousInternal
	}
	return cmsv1.PushAssetResponse_builder{}.Build(), nil
}

func (cms CmsService) PublishArticle(
	ctx context.Context,
	req *cmsv1.PublishArticleRequest,
) (*cmsv1.PublishArticleResponse, error) {
	tx, err := cms.db.Begin(ctx)
	if err != nil {
		slog.Error("Failed to open database transaction", "error", err)
		return nil, ErrAmbiguousInternal
	}
	defer tx.Rollback()

	// Get user info for permission enforcement
	userClaims := authutils.MustGetClaims(ctx)
	uid, err := uuid.Parse(userClaims.Subject)
	if err != nil {
		return nil, ErrAmbiguousInternal
	}

	revisionId, err := uuid.Parse(req.GetRevisionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid revision id"))
	}
	article, err := tx.Query().GetArticleRevision(ctx, revisionId)
	if err != nil {
		return nil, ErrAmbiguousInternal
	}
	if article.Article.Author.UUID != uid {
		return nil, ErrPermissionDenied
	}

	// Check that we have all the necessary assets first
	missingAssets, err := tx.Query().GetMissingArticleAssetsByUuid(ctx, revisionId)
	if err != nil {
		slog.Error("Failed to query missing article assets", "error", err)
		return nil, ErrAmbiguousInternal
	}
	if len(missingAssets) != 0 {
		missingAssetNames := make([]string, len(missingAssets))
		for i, asset := range missingAssets {
			missingAssetNames[i] = asset.FileName
		}
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf("missing assets: %v", missingAssetNames),
		)
	}

	if err := tx.Query().PublishArticleRevision(ctx, queries.PublishArticleRevisionParams{
		PublicID:    revisionId,
		PublishedAt: sql.NullTime{Valid: true, Time: time.Now()},
	}); err != nil {
		return nil, ErrAmbiguousInternal
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit database transaction", "error", err)
		return nil, ErrAmbiguousInternal
	}
	return cmsv1.PublishArticleResponse_builder{}.Build(), nil
}

func (cms CmsService) RedactArticle(
	ctx context.Context,
	req *cmsv1.RedactArticleRequest,
) (*cmsv1.RedactArticleResponse, error) {
	tx, err := cms.db.Begin(ctx)
	if err != nil {
		slog.Error("Failed to open database transaction", "error", err)
		return nil, ErrAmbiguousInternal
	}
	defer tx.Rollback()

	// Get user info for permission enforcement
	userClaims := authutils.MustGetClaims(ctx)
	uid, err := uuid.Parse(userClaims.Subject)
	if err != nil {
		return nil, ErrAmbiguousInternal
	}
	article, err := tx.Query().GetArticleBySlug(ctx, req.GetSlug())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		slog.Error("Failed to get article by slug", "error", err)
		return nil, ErrAmbiguousInternal
	}
	if article.Author.UUID != uid {
		return nil, ErrPermissionDenied
	}
	if err := tx.Query().RedactArticle(ctx, article.ID); err != nil {
		slog.Error("Failed to redact article", "error", err)
		return nil, ErrAmbiguousInternal
	}
	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit database transaction", "error", err)
		return nil, ErrAmbiguousInternal
	}
	return cmsv1.RedactArticleResponse_builder{}.Build(), nil
}

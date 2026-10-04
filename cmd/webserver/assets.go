package main

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
	"github.com/sploders101/personal-website/cmd/webserver/ht"
	"github.com/sploders101/personal-website/cmd/webserver/storage"
)

func servePostAsset(
	cfg config.ServerConfig,
	db dbapi.Db,
	storageDriver storage.StorageDriver,
) http.Handler {
	serve404 := ht.Serve404(cfg)
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// Get asset details from db
		slug := req.PathValue("slug")
		articleId := req.PathValue("articleId")
		var articleUuid uuid.UUID
		if articleId != "" {
			var err error
			articleUuid, err = uuid.Parse(articleId)
			if err != nil {
				serve404.ServeHTTP(resp, req)
				return
			}
		}
		fileName := req.PathValue("file")
		dbAsset, err := db.Query().GetAsset(ctx, queries.GetAssetParams{
			Slug:      slug,
			ArticleID: articleUuid,
			FileName:  fileName,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				serve404.ServeHTTP(resp, req)
				return
			}
			slog.Error("Failed to get asset", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Etag", "SHA512:"+hex.EncodeToString(dbAsset.Sha512Hash))
		if dbAsset.ContentType != "" {
			resp.Header().Set("Content-Type", dbAsset.ContentType)
		}

		// Fetch asset from storage
		objectName := "SHA512:" + hex.EncodeToString(dbAsset.Sha512Hash)
		file, err := storageDriver.GetFile(ctx, objectName)
		if err != nil {
			slog.Error("Failed to get asset from storage", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.ServeContent(resp, req, fileName, dbAsset.CreatedAt, file)
	})
}

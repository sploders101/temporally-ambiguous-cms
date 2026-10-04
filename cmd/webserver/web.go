package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/csrf"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	"github.com/sploders101/personal-website/cmd/webserver/ht"
	"github.com/sploders101/personal-website/cmd/webserver/postfeeds"
	"github.com/sploders101/personal-website/cmd/webserver/storage"
	"github.com/sploders101/personal-website/cmd/webserver/userdata"
	"github.com/sploders101/personal-website/internal/env"
)

func makeWebRouter(
	ctx context.Context,
	cfg config.ServerConfig,
	db dbapi.Db,
	storageDriver storage.StorageDriver,
) http.Handler {
	// Applies common middlewares
	mw := func(handler http.Handler) http.Handler {
		return userdata.UserMiddleware(db, handler)
	}

	handler404 := ht.Serve404(cfg)

	postServer := mw(ht.ServePost(cfg, db, handler404))
	postAssetServer := servePostAsset(cfg, db, storageDriver)

	webRouter := http.NewServeMux()
	webRouter.Handle("GET /", ht.ServeAssets(cfg, db, mw(handler404)))
	webRouter.Handle("GET /{$}", mw(ht.ServeHome(cfg)))

	webRouter.Handle("GET /feeds/rss", postfeeds.ServeRSS(ctx, cfg, db))
	webRouter.Handle("GET /feeds/atom", postfeeds.ServeAtom(ctx, cfg, db))
	webRouter.Handle("GET /feeds/json", postfeeds.ServeJSON(ctx, cfg, db))

	webRouter.Handle("GET /posts/{$}", mw(ht.ServePostFeed(cfg, db)))
	webRouter.Handle("GET /posts/{slug}/{$}", postServer)
	webRouter.Handle("GET /posts/{slug}/{file...}", postAssetServer)

	webRouter.Handle("GET /permalinks/posts/{articleId}/{$}", postServer)
	webRouter.Handle("GET /permalinks/posts/{articleId}/{file...}", postAssetServer)

	webRouter.Handle("GET /login/", mw(ht.ServeLogin(cfg)))
	webRouter.Handle("POST /logout/", mw(serveLogout(db)))

	webRouter.Handle("GET /profile/", mw(ht.ServeProfile(cfg, db)))
	webRouter.Handle("GET /profile/edit/", mw(ht.ServeProfileEdit(cfg)))
	webRouter.Handle("POST /profile/edit/", mw(editUserProfile(db)))
	webRouter.Handle("GET /profile/add_ssh_key/", mw(ht.ServeAddSshKey(cfg)))
	webRouter.Handle("POST /profile/add_ssh_key/", mw(addSSHKey(db)))
	webRouter.Handle("POST /profile/remove_ssh_key/", mw(removeSSHKey(db)))

	webRouter.Handle("POST /auth/local/firstfactor", serveLocalLogin(cfg, db))
	if err := registerOidcHandlers(ctx, cfg, db, webRouter); err != nil {
		slog.Error("Error registering oidc handlers", "error", err)
		os.Exit(1)
	}
	webRouter.Handle("GET /decorations/{svgfile}", ht.DecorationServer{})

	// Enable hot reloading
	if env.Devmode {
		webRouter.Handle(
			"GET /_reload",
			http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				resp.Header().Set("Content-Type", "text/event-stream")
				resp.Header().Set("Cache-Control", "no-cache")
				resp.Header().Set("Connection", "keep-alive")
				flusher, ok := resp.(http.Flusher)
				if !ok {
					http.Error(resp, "Streaming unsupported", http.StatusInternalServerError)
					return
				}
				if _, err := resp.Write([]byte("data: online\n\n")); err != nil {
					return
				}
				flusher.Flush()
				<-req.Context().Done()
			}),
		)
	}

	if env.Devmode {
		wrapped := csrf.Protect([]byte(cfg.Secrets.CsrfSecret), csrf.Secure(false))(webRouter)
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			wrapped.ServeHTTP(resp, csrf.PlaintextHTTPRequest(req))
		})
	} else {
		return csrf.Protect([]byte(cfg.Secrets.CsrfSecret))(webRouter)
	}
}

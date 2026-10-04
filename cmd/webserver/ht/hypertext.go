package ht

import (
	"embed"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"

	"github.com/gorilla/csrf"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
	"github.com/sploders101/personal-website/cmd/webserver/helpers"
	"github.com/sploders101/personal-website/cmd/webserver/userdata"
	"github.com/sploders101/personal-website/internal/env"
)

//go:embed templates assets
var assets embed.FS

var templates *template.Template

func init() {
	templates = template.New("").Funcs(TemplateFuncs)
	template.Must(templates.ParseFS(assets, "templates/html/components/*.html"))
	template.Must(templates.ParseFS(assets, "templates/html/pages/*.html"))
}

type baseTemplateVars struct {
	Devmode     bool
	DailyTheme  string
	Config      config.ServerConfig
	Path        string
	User        queries.User
	UserSession queries.UsersSession
	CsrfField   template.HTML
}

type profileTemplateVars struct {
	baseTemplateVars

	SSHKeys []queries.UsersSshKey
}

// var themes = []string{
// 	"oceanblue-decor",
// 	"darkred-decor",
// 	"pink-decor",
// 	"seagreen-decor",
// 	"brown-decor",
// 	"electricblue-decor",
// }

func getBasePageConfig(
	cfg config.ServerConfig,
	req *http.Request,
) (baseTemplateVars, error) {
	ctx := req.Context()
	// now := time.Now()
	// daysSinceEpoch := now.Unix() / 86400
	// dailyTheme := themes[daysSinceEpoch%int64(len(themes))]
	return baseTemplateVars{
		Devmode:     env.Devmode,
		DailyTheme:  "",
		Config:      cfg,
		Path:        req.URL.Path,
		User:        userdata.GetUserData(ctx),
		UserSession: userdata.GetSessionInfo(ctx),
		CsrfField:   csrf.TemplateField(req),
	}, nil
}

func getProfileTemplateVars(
	cfg config.ServerConfig,
	req *http.Request,
	db dbapi.Db,
) (profileTemplateVars, error) {
	ctx := req.Context()
	baseVars, err := getBasePageConfig(cfg, req)
	if err != nil {
		return profileTemplateVars{}, err
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return profileTemplateVars{}, err
	}
	defer tx.Rollback()
	sshKeys, err := tx.Query().ListSSHKeysForUser(ctx, baseVars.User.ID)
	if err != nil {
		return profileTemplateVars{}, err
	}
	return profileTemplateVars{
		baseTemplateVars: baseVars,
		SSHKeys:          sshKeys,
	}, nil
}

func BaseTemplate(cfg config.ServerConfig, templateName string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		baseCfg, err := getBasePageConfig(cfg, req)
		if err != nil {
			slog.Error("Error generating base page config", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "text/html")
		if err := templates.ExecuteTemplate(resp, templateName, baseCfg); err != nil {
			slog.Error("Failed to render page", "template", templateName, "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

func PostTemplate(
	cfg config.ServerConfig,
	db dbapi.Db,
	templateName string,
	handler404 http.Handler,
) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		tx, err := db.Begin(req.Context())
		if err != nil {
			slog.Error("Failed to open database transaction", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
		baseCfg, err := getPostTemplateConfig(cfg, req, tx.Query())
		tx.Rollback()
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				handler404.ServeHTTP(resp, req)
				return
			}
			slog.Error("Error generating post template config", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "text/html")
		if err := templates.ExecuteTemplate(resp, templateName, baseCfg); err != nil {
			slog.Error("Failed to render page", "template", templateName, "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

func PostsFeedTemplate(cfg config.ServerConfig, db dbapi.Db, templateName string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		tx, err := db.Begin(req.Context())
		if err != nil {
			slog.Error("Failed to open database transaction", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
		baseCfg, err := getPostFeedTemplateConfig(cfg, req, tx.Query())
		tx.Rollback()
		if err != nil {
			slog.Error("Error generating post template config", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "text/html")
		if err := templates.ExecuteTemplate(resp, templateName, baseCfg); err != nil {
			slog.Error("Failed to render page", "template", templateName, "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

func ProfileTemplate(cfg config.ServerConfig, db dbapi.Db, templateName string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		baseCfg, err := getProfileTemplateVars(cfg, req, db)
		if err != nil {
			slog.Error("Error generating base page config", "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp.Header().Set("Content-Type", "text/html")
		if err := templates.ExecuteTemplate(resp, templateName, baseCfg); err != nil {
			slog.Error("Failed to render page", "template", templateName, "error", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
}

func ServeAssets(cfg config.ServerConfig, db dbapi.Db, handler404 http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		file, err := assets.Open(path.Join("assets", req.URL.Path))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// Serve 404
				handler404.ServeHTTP(resp, req)
				return
			}
			// Log err, serve 500
			slog.Error("Failed to load file", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		stat, err := file.Stat()
		if err != nil {
			slog.Error("Failed to stat file", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Serve file
		http.ServeContent(resp, req, path.Base(req.URL.Path), stat.ModTime(), file.(io.ReadSeeker))
	})
}

func ServeHome(cfg config.ServerConfig) http.Handler {
	return BaseTemplate(cfg, "home.html")
}

func ServePost(cfg config.ServerConfig, db dbapi.Db, handler404 http.Handler) http.Handler {
	return PostTemplate(cfg, db, "post.html", handler404)
}

func ServePostFeed(cfg config.ServerConfig, db dbapi.Db) http.Handler {
	return PostsFeedTemplate(cfg, db, "postsfeed.html")
}

func ServeLogin(cfg config.ServerConfig) http.Handler {
	return BaseTemplate(cfg, "login.html")
}

func ServeProfile(cfg config.ServerConfig, db dbapi.Db) http.Handler {
	return helpers.RequireLogin(ProfileTemplate(cfg, db, "profile.html"))
}

func ServeProfileEdit(cfg config.ServerConfig) http.Handler {
	return helpers.RequireLogin(BaseTemplate(cfg, "editprofile.html"))
}

func ServeAddSshKey(cfg config.ServerConfig) http.Handler {
	return helpers.RequireLogin(BaseTemplate(cfg, "add_ssh_key.html"))
}

func Serve404(cfg config.ServerConfig) http.Handler {
	serve404Html := BaseTemplate(cfg, "404.html")
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusNotFound)
		serve404Html.ServeHTTP(resp, req)
	})
}

package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	// "github.com/yuin/goldmark"

	"github.com/sploders101/personal-website/cmd/webserver/apiservices"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
	"github.com/sploders101/personal-website/internal/authutils"
	"github.com/sploders101/personal-website/internal/env"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/auth/v1/authv1connect"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/cms/v1/cmsv1connect"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	if env.Devmode {
		slog.Warn("Devmode is enabled!")
	}

	cfg, err := config.Load(configPath())
	if err != nil {
		slog.Error("Error loading configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("Loaded configuration")

	db, err := dbapi.NewDb(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("Error opening connection to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Seed first user if configured
	if cfg.Authentication.Local.Enabled && cfg.Authentication.Local.FirstUser != (config.FirstUser{}) {
		passwordHash, err := hashPassword(cfg.Authentication.Local.FirstUser.Password)
		if err != nil {
			slog.Error("Error hashing initial user's password", "error", err)
			os.Exit(1)
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			slog.Error("Failed to start database transaction", "error", err)
			os.Exit(1)
		}
		_, err = tx.Query().SeedUser(ctx, queries.SeedUserParams{
			Username:     cfg.Authentication.Local.FirstUser.Username,
			PasswordHash: sql.NullString{Valid: true, String: passwordHash},
		})
		if err != nil {
			tx.Rollback()
			if !errors.Is(err, sql.ErrNoRows) {
				slog.Error("Failed to seed first user", "error", err)
				os.Exit(1)
			}
		} else {
			if err := tx.Commit(); err != nil {
				slog.Info("Failed to commit database transaction", "error", err)
				os.Exit(1)
			}
			slog.Info("Seeded initial user", "username", cfg.Authentication.Local.FirstUser.Username)
		}
	}

	storageDriver, err := createStorageDriver(ctx, cfg)
	if err != nil {
		slog.Error("Error creating storage driver", "error", err)
		os.Exit(1)
	}

	address := "[::]:8080"

	router := http.NewServeMux()
	router.Handle("/", makeWebRouter(ctx, cfg, db, storageDriver))
	router.Handle(authv1connect.NewAuthServiceHandler(
		apiservices.NewAuthService(cfg, db),
	))
	cmsServicePath, cmsServiceHandler := cmsv1connect.NewCmsServiceHandler(
		apiservices.NewCmsService(cfg, db, storageDriver),
	)
	router.Handle(
		cmsServicePath,
		authutils.AuthenticateJwt(cfg.Secrets.JwtSecret, cmsServiceHandler),
	)

	// Start server
	listener, err := net.Listen("tcp", address)
	if err != nil {
		slog.Error("Error listening on specified address", "address", address, "error", err)
		os.Exit(1)
	}
	slog.Info("Listening for connections", "address", address)

	server := &http.Server{
		Handler:   router,
		Protocols: &http.Protocols{},
	}
	server.Protocols.SetHTTP1(true)
	server.Protocols.SetUnencryptedHTTP2(true)
	if err := server.Serve(listener); err != nil {
		slog.Error("Error serving connections", "address", address, "error", err)
		os.Exit(1)
	}
}

// configPath returns the path of the config file, overridable via CONFIG_FILE.
func configPath() string {
	if p := os.Getenv("CONFIG_FILE"); p != "" {
		return p
	}
	return "config.json"
}

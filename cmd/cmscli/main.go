package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"connectrpc.com/connect"
	"github.com/sploders101/personal-website/internal/authclient"
	"github.com/sploders101/personal-website/internal/configutil"
	"github.com/sploders101/personal-website/internal/env"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/cms/v1/cmsv1connect"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/content/v1/contentv1connect"
	"github.com/sploders101/personal-website/pkg/tabrowser"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller"
	"github.com/sploders101/personal-website/pkg/tabrowser/controller/modules/browsing"
	"github.com/urfave/cli/v3"
)

// TODO: Source these from other means
const ASSET_MAX_CHUNK_SIZE = 8192

// Wraps a CLI action handler to provide the config file.
// The config shouldn't be loaded before the CLI handler is called, but it is needed by
// almost every handler, so this makes it easy to wrap every function with a config provider.
func needsConfig(
	handler func(ctx context.Context, cfg Config, cmd *cli.Command) error,
) func(ctx context.Context, cmd *cli.Command) error {
	return func(ctx context.Context, cmd *cli.Command) error {
		config, err := configutil.FindConfig[Config](filepath.Join("tacms", "config.json"))
		if err != nil {
			return err
		}
		return handler(ctx, config, cmd)
	}
}

func main() {
	handler := log.New(os.Stderr)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	if env.Devmode {
		slog.Warn("Devmode is enabled!")
	}

	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "ping",
				Usage: "Authenticate against the CMS and ping the API.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Aliases: []string{"m"},
						Name:    "message",
						Value:   "Hello, world!",
						Usage:   "Changes the message the server should send back.",
					},
				},
				Action: needsConfig(
					func(ctx context.Context, cfg Config, cmd *cli.Command) error {
						return pingAPI(ctx, cfg, cmd.String("message"))
					},
				),
			},
			{
				Name:  "push",
				Usage: "Pushes an article to the CMS.",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "mdfile",
						Required:  true,
						UsageText: "<mdfile>",
					},
				},
				Action: needsConfig(
					func(ctx context.Context, cfg Config, cmd *cli.Command) error {
						return pushArticle(ctx, cfg, cmd.StringArg("mdfile"))
					},
				),
			},
			{
				Name:  "browse",
				Usage: "Browses articles on the CMS",
				Action: needsConfig(
					func(ctx context.Context, cfg Config, cmd *cli.Command) error {
						client := contentv1connect.NewContentServiceClient(
							env.ConnectHttp,
							cfg.Endpoint,
						)

						model := tabrowser.NewModel(ctx, []controller.Controller{
							browsing.NewArticleIndexController(client),
						})
						p := tea.NewProgram(model)
						if _, err := p.Run(); err != nil {
							return err
						}

						return nil
					},
				),
			},
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error(err.Error())
	}
}

func getClient(endpoint string) cmsv1connect.CmsServiceClient {
	authenticator := authclient.NewAuthenticator(endpoint)
	return cmsv1connect.NewCmsServiceClient(
		env.ConnectHttp,
		endpoint,
		connect.WithInterceptors(authenticator),
	)
}

package main

import (
	"context"
	"log/slog"

	cmsv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/cms/v1"
)

// Pings the API to see if we can properly authenticate
func pingAPI(ctx context.Context, cfg Config, message string) error {
	client := getClient(cfg.Endpoint)

	slog.Info("Starting ping", "endpoint", cfg.Endpoint)
	resp, err := client.Ping(ctx, cmsv1.PingRequest_builder{Message: message}.Build())
	if err != nil {
		return err
	}
	slog.Info(
		"Got ping response from server!",
		"message",
		resp.GetMessage(),
		"userID",
		resp.GetUserId(),
	)

	return nil
}

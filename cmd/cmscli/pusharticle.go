package main

import (
	"bufio"
	"context"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"path"
	"path/filepath"
	"slices"

	cmsv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/cms/v1"
	"github.com/sploders101/personal-website/internal/markdown"
	"google.golang.org/protobuf/proto"
)

// Pushes an article to the server
func pushArticle(ctx context.Context, cfg Config, mdfile string) error {
	client := getClient(cfg.Endpoint)
	baseDir := filepath.Dir(mdfile)

	slog.Info("Opening markdown", "file", mdfile)
	mdcontents, err := os.ReadFile(mdfile)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	frontmatter, _, err := markdown.Render(mdcontents, markdown.RenderOptions{})
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Validate metadata
	if frontmatter.Title == "" {
		return fmt.Errorf("missing title in frontmatter")
	}
	if frontmatter.Description == "" {
		return fmt.Errorf("missing description in frontmatter")
	}
	if frontmatter.Slug == "" {
		return fmt.Errorf("missing slug in frontmatter")
	}

	// Process assets
	var assetDescriptors []*cmsv1.AssetDescriptor
	var assetFiles []*os.File
	defer func() {
		for _, file := range assetFiles {
			if err := file.Close(); err != nil {
				slog.Error("Failed to close asset file", "error", err)
			}
		}
	}()
	for _, assetPath := range frontmatter.Assets {
		// Check asset validity
		if !filepath.IsLocal(assetPath) {
			return fmt.Errorf("asset %q must be local", assetPath)
		}

		// Open asset
		slog.Info("Processing asset", "asset", assetPath)
		file, err := os.Open(filepath.Join(baseDir, assetPath))
		if err != nil {
			return fmt.Errorf("failed to open asset %q: %w", assetPath, err)
		}
		stat, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat asset %q: %w", assetPath, err)
		}

		// Hash asset
		hasher := sha512.New()
		bytesRead, err := io.Copy(hasher, file)
		if err != nil {
			return fmt.Errorf("failed to read file %q: %w", assetPath, err)
		}
		if bytesRead != stat.Size() {
			return fmt.Errorf(
				"failed to hash file: expected %v bytes, got %v",
				stat.Size(),
				bytesRead,
			)
		}
		hash := hasher.Sum(nil)

		assetDescriptors = append(assetDescriptors, cmsv1.AssetDescriptor_builder{
			Filename:      assetPath,
			Sha512Hash:    hash,
			ContentType:   mime.TypeByExtension(path.Ext(assetPath)),
			ContentLength: stat.Size(),
		}.Build())
		assetFiles = append(assetFiles, file)
	}

	// Seed article
	slog.Info("Seeding article")
	seedResp, err := client.SeedArticle(ctx, cmsv1.SeedArticleRequest_builder{
		Markdown: string(mdcontents),
		Assets:   assetDescriptors,
	}.Build())
	if err != nil {
		return fmt.Errorf("failed to seed article: %w", err)
	}

	// Upload requested assets
	for _, uploadAsset := range seedResp.GetMissingAssets() {
		// Find corresponding file
		fileIndex := slices.IndexFunc(
			assetDescriptors,
			func(markedDescriptor *cmsv1.AssetDescriptor) bool {
				return proto.Equal(markedDescriptor, uploadAsset)
			},
		)
		if fileIndex == -1 {
			return fmt.Errorf(
				"security violation: server requested file %v, which was not marked for upload",
				uploadAsset.GetFilename(),
			)
		}
		file := assetFiles[fileIndex]
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek asset %q: %w", uploadAsset.GetFilename(), err)
		}

		// Initiate upload
		slog.Info("Uploading asset", "asset", uploadAsset.GetFilename())
		assetStream, err := client.PushAsset(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin asset upload: %w", err)
		}
		if err := assetStream.Send(cmsv1.PushAssetRequest_builder{
			Descriptor: uploadAsset,
		}.Build()); err != nil {
			return fmt.Errorf(
				"failed to send descriptor for asset %q: %w",
				uploadAsset.GetFilename(),
				err,
			)
		}

		// Upload with streaming chunks
		bufreader := bufio.NewReader(file)
		buffer := make([]byte, ASSET_MAX_CHUNK_SIZE)
		for {
			bytesRead, err := bufreader.Read(buffer)
			isEOF := errors.Is(err, io.EOF)
			if err != nil && !isEOF {
				return fmt.Errorf("failed to read %q: %w", uploadAsset.GetFilename(), err)
			}

			if bytesRead > 0 {
				slog.Info("Sending chunk")
				if err := assetStream.Send(cmsv1.PushAssetRequest_builder{
					Chunk: buffer[0:bytesRead],
				}.Build()); err != nil {
					return fmt.Errorf(
						"failed to send chunk from %q: %w",
						uploadAsset.GetFilename(),
						err,
					)
				}
			}

			if isEOF {
				break
			}
		}
		if _, err := assetStream.CloseAndReceive(); err != nil {
			return fmt.Errorf("failed to finalize asset %q: %w", uploadAsset.GetFilename(), err)
		}
	}

	// Publish
	slog.Info("Publishing article")
	if _, err := client.PublishArticle(ctx, cmsv1.PublishArticleRequest_builder{
		RevisionId: seedResp.GetRevisionId(),
	}.Build()); err != nil {
		return fmt.Errorf("failed to publish article: %w", err)
	}

	return nil
}

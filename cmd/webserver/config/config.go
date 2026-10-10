package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/sploders101/personal-website/internal/configutil"
)

type ServerConfig struct {
	BaseUrl        string               `mapstructure:"base_url"`
	ListenAddress  string               `mapstructure:"listen_address"`
	SiteSettings   SiteSettings         `mapstructure:"site_settings"`
	Authentication AuthenticationConfig `mapstructure:"authentication"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Storage        StorageBackend       `mapstructure:"storage"`
	Secrets        SecretConfig         `mapstructure:"secrets"`
}

type SiteSettings struct {
	ImagePath   string     `mapstructure:"image_path"`
	Title       string     `mapstructure:"title"`
	ShortTitle  string     `mapstructure:"short_title"`
	Description string     `mapstructure:"description"`
	Author      SiteAuthor `mapstructure:"author"`
}
type SiteAuthor struct {
	Name  string `mapstructure:"name"`
	Email string `mapstructure:"email"`
}

type DatabaseConfig struct {
	Dialect string `mapstructure:"dialect"`
	URL     string `mapstructure:"url"`
}

type StorageBackend struct {
	S3      *S3Config      `mapstructure:"s3"`
	LocalFS *LocalFSConfig `mapstructure:"local_fs"`
}

// S3Config configures an S3-compatible storage backend.
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	BucketName      string `mapstructure:"bucket_name"`
	BucketPort      int    `mapstructure:"bucket_port"`
	BucketRegion    string `mapstructure:"bucket_region"`
	BucketSubregion string `mapstructure:"bucket_subregion"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	UseSSL          bool   `mapstructure:"use_ssl"`
}

// LocalFSConfig configures a storage backend backed by a local filesystem path.
type LocalFSConfig struct {
	Path string `mapstructure:"path"`
}

type SecretConfig struct {
	CsrfSecret       string `mapstructure:"csrf_secret"`
	JwtSecret        string `mapstructure:"jwt_secret"`
	AuthPermitSecret string `mapstructure:"auth_permit_secret"`
}

// Load reads the JSON config file at path, resolves any secret references,
// and decodes the result into a ServerConfig. The path can be overridden at runtime
// with the `CONFIG_FILE` environment variable.
func Load(path string) (ServerConfig, error) {
	var cfg ServerConfig
	var err error
	if configPath := os.Getenv("CONFIG_FILE"); configPath != "" {
		cfg, err = configutil.Load[ServerConfig](configPath)
		if err != nil {
			return ServerConfig{}, err
		}
	} else {
		cfg, err = configutil.FindConfig[ServerConfig]("tacms/tacms_server.json")
		if err != nil {
			return ServerConfig{}, err
		}
	}

	if cfg.BaseUrl == "" {
		slog.Warn("Missing base_url from config. Some features may not work properly.")
	}
	if cfg.SiteSettings.ShortTitle == "" {
		cfg.SiteSettings.ShortTitle = cfg.SiteSettings.Title
	}
	if cfg.ListenAddress == "" {
		return ServerConfig{}, errors.New("missing `listen_address`")
	}
	if cfg.Storage.LocalFS != nil && cfg.Storage.S3 != nil {
		return ServerConfig{}, errors.New("multiple storage backends specified")
	}
	if cfg.Secrets.CsrfSecret == "" {
		return ServerConfig{}, errors.New("missing csrf secret")
	}
	if cfg.Secrets.JwtSecret == "" {
		return ServerConfig{}, errors.New("missing jwt secret")
	}

	return cfg, nil
}

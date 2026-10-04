package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

const (
	secretEnvSuffix  = "_fromenv"
	secretFileSuffix = "_fromfile"
)

type ServerConfig struct {
	BaseUrl        string               `mapstructure:"base_url"`
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
	CsrfSecret string `mapstructure:"csrf_secret"`
	JwtSecret  string `mapstructure:"jwt_secret"`
}

// Load reads the JSON config file at path, resolves any secret references,
// and decodes the result into a ServerConfig.
func Load(path string) (ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("read config file %s: %w", path, err)
	}
	var configRaw map[string]any
	if err := json.Unmarshal(data, &configRaw); err != nil {
		return ServerConfig{}, fmt.Errorf("parse config file %s: %w", path, err)
	}
	if err := resolveSecrets(configRaw); err != nil {
		return ServerConfig{}, err
	}
	var cfg ServerConfig
	if err := mapstructure.Decode(configRaw, &cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("decode config file %s: %w", path, err)
	}

	if cfg.BaseUrl == "" {
		slog.Warn("Missing base_url from config. Some features may not work properly.")
	}
	if cfg.SiteSettings.ShortTitle == "" {
		cfg.SiteSettings.ShortTitle = cfg.SiteSettings.Title
	}
	if cfg.Authentication.Local.Enabled {
		slog.Warn("Local authentication not yet implemented. Please use OIDC.")
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

// resolveSecrets walks the config tree, replacing keys ending in _fromenv or
// _fromfile with the value of the referenced environment variable or file, and
// rejects conflicting definitions of the same field.
func resolveSecrets(m map[string]any) error {
	for key := range m {
		if err := checkSecretConflict(m, key); err != nil {
			return err
		}
	}
	for key, val := range m {
		switch {
		case strings.HasSuffix(key, secretEnvSuffix):
			base := strings.TrimSuffix(key, secretEnvSuffix)
			name, ok := val.(string)
			if !ok || name == "" {
				return fmt.Errorf("%q must be a non-empty string (environment variable name)", key)
			}
			resolved, ok := os.LookupEnv(name)
			if !ok {
				return fmt.Errorf("environment variable %q for field %q is not set", name, base)
			}
			m[base] = resolved
			delete(m, key)
		case strings.HasSuffix(key, secretFileSuffix):
			base := strings.TrimSuffix(key, secretFileSuffix)
			path, ok := val.(string)
			if !ok || path == "" {
				return fmt.Errorf("%q must be a non-empty string (path to a secret file)", key)
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read secret file %q for field %q: %w", path, base, err)
			}
			m[base] = strings.TrimRight(string(contents), "\r\n")
			delete(m, key)
		default:
			if err := resolveNested(val); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveNested descends into child maps and slices so secret resolution
// applies at every level of the config tree.
func resolveNested(val any) error {
	switch v := val.(type) {
	case map[string]any:
		return resolveSecrets(v)
	case []any:
		for _, item := range v {
			if err := resolveNested(item); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkSecretConflict returns an error if key (a _fromenv or _fromfile field) is
// defined alongside a literal value or the other secret form of the same field.
func checkSecretConflict(m map[string]any, key string) error {
	var base string
	isEnv := strings.HasSuffix(key, secretEnvSuffix)
	switch {
	case isEnv:
		base = strings.TrimSuffix(key, secretEnvSuffix)
	case strings.HasSuffix(key, secretFileSuffix):
		base = strings.TrimSuffix(key, secretFileSuffix)
	default:
		return nil
	}
	if _, ok := m[base]; ok {
		return fmt.Errorf("field %q is set both literally and via %q; use only one", base, key)
	}
	other := base + secretEnvSuffix
	if isEnv {
		other = base + secretFileSuffix
	}
	if _, ok := m[other]; ok {
		return fmt.Errorf("field %q is set via both %q and %q; use only one", base, key, other)
	}
	return nil
}

// Package config loads Weather CLI configuration from defaults, files, dotenv, and environment variables.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Config contains user-facing Weather CLI settings.
type Config struct {
	LocationName  string  `json:"location_name"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Units         string  `json:"units"`
	MapProvider   string  `json:"map_provider"`
	RenderMode    string  `json:"render_mode"`
	IconStyle     string  `json:"icon_style"`
	RadarAutoplay bool    `json:"radar_autoplay"`
	Animation     bool    `json:"animation"`
}

// Default returns the built-in application configuration.
func Default() Config {
	return Config{
		LocationName: "Vienna",
		Latitude:     48.2082,
		Longitude:    16.3738,
		Units:        "metric",
		MapProvider:  "auto",
		RenderMode:   "auto",
		IconStyle:    "ascii",
		Animation:    true,
	}
}

// Load reads config from disk and applies environment overrides.
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return cfg, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}

	if v := os.Getenv("WEATHER_CLI_LOCATION"); v != "" {
		cfg.LocationName = v
	}
	if v := os.Getenv("WEATHER_CLI_UNITS"); v != "" {
		cfg.Units = v
	}
	if v := os.Getenv("WEATHER_CLI_MAP_PROVIDER"); v != "" {
		cfg.MapProvider = v
	}
	if v := os.Getenv("WEATHER_CLI_RENDER_MODE"); v != "" {
		cfg.RenderMode = v
	}
	if v := os.Getenv("WEATHER_CLI_ICON_STYLE"); v != "" {
		cfg.IconStyle = v
	}
	if v := os.Getenv("WEATHER_CLI_RADAR_AUTOPLAY"); v != "" {
		cfg.RadarAutoplay = truthy(v)
	}
	normalize(&cfg)
	return cfg, nil
}

// Save writes config to the user's config directory.
func Save(cfg Config) error {
	normalize(&cfg)
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// Path returns the user config file path.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "weather-cli", "config.json"), nil
}

func normalize(cfg *Config) {
	cfg.Units = strings.ToLower(strings.TrimSpace(cfg.Units))
	if cfg.Units != "imperial" {
		cfg.Units = "metric"
	}
	cfg.MapProvider = strings.ToLower(strings.TrimSpace(cfg.MapProvider))
	switch cfg.MapProvider {
	case "rainviewer", "openmeteo", "auto":
	default:
		cfg.MapProvider = "auto"
	}
	cfg.RenderMode = strings.ToLower(strings.TrimSpace(cfg.RenderMode))
	switch cfg.RenderMode {
	case "viu", "termimg", "blocks", "auto":
	default:
		cfg.RenderMode = "auto"
	}
	cfg.IconStyle = strings.ToLower(strings.TrimSpace(cfg.IconStyle))
	switch cfg.IconStyle {
	case "ascii", "unicode":
	default:
		cfg.IconStyle = "ascii"
	}
	if cfg.LocationName == "" {
		cfg.LocationName = "Vienna"
	}
	if !cfg.Animation {
		return
	}
	cfg.Animation = true
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

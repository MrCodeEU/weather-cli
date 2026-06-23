package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.LocationName == "" || cfg.Units != "metric" || cfg.MapProvider != "auto" || cfg.RenderMode != "auto" || cfg.IconStyle != "ascii" || cfg.RadarAutoplay || !cfg.Animation {
		t.Fatalf("bad default config: %+v", cfg)
	}
}

func TestNormalizeConfig(t *testing.T) {
	cfg := Config{Units: "weird", MapProvider: "bad", RenderMode: "bad", IconStyle: "bad", LocationName: ""}
	normalize(&cfg)
	if cfg.Units != "metric" || cfg.MapProvider != "auto" || cfg.RenderMode != "auto" || cfg.IconStyle != "ascii" || cfg.LocationName != "Vienna" {
		t.Fatalf("bad normalized config: %+v", cfg)
	}
}

func TestNormalizeConfigAcceptsViuRenderMode(t *testing.T) {
	cfg := Config{RenderMode: "viu"}
	normalize(&cfg)
	if cfg.RenderMode != "viu" {
		t.Fatalf("render mode got %q", cfg.RenderMode)
	}
}

func TestNormalizeConfigAcceptsBlocksRenderMode(t *testing.T) {
	cfg := Config{RenderMode: "blocks"}
	normalize(&cfg)
	if cfg.RenderMode != "blocks" {
		t.Fatalf("render mode got %q", cfg.RenderMode)
	}
}

func TestNormalizeConfigAcceptsTermimgRenderMode(t *testing.T) {
	cfg := Config{RenderMode: "termimg"}
	normalize(&cfg)
	if cfg.RenderMode != "termimg" {
		t.Fatalf("render mode got %q", cfg.RenderMode)
	}
}

func TestNormalizeConfigAcceptsInlineRenderMode(t *testing.T) {
	cfg := Config{RenderMode: "inline"}
	normalize(&cfg)
	if cfg.RenderMode != "inline" {
		t.Fatalf("render mode got %q", cfg.RenderMode)
	}
}

func TestNormalizeConfigAcceptsUnicodeIconStyle(t *testing.T) {
	cfg := Config{IconStyle: "unicode"}
	normalize(&cfg)
	if cfg.IconStyle != "unicode" {
		t.Fatalf("icon style got %q", cfg.IconStyle)
	}
}

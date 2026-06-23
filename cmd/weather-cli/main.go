// Command weather-cli starts the terminal weather application.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"weather-cli/internal/app"
	"weather-cli/internal/config"
	"weather-cli/internal/maps"
	"weather-cli/internal/weather"
)

func main() {
	if err := config.LoadDotenv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, ".env:", err)
		os.Exit(1)
	}

	var (
		location    = flag.String("location", "", "location name to load")
		lat         = flag.Float64("lat", 0, "latitude")
		lon         = flag.Float64("lon", 0, "longitude")
		noAnim      = flag.Bool("no-animation", false, "disable animated weather icons")
		units       = flag.String("units", "", "units: metric or imperial")
		mapProvider = flag.String("map-provider", "", "map provider: auto, rainviewer, openmeteo")
		renderMode  = flag.String("render-mode", "", "render mode: auto, viu, termimg, blocks")
		iconStyle   = flag.String("icon-style", "", "icon style: ascii or unicode")
		radarAuto   = flag.Bool("radar-autoplay", false, "autoplay radar frames every second")
		initialPage = flag.String("page", "", "initial page: current, hourly, daily, details, radar, settings, icons")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if *location != "" {
		cfg.LocationName = *location
	}
	if *lat != 0 || *lon != 0 {
		cfg.Latitude = *lat
		cfg.Longitude = *lon
	}
	if *units != "" {
		cfg.Units = *units
	}
	if *mapProvider != "" {
		cfg.MapProvider = *mapProvider
	}
	if *renderMode != "" {
		cfg.RenderMode = *renderMode
	}
	if *iconStyle != "" {
		cfg.IconStyle = *iconStyle
	}
	if *radarAuto {
		cfg.RadarAutoplay = true
	}
	if *noAnim {
		cfg.Animation = false
	}

	httpClient := weather.NewHTTPClient(12 * time.Second)
	weatherClient := weather.NewOpenMeteoClient(httpClient)
	mapClient := maps.NewService(httpClient, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := app.NewWithInitialPage(ctx, cfg, weatherClient, mapClient, *initialPage)
	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

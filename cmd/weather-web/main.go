// Command weather-web serves Weather CLI in a browser terminal.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"weather-cli/internal/config"
	"weather-cli/internal/webserver"
)

var version = "dev"

func main() {
	if err := config.LoadDotenv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, ".env:", err)
		os.Exit(1)
	}

	var (
		addr       = flag.String("addr", env("WEATHER_CLI_WEB_ADDR", ":8080"), "HTTP listen address")
		cliPath    = flag.String("cli", env("WEATHER_CLI_BIN", "weather-cli"), "weather-cli binary path")
		staticRoot = flag.String("static", env("WEATHER_CLI_STATIC_ROOT", "web/static"), "static asset directory")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	handler, err := webserver.New(webserver.Options{
		Addr:       *addr,
		CLIPath:    *cliPath,
		StaticRoot: *staticRoot,
		Version:    version,
	}, logger)
	if err != nil {
		logger.Error("init failed", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.Info("weather web listening", "addr", *addr, "version", version)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

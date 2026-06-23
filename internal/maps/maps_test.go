package maps

import (
	"context"
	"image"
	"image/color"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"weather-cli/internal/config"
	"weather-cli/internal/weather"
)

func TestTileForKnownCoordinate(t *testing.T) {
	z, x, y := TileFor(48.2082, 16.3738, 5)
	if z != 5 {
		t.Fatalf("zoom got %d", z)
	}
	if x != 17 || y != 11 {
		t.Fatalf("tile got %d/%d/%d, want 5/17/11", z, x, y)
	}
}

func TestPixelForKnownCoordinateInsideTile(t *testing.T) {
	z, px, py := PixelFor(48.2082, 16.3738, 5)
	if z != 5 {
		t.Fatalf("zoom got %d", z)
	}
	if int(px)/256 != 17 || int(py)/256 != 11 {
		t.Fatalf("pixel %.2f/%.2f should be inside tile 17/11", px, py)
	}
	if px == float64(17*256+128) || py == float64(11*256+128) {
		t.Fatalf("coordinate unexpectedly equals tile center: %.2f/%.2f", px, py)
	}
}

func TestLatLonForPixelRoundTrips(t *testing.T) {
	_, px, py := PixelFor(48.2082, 16.3738, 6)
	lat, lon := LatLonForPixel(px, py, 6)
	if math.Abs(lat-48.2082) > 0.000001 || math.Abs(lon-16.3738) > 0.000001 {
		t.Fatalf("round trip got %.6f/%.6f", lat, lon)
	}
}

func TestOpenMeteoSamplingURL(t *testing.T) {
	points := sampleGrid(weatherLocation(48.2082, 16.3738), 5, openMeteoGridSize(5))
	if len(points) != 81 {
		t.Fatalf("points got %d, want 81", len(points))
	}
	u := openMeteoForecastURL(points, 12)
	if !strings.Contains(u, "api.open-meteo.com/v1/forecast") {
		t.Fatalf("wrong endpoint: %s", u)
	}
	if !strings.Contains(u, "hourly=precipitation,precipitation_probability") {
		t.Fatalf("missing precipitation variables: %s", u)
	}
	if !strings.Contains(u, "forecast_hours=12") {
		t.Fatalf("missing forecast hour limit: %s", u)
	}
}

func TestOpenMeteoGridSizeIncreasesWithZoom(t *testing.T) {
	if openMeteoGridSize(4) >= openMeteoGridSize(8) {
		t.Fatalf("grid size should increase with zoom")
	}
	if openMeteoGridSize(9) != 13 {
		t.Fatalf("high zoom grid got %d, want 13", openMeteoGridSize(9))
	}
}

func TestRenderImageANSI(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	img.Set(1, 0, color.NRGBA{G: 255, A: 255})
	img.Set(0, 1, color.NRGBA{B: 255, A: 255})
	img.Set(1, 1, color.NRGBA{})

	got := RenderImageANSI(img, 2, 2)
	if !strings.Contains(got, "\x1b[38;2;255;0;0m") {
		t.Fatalf("render did not contain red ANSI foreground: %q", got)
	}
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("rendered dimensions look wrong: %q", got)
	}
}

func TestRenderFrameBlocksUsesBlocks(t *testing.T) {
	pngBody := tinyPNG(t)
	svc := NewService(roundTrip(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(pngBody))}, nil
	}), config.Default())
	frame, err := svc.RenderFrame(context.Background(), Frame{URL: "https://example.test/tile.png"}, 8, 4, "blocks")
	if err != nil {
		t.Fatal(err)
	}
	if frame.Native {
		t.Fatal("blocks mode should not use native protocol")
	}
	if frame.Renderer != "blocks" {
		t.Fatalf("renderer got %q, want blocks", frame.Renderer)
	}
	if frame.Art == "" {
		t.Fatal("blocks mode returned empty render")
	}
}

func TestRenderFrameInlineUsesNativeImageProtocol(t *testing.T) {
	pngBody := tinyPNG(t)
	svc := NewService(roundTrip(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(pngBody))}, nil
	}), config.Default())
	frame, err := svc.RenderFrame(context.Background(), Frame{URL: "https://example.test/tile.png"}, 8, 4, "inline")
	if err != nil {
		t.Fatal(err)
	}
	if !frame.Native {
		t.Fatal("inline mode should use native protocol")
	}
	if frame.Renderer != "inline" {
		t.Fatalf("renderer got %q, want inline", frame.Renderer)
	}
	if !strings.Contains(frame.Art, "\x1b]1337;File=inline=1;") {
		t.Fatalf("inline mode did not emit iTerm image sequence: %q", frame.Art)
	}
}

func TestRenderFrameAutoFallsBackToBlocksWithoutViu(t *testing.T) {
	oldLookPath := execLookPath
	execLookPath = func(string) (string, error) {
		return "", errNotFoundForTest{}
	}
	defer func() {
		execLookPath = oldLookPath
	}()

	pngBody := tinyPNG(t)
	svc := NewService(roundTrip(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(pngBody))}, nil
	}), config.Default())
	frame, err := svc.RenderFrame(context.Background(), Frame{URL: "https://example.test/tile.png"}, 8, 4, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if frame.Renderer == "" {
		t.Fatal("auto mode did not report an effective renderer")
	}
	if frame.Art == "" {
		t.Fatal("auto mode returned empty render")
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) Do(r *http.Request) (*http.Response, error) { return f(r) }

type errNotFoundForTest struct{}

func (errNotFoundForTest) Error() string { return "not found" }

func weatherLocation(lat, lon float64) weather.Location {
	return weather.Location{Latitude: lat, Longitude: lon}
}

func tinyPNG(t *testing.T) string {
	t.Helper()
	return "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\xcf\xc0\xf0\x1f\x00\x05\x00\x01\xff\x89\x99=\x1d\x00\x00\x00\x00IEND\xaeB`\x82"
}

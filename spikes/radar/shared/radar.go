package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math"
	"net/http"
	"time"
)

const (
	DefaultLat  = 48.2082
	DefaultLon  = 16.3738
	DefaultZoom = 5
)

type Frame struct {
	Time    time.Time
	Kind    string
	URL     string
	BaseURL string
}

func LoadFrames(ctx context.Context, lat, lon float64, zoom int) ([]Frame, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.rainviewer.com/public/weather-maps.json", nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("rainviewer failed: %s", res.Status)
	}
	var raw struct {
		Host  string `json:"host"`
		Radar struct {
			Past []struct {
				Time int64  `json:"time"`
				Path string `json:"path"`
			} `json:"past"`
			Nowcast []struct {
				Time int64  `json:"time"`
				Path string `json:"path"`
			} `json:"nowcast"`
		} `json:"radar"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}
	z, x, y := TileFor(lat, lon, zoom)
	baseURL := fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png", z, x, y)
	frames := make([]Frame, 0, len(raw.Radar.Past)+len(raw.Radar.Nowcast))
	add := func(kind string, xs []struct {
		Time int64  `json:"time"`
		Path string `json:"path"`
	}) {
		for _, item := range xs {
			frames = append(frames, Frame{
				Time:    time.Unix(item.Time, 0),
				Kind:    kind,
				URL:     fmt.Sprintf("%s%s/256/%d/%d/%d/2/1_1.png", raw.Host, item.Path, z, x, y),
				BaseURL: baseURL,
			})
		}
	}
	add("past", raw.Radar.Past)
	add("nowcast", raw.Radar.Nowcast)
	return frames, nil
}

func Composite(ctx context.Context, frame Frame) (image.Image, error) {
	base, err := FetchImage(ctx, frame.BaseURL)
	if err != nil {
		return nil, err
	}
	overlay, err := FetchImage(ctx, frame.URL)
	if err != nil {
		return nil, err
	}
	bounds := base.Bounds()
	canvas := image.NewNRGBA(bounds)
	draw.Draw(canvas, bounds, base, base.Bounds().Min, draw.Src)
	draw.Draw(canvas, bounds, overlay, overlay.Bounds().Min, draw.Over)
	DrawMarker(canvas)
	return canvas, nil
}

func FetchImage(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "weather-cli-radar-spike/0.1")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("tile failed: %s", res.Status)
	}
	img, _, err := image.Decode(res.Body)
	return img, err
}

func DrawMarker(img *image.NRGBA) {
	bounds := img.Bounds()
	cx := bounds.Min.X + bounds.Dx()/2
	cy := bounds.Min.Y + bounds.Dy()/2
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	red := color.NRGBA{R: 255, G: 32, B: 32, A: 255}
	for d := -8; d <= 8; d++ {
		setSafe(img, cx+d, cy, white)
		setSafe(img, cx, cy+d, white)
	}
	for d := -5; d <= 5; d++ {
		setSafe(img, cx+d, cy, red)
		setSafe(img, cx, cy+d, red)
	}
}

func TileFor(lat, lon float64, zoom int) (z, x, y int) {
	latRad := lat * math.Pi / 180
	n := math.Exp2(float64(zoom))
	x = int((lon + 180) / 360 * n)
	y = int((1 - math.Log(math.Tan(latRad)+1/math.Cos(latRad))/math.Pi) / 2 * n)
	return zoom, x, y
}

func setSafe(img *image.NRGBA, x, y int, c color.NRGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetNRGBA(x, y, c)
	}
}

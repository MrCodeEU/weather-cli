// Package maps provides weather map timeline providers and tile helpers.
package maps

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	termimg "github.com/blacktop/go-termimg"
	sixel "github.com/mattn/go-sixel"
	xdraw "golang.org/x/image/draw"

	"weather-cli/internal/config"
	"weather-cli/internal/weather"
)

// Service selects and queries configured map providers.
type Service struct {
	http HTTPDoer
	cfg  config.Config
}

// HTTPDoer is the subset of http.Client used by map providers.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Frame describes one map frame in a provider timeline.
type Frame struct {
	Time            time.Time
	Kind            string
	URL             string
	BaseURL         string
	URLTemplate     string
	BaseURLTemplate string
	Zoom            int
	CenterPixelX    float64
	CenterPixelY    float64
	Image           image.Image
}

// Timeline is the map/radar frame sequence displayed by the radar page.
type Timeline struct {
	Provider    string
	Attribution string
	Layer       string
	Frames      []Frame
	Message     string
}

// RenderedFrame is terminal output for one map frame plus renderer metadata.
type RenderedFrame struct {
	Art      string
	Native   bool
	Renderer string
}

var execLookPath = exec.LookPath

// NewService creates a map provider service.
func NewService(httpClient HTTPDoer, cfg config.Config) *Service {
	return &Service{http: httpClient, cfg: cfg}
}

// SetConfig updates provider settings after in-app config edits.
func (s *Service) SetConfig(cfg config.Config) {
	s.cfg = cfg
}

// RenderFrame fetches a map frame image and renders it with the requested terminal mode.
func (s *Service) RenderFrame(ctx context.Context, frame Frame, width, height int, mode string) (RenderedFrame, error) {
	if width < 12 {
		width = 12
	}
	if height < 4 {
		height = 4
	}
	if width > 220 {
		width = 220
	}
	if height > 80 {
		height = 80
	}
	img, err := s.compositeFrame(ctx, frame)
	if err != nil {
		return RenderedFrame{}, err
	}
	return renderImage(ctx, img, width, height, mode)
}

func renderImage(ctx context.Context, img image.Image, width, height int, mode string) (RenderedFrame, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "blocks":
		return renderBlocks(img, width, height), nil
	case "sixel", "xterm":
		return renderSixelImage(img, width, height)
	case "inline":
		return renderInlineImage(img, width, height, "inline")
	case "iterm":
		return renderInlineImage(img, width, height, "iterm")
	case "viu":
		if frame, err := renderViu(ctx, img, width, height); err == nil {
			return frame, nil
		}
		return renderBlocks(img, width, height), nil
	case "termimg":
		if frame, err := renderTermimg(img, width, height); err == nil {
			return frame, nil
		}
		return renderBlocks(img, width, height), nil
	default:
		if frame, err := renderViu(ctx, img, width, height); err == nil {
			return frame, nil
		}
		if frame, err := renderTermimg(img, width, height); err == nil {
			return frame, nil
		}
		return renderBlocks(img, width, height), nil
	}
}

func renderBlocks(img image.Image, width, height int) RenderedFrame {
	return RenderedFrame{Art: RenderImageANSI(img, width, height), Renderer: "blocks"}
}

func renderSixelImage(img image.Image, width, height int) (RenderedFrame, error) {
	const cellWidthPx = 8
	const cellHeightPx = 16

	scaled := image.NewNRGBA(image.Rect(0, 0, max(1, width*cellWidthPx), max(1, height*cellHeightPx)))
	xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	encoder := sixel.NewEncoder(&out)
	encoder.Colors = 192
	if err := encoder.Encode(scaled); err != nil {
		return RenderedFrame{}, err
	}
	out.WriteString(strings.Repeat("\r\n", height))
	return RenderedFrame{Art: out.String(), Native: true, Renderer: "inline"}, nil
}

func renderInlineImage(img image.Image, width, height int, renderer string) (RenderedFrame, error) {
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		return RenderedFrame{}, err
	}
	encoded := base64.StdEncoding.EncodeToString(pngData.Bytes())
	sequence := fmt.Sprintf("\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=1:%s\a%s", pngData.Len(), width, height, encoded, strings.Repeat("\r\n", height))
	return RenderedFrame{Art: sequence, Native: true, Renderer: renderer}, nil
}

func renderViu(ctx context.Context, img image.Image, width, height int) (RenderedFrame, error) {
	bin, err := execLookPath("viu")
	if err != nil {
		return RenderedFrame{}, err
	}
	tmp, err := os.CreateTemp("", "weather-cli-radar-*.png")
	if err != nil {
		return RenderedFrame{}, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if err := png.Encode(tmp, img); err != nil {
		_ = tmp.Close()
		return RenderedFrame{}, err
	}
	if err := tmp.Close(); err != nil {
		return RenderedFrame{}, err
	}
	cmd := exec.CommandContext(ctx, bin, "-w", fmt.Sprint(width), "-h", fmt.Sprint(height), tmpName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return RenderedFrame{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
	return RenderedFrame{Art: out.String(), Native: true, Renderer: "viu"}, nil
}

func renderTermimg(img image.Image, width, height int) (RenderedFrame, error) {
	out, err := termimg.New(img).Size(width, height).Scale(termimg.ScaleStretch).Protocol(termimg.Auto).Render()
	if err != nil {
		return RenderedFrame{}, err
	}
	return RenderedFrame{Art: out, Native: true, Renderer: "termimg"}, nil
}

func (s *Service) compositeFrame(ctx context.Context, frame Frame) (image.Image, error) {
	if frame.Image != nil {
		return frame.Image, nil
	}
	if frame.URLTemplate != "" {
		return s.compositeCenteredFrame(ctx, frame)
	}
	weatherImg, err := s.fetchImage(ctx, frame.URL)
	if err != nil {
		return nil, err
	}
	bounds := weatherImg.Bounds()
	canvas := image.NewNRGBA(bounds)
	if frame.BaseURL != "" {
		baseImg, err := s.fetchImage(ctx, frame.BaseURL)
		if err == nil {
			draw.Draw(canvas, bounds, baseImg, baseImg.Bounds().Min, draw.Src)
		}
	}
	draw.Draw(canvas, bounds, weatherImg, bounds.Min, draw.Over)
	drawMarker(canvas)
	return canvas, nil
}

func (s *Service) compositeCenteredFrame(ctx context.Context, frame Frame) (image.Image, error) {
	const tileSize = 256
	centerTileX := int(math.Floor(frame.CenterPixelX / tileSize))
	centerTileY := int(math.Floor(frame.CenterPixelY / tileSize))
	centerInMosaicX := int(math.Round(frame.CenterPixelX - float64((centerTileX-1)*tileSize)))
	centerInMosaicY := int(math.Round(frame.CenterPixelY - float64((centerTileY-1)*tileSize)))
	mosaic := image.NewNRGBA(image.Rect(0, 0, tileSize*3, tileSize*3))
	maxTile := int(math.Exp2(float64(frame.Zoom)))
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x := wrapTileX(centerTileX+dx, maxTile)
			y := centerTileY + dy
			if y < 0 || y >= maxTile {
				continue
			}
			dst := image.Rect((dx+1)*tileSize, (dy+1)*tileSize, (dx+2)*tileSize, (dy+2)*tileSize)
			if frame.BaseURLTemplate != "" {
				if baseImg, err := s.fetchImage(ctx, tileURL(frame.BaseURLTemplate, frame.Zoom, x, y)); err == nil {
					draw.Draw(mosaic, dst, baseImg, baseImg.Bounds().Min, draw.Src)
				}
			}
			weatherImg, err := s.fetchImage(ctx, tileURL(frame.URLTemplate, frame.Zoom, x, y))
			if err != nil {
				return nil, err
			}
			draw.Draw(mosaic, dst, weatherImg, weatherImg.Bounds().Min, draw.Over)
		}
	}
	crop := image.NewNRGBA(image.Rect(0, 0, tileSize, tileSize))
	src := image.Rect(centerInMosaicX-tileSize/2, centerInMosaicY-tileSize/2, centerInMosaicX+tileSize/2, centerInMosaicY+tileSize/2)
	draw.Draw(crop, crop.Bounds(), mosaic, src.Min, draw.Src)
	drawMarker(crop)
	return crop, nil
}

func (s *Service) fetchImage(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "weather-cli/0.1")
	res, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("map tile failed: %s", res.Status)
	}
	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// Timeline loads map frames for the configured provider.
func (s *Service) Timeline(ctx context.Context, loc weather.Location, _ string, zoom int) (Timeline, error) {
	switch s.cfg.MapProvider {
	case "rainviewer":
		return s.rainViewerTimeline(ctx, loc, zoom)
	case "openmeteo":
		return s.openMeteoTimeline(ctx, loc, zoom)
	default:
		return s.rainViewerTimeline(ctx, loc, zoom)
	}
}

func (s *Service) rainViewerTimeline(ctx context.Context, loc weather.Location, zoom int) (Timeline, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.rainviewer.com/public/weather-maps.json", nil)
	if err != nil {
		return Timeline{}, err
	}
	res, err := s.http.Do(req)
	if err != nil {
		return Timeline{}, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return Timeline{}, fmt.Errorf("rainviewer failed: %s", res.Status)
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
		return Timeline{}, err
	}
	frames := make([]Frame, 0, len(raw.Radar.Past)+len(raw.Radar.Nowcast))
	z, tileX, tileY := TileFor(loc.Latitude, loc.Longitude, zoom)
	_, centerPixelX, centerPixelY := PixelFor(loc.Latitude, loc.Longitude, zoom)
	baseURL := baseMapURL(z, tileX, tileY)
	baseTemplate := baseMapTemplate()
	add := func(kind string, xs []struct {
		Time int64  `json:"time"`
		Path string `json:"path"`
	}) {
		for _, frame := range xs {
			template := fmt.Sprintf("%s%s/256/{z}/{x}/{y}/2/1_1.png", raw.Host, frame.Path)
			u := tileURL(template, z, tileX, tileY)
			frames = append(frames, Frame{
				Time:            time.Unix(frame.Time, 0),
				Kind:            kind,
				URL:             u,
				BaseURL:         baseURL,
				URLTemplate:     template,
				BaseURLTemplate: baseTemplate,
				Zoom:            z,
				CenterPixelX:    centerPixelX,
				CenterPixelY:    centerPixelY,
			})
		}
	}
	add("past", raw.Radar.Past)
	add("nowcast", raw.Radar.Nowcast)
	return Timeline{Provider: "RainViewer", Attribution: "Weather radar by RainViewer · Map data © OpenStreetMap contributors", Layer: "radar", Frames: frames}, nil
}

func (s *Service) openMeteoTimeline(ctx context.Context, loc weather.Location, zoom int) (Timeline, error) {
	const tileSize = 256
	z, _, _ := TileFor(loc.Latitude, loc.Longitude, zoom)
	_, centerPixelX, centerPixelY := PixelFor(loc.Latitude, loc.Longitude, zoom)
	points := sampleGrid(loc, zoom, openMeteoGridSize(zoom))
	q := openMeteoForecastURL(points, 12)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, q, nil)
	if err != nil {
		return Timeline{}, err
	}
	req.Header.Set("User-Agent", "weather-cli/0.1")
	res, err := s.http.Do(req)
	if err != nil {
		return Timeline{}, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return Timeline{}, fmt.Errorf("open-meteo failed: %s", res.Status)
	}
	var raw []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Hourly    struct {
			Time          []string  `json:"time"`
			Precipitation []float64 `json:"precipitation"`
			Probability   []int     `json:"precipitation_probability"`
		} `json:"hourly"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return Timeline{}, err
	}
	if len(raw) == 0 || len(raw[0].Hourly.Time) == 0 {
		return Timeline{Provider: "Open-Meteo", Message: "Open-Meteo returned no precipitation forecast frames."}, nil
	}
	base, err := s.baseMosaic(ctx, zoom, centerPixelX, centerPixelY)
	if err != nil {
		base = image.NewNRGBA(image.Rect(0, 0, tileSize, tileSize))
	}
	frameCount := len(raw[0].Hourly.Time)
	frames := make([]Frame, 0, frameCount)
	for i := 0; i < frameCount; i++ {
		img := image.NewNRGBA(image.Rect(0, 0, tileSize, tileSize))
		draw.Draw(img, img.Bounds(), base, base.Bounds().Min, draw.Src)
		for _, point := range raw {
			if i >= len(point.Hourly.Precipitation) || i >= len(point.Hourly.Probability) {
				continue
			}
			_, px, py := PixelFor(point.Latitude, point.Longitude, zoom)
			x := int(math.Round(float64(tileSize/2) + px - centerPixelX))
			y := int(math.Round(float64(tileSize/2) + py - centerPixelY))
			drawPrecipBlob(img, x, y, point.Hourly.Precipitation[i], point.Hourly.Probability[i], zoom)
		}
		drawMarker(img)
		t, _ := time.Parse("2006-01-02T15:04", raw[0].Hourly.Time[i])
		frames = append(frames, Frame{
			Time:  t,
			Kind:  "forecast",
			URL:   q,
			Image: img,
			Zoom:  z,
		})
	}
	return Timeline{
		Provider:    "Open-Meteo",
		Attribution: "Forecast data by Open-Meteo · Map data © OpenStreetMap contributors",
		Layer:       "synthetic precipitation forecast",
		Frames:      frames,
	}, nil
}

func (s *Service) baseMosaic(ctx context.Context, zoom int, centerPixelX, centerPixelY float64) (image.Image, error) {
	const tileSize = 256
	centerTileX := int(math.Floor(centerPixelX / tileSize))
	centerTileY := int(math.Floor(centerPixelY / tileSize))
	centerInMosaicX := int(math.Round(centerPixelX - float64((centerTileX-1)*tileSize)))
	centerInMosaicY := int(math.Round(centerPixelY - float64((centerTileY-1)*tileSize)))
	mosaic := image.NewNRGBA(image.Rect(0, 0, tileSize*3, tileSize*3))
	maxTile := int(math.Exp2(float64(zoom)))
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x := wrapTileX(centerTileX+dx, maxTile)
			y := centerTileY + dy
			if y < 0 || y >= maxTile {
				continue
			}
			baseImg, err := s.fetchImage(ctx, baseMapURL(zoom, x, y))
			if err != nil {
				return nil, err
			}
			dst := image.Rect((dx+1)*tileSize, (dy+1)*tileSize, (dx+2)*tileSize, (dy+2)*tileSize)
			draw.Draw(mosaic, dst, baseImg, baseImg.Bounds().Min, draw.Src)
		}
	}
	crop := image.NewNRGBA(image.Rect(0, 0, tileSize, tileSize))
	src := image.Rect(centerInMosaicX-tileSize/2, centerInMosaicY-tileSize/2, centerInMosaicX+tileSize/2, centerInMosaicY+tileSize/2)
	draw.Draw(crop, crop.Bounds(), mosaic, src.Min, draw.Src)
	return crop, nil
}

type samplePoint struct {
	Lat float64
	Lon float64
}

func sampleGrid(loc weather.Location, zoom, size int) []samplePoint {
	const tileSize = 256
	_, centerX, centerY := PixelFor(loc.Latitude, loc.Longitude, zoom)
	step := float64(tileSize) / float64(size-1)
	start := -float64(tileSize) / 2
	points := make([]samplePoint, 0, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			lat, lon := LatLonForPixel(centerX+start+float64(x)*step, centerY+start+float64(y)*step, zoom)
			points = append(points, samplePoint{Lat: lat, Lon: lon})
		}
	}
	return points
}

func openMeteoGridSize(zoom int) int {
	switch {
	case zoom >= 9:
		return 13
	case zoom >= 7:
		return 11
	case zoom >= 5:
		return 9
	default:
		return 7
	}
}

func openMeteoForecastURL(points []samplePoint, hours int) string {
	lats := make([]string, len(points))
	lons := make([]string, len(points))
	for i, p := range points {
		lats[i] = fmt.Sprintf("%.5f", p.Lat)
		lons[i] = fmt.Sprintf("%.5f", p.Lon)
	}
	return fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s&hourly=precipitation,precipitation_probability&forecast_hours=%d&timezone=UTC", strings.Join(lats, ","), strings.Join(lons, ","), hours)
}

func drawPrecipBlob(img *image.NRGBA, cx, cy int, mm float64, probability, zoom int) {
	if mm <= 0 && probability < 20 {
		return
	}
	alpha := uint8(clampInt(45+probability*2, 45, 210))
	baseRadius := 42 / openMeteoGridSize(zoom)
	radius := clampInt(baseRadius+int(mm*5), 5, 18)
	c := precipColor(mm, alpha)
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y > radius*radius {
				continue
			}
			setSafe(img, cx+x, cy+y, c)
		}
	}
}

func precipColor(mm float64, alpha uint8) color.NRGBA {
	switch {
	case mm >= 5:
		return color.NRGBA{R: 160, G: 40, B: 220, A: alpha}
	case mm >= 2:
		return color.NRGBA{R: 40, G: 90, B: 230, A: alpha}
	case mm >= 0.5:
		return color.NRGBA{R: 40, G: 170, B: 255, A: alpha}
	default:
		return color.NRGBA{R: 110, G: 210, B: 255, A: alpha}
	}
}

// TileFor converts latitude and longitude into a slippy-map tile coordinate.
func TileFor(lat, lon float64, zoom int) (z, x, y int) {
	z, pixelX, pixelY := PixelFor(lat, lon, zoom)
	return z, int(math.Floor(pixelX / 256)), int(math.Floor(pixelY / 256))
}

// PixelFor converts latitude and longitude into global slippy-map pixel coordinates.
func PixelFor(lat, lon float64, zoom int) (z int, x, y float64) {
	latRad := lat * math.Pi / 180
	n := math.Exp2(float64(zoom)) * 256
	x = (lon + 180) / 360 * n
	y = (1 - math.Log(math.Tan(latRad)+1/math.Cos(latRad))/math.Pi) / 2 * n
	return zoom, x, y
}

// LatLonForPixel converts global slippy-map pixel coordinates into latitude and longitude.
func LatLonForPixel(x, y float64, zoom int) (lat, lon float64) {
	n := math.Exp2(float64(zoom)) * 256
	lon = x/n*360 - 180
	latRad := math.Atan(math.Sinh(math.Pi * (1 - 2*y/n)))
	lat = latRad * 180 / math.Pi
	return lat, lon
}

func baseMapURL(z, x, y int) string {
	return tileURL(baseMapTemplate(), z, x, y)
}

func baseMapTemplate() string {
	return "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
}

func tileURL(template string, z, x, y int) string {
	u := strings.ReplaceAll(template, "{z}", fmt.Sprint(z))
	u = strings.ReplaceAll(u, "{x}", fmt.Sprint(x))
	u = strings.ReplaceAll(u, "{y}", fmt.Sprint(y))
	return u
}

func wrapTileX(x, maxTile int) int {
	if maxTile <= 0 {
		return x
	}
	x %= maxTile
	if x < 0 {
		x += maxTile
	}
	return x
}

func clampInt(v, lower, upper int) int {
	if v < lower {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}

func drawMarker(img *image.NRGBA) {
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
	for y := -2; y <= 2; y++ {
		for x := -2; x <= 2; x++ {
			setSafe(img, cx+x, cy+y, red)
		}
	}
}

func setSafe(img *image.NRGBA, x, y int, c color.NRGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetNRGBA(x, y, c)
	}
}

// ClearNativeImages removes terminal image layers from renderers that keep image
// storage outside the text buffer.
func ClearNativeImages() string {
	return "\x1b[!p\x1b_Ga=d,d=A\x1b\\"
}

// RenderImageANSI converts an image into terminal-safe ANSI true-color blocks.
func RenderImageANSI(img image.Image, width, height int) string {
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return ""
	}
	var b strings.Builder
	for row := 0; row < height; row++ {
		upperY := bounds.Min.Y + (row*2)*bounds.Dy()/(height*2)
		lowerY := bounds.Min.Y + (row*2+1)*bounds.Dy()/(height*2)
		for col := 0; col < width; col++ {
			x := bounds.Min.X + col*bounds.Dx()/width
			ur, ug, ub, ua := img.At(x, upperY).RGBA()
			lr, lg, lb, la := img.At(x, lowerY).RGBA()
			upperVisible := ua >= 0x2000
			lowerVisible := la >= 0x2000
			switch {
			case upperVisible && lowerVisible:
				_, _ = fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", uint8(ur>>8), uint8(ug>>8), uint8(ub>>8), uint8(lr>>8), uint8(lg>>8), uint8(lb>>8))
			case upperVisible:
				_, _ = fmt.Fprintf(&b, "\x1b[0m\x1b[38;2;%d;%d;%dm▀", uint8(ur>>8), uint8(ug>>8), uint8(ub>>8))
			case lowerVisible:
				_, _ = fmt.Fprintf(&b, "\x1b[0m\x1b[38;2;%d;%d;%dm▄", uint8(lr>>8), uint8(lg>>8), uint8(lb>>8))
			default:
				b.WriteString("\x1b[0m")
				b.WriteString(" ")
			}
		}
		b.WriteString("\x1b[0m")
		if row != height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

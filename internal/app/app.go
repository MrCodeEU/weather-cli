// Package app contains the Bubble Tea application model and page rendering.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"weather-cli/internal/config"
	"weather-cli/internal/icons"
	"weather-cli/internal/maps"
	"weather-cli/internal/ui"
	"weather-cli/internal/weather"
)

// WeatherClient loads locations and forecasts for the application.
type WeatherClient interface {
	Geocode(context.Context, string) (weather.Location, error)
	Forecast(context.Context, weather.Location, string) (weather.Report, error)
}

// MapClient loads map timeline data for the selected weather location.
type MapClient interface {
	Timeline(context.Context, weather.Location, string, int) (maps.Timeline, error)
	RenderFrame(context.Context, maps.Frame, int, int, string) (maps.RenderedFrame, error)
	SetConfig(config.Config)
}

type page int

const (
	pageCurrent page = iota
	pageHourly
	pageDaily
	pageDetails
	pageRadar
	pageSettings
	pageIcons
)

var pageNames = []string{"Current", "Hourly", "Daily", "Details", "Radar", "Settings"}

// Model is the Bubble Tea state machine for the terminal application.
type Model struct {
	ctx     context.Context
	cfg     config.Config
	weather WeatherClient
	maps    MapClient

	width  int
	height int
	page   page
	tick   int
	cursor int
	debug  bool
	zoom   int
	report weather.Report
	tl     maps.Timeline
	radar  string
	native bool
	clear  bool
	render string
	err    error
	status string
	ready  bool

	editing   bool
	editField int
	input     textinput.Model
}

type reportMsg struct {
	report weather.Report
	err    error
}

type radarMsg struct {
	timeline maps.Timeline
	err      error
}

type radarImageMsg struct {
	frame maps.RenderedFrame
	err   error
}

type tickMsg time.Time

type radarTickMsg time.Time

// New creates the initial application model.
func New(ctx context.Context, cfg config.Config, weatherClient WeatherClient, mapClient MapClient) Model {
	return NewWithInitialPage(ctx, cfg, weatherClient, mapClient, "")
}

// NewWithInitialPage creates the initial application model and opens a named page when provided.
func NewWithInitialPage(ctx context.Context, cfg config.Config, weatherClient WeatherClient, mapClient MapClient, initialPage string) Model {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.Width = 36
	initial := pageFromName(initialPage)
	m := Model{ctx: ctx, cfg: cfg, weather: weatherClient, maps: mapClient, page: initial, debug: initial == pageIcons, zoom: 5, input: ti, status: "Loading weather..."}
	return m
}

// Init starts the initial weather load and animation timer.
func (m Model) Init() tea.Cmd {
	if m.cfg.RadarAutoplay {
		return tea.Batch(m.loadWeather(), tick(), radarTick())
	}
	return tea.Batch(m.loadWeather(), tick())
}

// Update handles terminal input, async data messages, and animation ticks.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.page == pageRadar && len(m.tl.Frames) > 0 {
			hadNative := m.clearRadarImage()
			if hadNative {
				return m, tea.Batch(tea.ClearScreen, m.loadRadarImage())
			}
			return m, m.loadRadarImage()
		}
	case tea.KeyMsg:
		if m.editing {
			return m.updateEditing(msg)
		}
		if m.debug {
			return m.updateDebug(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "right", "l", "tab":
			if m.debug {
				return m, nil
			}
			m.page = (m.page + 1) % page(len(pageNames))
			m.cursor = 0
			clearCmd := m.clearRadarIfLeaving()
			return m, tea.Batch(clearCmd, m.maybeLoadRadar())
		case "left", "h", "shift+tab":
			if m.debug {
				return m, nil
			}
			m.page = (m.page + page(len(pageNames)) - 1) % page(len(pageNames))
			m.cursor = 0
			return m, m.clearRadarIfLeaving()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			if m.page == pageRadar && len(m.tl.Frames) > 0 {
				hadNative := m.clearRadarImage()
				if hadNative {
					return m, tea.Batch(tea.ClearScreen, m.loadRadarImage())
				}
				return m, m.loadRadarImage()
			}
		case "down", "j":
			m.cursor++
			if m.page == pageRadar && len(m.tl.Frames) > 0 {
				hadNative := m.clearRadarImage()
				if hadNative {
					return m, tea.Batch(tea.ClearScreen, m.loadRadarImage())
				}
				return m, m.loadRadarImage()
			}
		case "r":
			m.status = "Refreshing..."
			hadNative := m.clearRadarImage()
			cmd := tea.Batch(m.loadWeather(), m.loadRadar())
			if hadNative {
				return m, tea.Batch(tea.ClearScreen, cmd)
			}
			return m, cmd
		case "a":
			m.cfg.Animation = !m.cfg.Animation
			if err := m.persistConfig(); err != nil {
				m.status = "Save failed: " + err.Error()
			}
			if m.cfg.Animation {
				return m, tick()
			}
		case "p":
			if m.page == pageRadar {
				m.cfg.RadarAutoplay = !m.cfg.RadarAutoplay
				if err := m.persistConfig(); err != nil {
					m.status = "Save failed: " + err.Error()
				}
				if m.cfg.RadarAutoplay {
					return m, radarTick()
				}
			}
		case "1":
			m.page = pageCurrent
			return m, m.clearRadarIfLeaving()
		case "2":
			m.page = pageHourly
			return m, m.clearRadarIfLeaving()
		case "3":
			m.page = pageDaily
			return m, m.clearRadarIfLeaving()
		case "4":
			m.page = pageDetails
			return m, m.clearRadarIfLeaving()
		case "5":
			m.page = pageRadar
			return m, m.maybeLoadRadar()
		case "6":
			m.page = pageSettings
			return m, m.clearRadarIfLeaving()
		case "+", "=":
			if m.page == pageRadar && m.zoom < 10 {
				m.zoom++
				hadNative := m.clearRadarImage()
				if hadNative {
					return m, tea.Batch(tea.ClearScreen, m.loadRadar())
				}
				return m, m.loadRadar()
			}
		case "-", "_":
			if m.page == pageRadar && m.zoom > 3 {
				m.zoom--
				hadNative := m.clearRadarImage()
				if hadNative {
					return m, tea.Batch(tea.ClearScreen, m.loadRadar())
				}
				return m, m.loadRadar()
			}
		case "m":
			if m.page == pageRadar {
				m.cfg.RenderMode = nextRenderMode(m.cfg.RenderMode)
				if err := m.persistConfig(); err != nil {
					m.status = "Save failed: " + err.Error()
				}
				hadNative := m.clearRadarImage()
				if hadNative {
					return m, tea.Batch(tea.ClearScreen, m.loadRadarImage())
				}
				return m, m.loadRadarImage()
			}
		case "enter":
			if m.page == pageSettings {
				return m.startEditing()
			}
		case "s":
			if m.page == pageSettings {
				return m.saveConfig()
			}
		}
	case reportMsg:
		m.ready = true
		m.err = msg.err
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.report = msg.report
			m.status = "Updated " + msg.report.Fetched.Format("15:04:05")
			if m.page == pageRadar {
				return m, m.maybeLoadRadar()
			}
		}
	case radarMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.tl = msg.timeline
			m.clearRadarImage()
			m.status = "Radar updated"
			return m, m.loadRadarImage()
		}
	case radarImageMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			if msg.frame.Native {
				m.radar = maps.ClearNativeImages() + msg.frame.Art
			} else if m.native {
				m.radar = maps.ClearNativeImages() + msg.frame.Art
			} else {
				m.radar = msg.frame.Art
			}
			m.native = msg.frame.Native
			m.render = msg.frame.Renderer
			m.status = "Radar image rendered"
			if m.page == pageRadar && m.cfg.RadarAutoplay && len(m.tl.Frames) > 1 {
				return m, radarTick()
			}
		}
	case tickMsg:
		m.tick++
		if m.cfg.Animation {
			return m, tick()
		}
	case radarTickMsg:
		if m.page == pageRadar && m.cfg.RadarAutoplay && len(m.tl.Frames) > 1 {
			m.cursor = (m.cursor + 1) % len(m.tl.Frames)
			return m, m.loadRadarImage()
		}
		if m.cfg.RadarAutoplay {
			return m, radarTick()
		}
	}
	return m, nil
}

func (m *Model) updateDebug(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		m.cursor++
	case "a":
		m.cfg.Animation = !m.cfg.Animation
		if m.cfg.Animation {
			return m, tick()
		}
	}
	return m, nil
}

func (m *Model) clearRadarImage() bool {
	hadNative := m.native
	if hadNative {
		m.clear = true
		m.native = false
	}
	m.radar = ""
	m.render = ""
	return hadNative
}

func (m *Model) clearRadarIfLeaving() tea.Cmd {
	if m.page != pageRadar && m.native {
		m.clearRadarImage()
		return tea.ClearScreen
	}
	return nil
}

// View renders the active application page.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	var b strings.Builder
	if m.clear {
		b.WriteString(maps.ClearNativeImages())
		m.clear = false
	}
	b.WriteString(ui.Title.Render("Weather CLI"))
	b.WriteString("  ")
	b.WriteString(ui.Subtle.Render(m.tabs()))
	b.WriteString("\n\n")
	if m.err != nil && !m.ready {
		b.WriteString(ui.Warn.Render(m.err.Error()))
	} else {
		switch m.page {
		case pageCurrent:
			b.WriteString(m.currentView())
		case pageHourly:
			b.WriteString(m.hourlyView())
		case pageDaily:
			b.WriteString(m.dailyView())
		case pageDetails:
			b.WriteString(m.detailsView())
		case pageRadar:
			b.WriteString(m.radarView())
		case pageSettings:
			b.WriteString(m.settingsView())
		case pageIcons:
			b.WriteString(m.iconsView())
		}
	}
	b.WriteString("\n\n")
	b.WriteString(ui.Subtle.Render(m.navLine()))
	b.WriteString("\n")
	b.WriteString(ui.Subtle.Render(m.status))
	return b.String()
}

func (m Model) navLine() string {
	if m.editing {
		return "enter apply · esc cancel · text edits current setting"
	}
	if m.debug {
		return "↑/↓ select icon · a animation · q quit"
	}
	if m.page == pageSettings {
		return "←/→ pages · ↑/↓ setting · enter edit/cycle · s save · r refresh · q quit"
	}
	if m.page == pageRadar {
		return "←/→ pages · ↑/↓ frame · +/- zoom · m render · p autoplay · r refresh · a animation · q quit"
	}
	return "←/→ pages · ↑/↓ select · 1-6 jump · r refresh · a animation · q quit"
}

func (m Model) tabs() string {
	if m.debug {
		return ui.Accent.Render("[Icon Test]")
	}
	parts := make([]string, len(pageNames))
	for i, name := range pageNames {
		if page(i) == m.page {
			parts[i] = ui.Accent.Render(fmt.Sprintf("[%d %s]", i+1, name))
		} else {
			parts[i] = fmt.Sprintf("%d %s", i+1, name)
		}
	}
	return strings.Join(parts, "  ")
}

func (m Model) currentView() string {
	r := m.report
	unit := tempUnit(m.cfg.Units)
	wind := windUnit(m.cfg.Units)
	art := m.icon(r.Current.Condition, r.Current.IsDay)
	body := fmt.Sprintf("%s, %s\n\n%s\n\n%.1f%s  feels %.1f%s\n%s · code %d\nWind %.1f %s from %d° · gust %.1f\nCloud %d%% · Pressure %.0f hPa · Precip %.1f mm",
		ui.Accent.Render(r.Location.Name), r.Location.Country, art, r.Current.Temperature, unit, r.Current.Apparent, unit,
		conditionLabel(r.Current.Condition), r.Current.Code, r.Current.WindSpeed, wind, r.Current.WindDirection, r.Current.WindGusts,
		r.Current.CloudCover, r.Current.Pressure, r.Current.Precipitation)
	return ui.Panel.Width(clamp(m.width-4, 40, 100)).Render(body)
}

func (m Model) hourlyView() string {
	var b strings.Builder
	limit := minInt(len(m.report.Hourly), 24)
	if limit == 0 {
		return ui.Panel.Width(contentWidth(m.width)).Render("No hourly forecast loaded yet.")
	}
	selected := m.cursor % limit
	visible := clamp((m.height-8)/6, 1, limit)
	start := centeredStart(selected, visible, limit)
	_, _ = fmt.Fprintf(&b, "%s %s\n\n", ui.Accent.Render("Next 24 hours"), ui.Subtle.Render(fmt.Sprintf("%d/%d", selected+1, limit)))
	for i := start; i < start+visible; i++ {
		h := m.report.Hourly[i]
		card := forecastCard(
			m.icon(h.Condition, true),
			h.Condition,
			h.Time.Format("Mon 15:04"),
			fmt.Sprintf("%.1f%s", h.Temp, tempUnit(m.cfg.Units)),
			fmt.Sprintf("rain %d%% · precip %.1f mm", h.Pop, h.Precip),
			fmt.Sprintf("cloud %d%% · wind %.1f %s", h.CloudCover, h.WindSpeed, windUnit(m.cfg.Units)),
			i == selected,
		)
		b.WriteString(card)
		b.WriteString("\n")
	}
	return ui.Panel.Width(contentWidth(m.width)).Render(b.String())
}

func (m Model) dailyView() string {
	var b strings.Builder
	limit := len(m.report.Daily)
	if limit == 0 {
		return ui.Panel.Width(contentWidth(m.width)).Render("No daily forecast loaded yet.")
	}
	selected := m.cursor % limit
	visible := clamp((m.height-8)/6, 1, limit)
	start := centeredStart(selected, visible, limit)
	_, _ = fmt.Fprintf(&b, "%s %s\n\n", ui.Accent.Render("7 day forecast"), ui.Subtle.Render(fmt.Sprintf("%d/%d", selected+1, limit)))
	for i := start; i < start+visible; i++ {
		d := m.report.Daily[i]
		card := forecastCard(
			m.icon(d.Condition, true),
			d.Condition,
			d.Date.Format("Mon 02 Jan"),
			fmt.Sprintf("%.1f / %.1f%s", d.TempMin, d.TempMax, tempUnit(m.cfg.Units)),
			fmt.Sprintf("rain %d%% · precip %.1f mm", d.PopMax, d.PrecipSum),
			fmt.Sprintf("wind %.1f %s · sun %s-%s", d.WindMax, windUnit(m.cfg.Units), d.Sunrise.Format("15:04"), d.Sunset.Format("15:04")),
			i == selected,
		)
		b.WriteString(card)
		b.WriteString("\n")
	}
	return ui.Panel.Width(contentWidth(m.width)).Render(b.String())
}

func (m Model) detailsView() string {
	c := m.report.Current
	body := fmt.Sprintf("Location: %.4f, %.4f\nTime: %s\nWeather code: %d\nCondition: %s\nDaylight: %t\nTemperature: %.1f%s\nApparent: %.1f%s\nCloud cover: %d%%\nPressure: %.0f hPa\nWind: %.1f %s\nGusts: %.1f %s\nPrecipitation: %.1f mm",
		m.report.Location.Latitude, m.report.Location.Longitude, c.Time.Format(time.RFC1123), c.Code, c.Condition, c.IsDay, c.Temperature, tempUnit(m.cfg.Units), c.Apparent, tempUnit(m.cfg.Units), c.CloudCover, c.Pressure, c.WindSpeed, windUnit(m.cfg.Units), c.WindGusts, windUnit(m.cfg.Units), c.Precipitation)
	return ui.Panel.Width(clamp(m.width-4, 50, 100)).Render(body)
}

func (m Model) radarView() string {
	if len(m.tl.Frames) == 0 {
		msg := m.tl.Message
		if msg == "" {
			msg = "Radar timeline not loaded yet. Press r to refresh."
		}
		return ui.Panel.Width(clamp(m.width-4, 50, 100)).Render(msg)
	}
	idx := m.cursor % len(m.tl.Frames)
	f := m.tl.Frames[idx]
	image := m.radar
	if image == "" {
		image = "Rendering radar tile..."
	}
	renderMode := m.cfg.RenderMode
	if m.render != "" && m.render != m.cfg.RenderMode {
		renderMode = fmt.Sprintf("%s -> %s", m.cfg.RenderMode, m.render)
	}
	body := fmt.Sprintf("Provider: %s · Layer: %s · Zoom: %d · Render: %s · Frame: %d/%d · %s · %s\n\n%s\n\n%s\n%s",
		m.tl.Provider, m.tl.Layer, m.zoom, renderMode, idx+1, len(m.tl.Frames), f.Kind, f.Time.Format(time.RFC1123), image, m.tl.Attribution, ui.Subtle.Render(f.URL))
	if m.native {
		return body
	}
	return ui.Panel.Width(contentWidth(m.width)).Render(body)
}

func (m Model) settingsView() string {
	anim := "off"
	if m.cfg.Animation {
		anim = "on"
	}
	autoplay := "off"
	if m.cfg.RadarAutoplay {
		autoplay = "on"
	}
	rows := []string{
		"Location: " + m.cfg.LocationName,
		fmt.Sprintf("Latitude: %.4f", m.cfg.Latitude),
		fmt.Sprintf("Longitude: %.4f", m.cfg.Longitude),
		"Units: " + m.cfg.Units,
		"Map provider: " + m.cfg.MapProvider,
		"Render mode: " + m.cfg.RenderMode,
		"Icon style: " + m.cfg.IconStyle,
		"Radar autoplay: " + autoplay,
		"Animation: " + anim,
	}
	selected := m.cursor % len(rows)
	for i := range rows {
		if i == selected {
			value := rows[i]
			if m.editing {
				value = settingLabel(i) + ": " + m.input.View()
			}
			rows[i] = ui.Selected.Render(value)
		}
	}
	body := "Config\n\n" + strings.Join(rows, "\n") + "\n\nunits/provider/render/icons/autoplay/animation cycle with enter · text fields edit with enter · changes save automatically\n.env values still override saved config on startup"
	return ui.Panel.Width(clamp(m.width-4, 50, 100)).Render(body)
}

func (m Model) iconsView() string {
	conditions := []weather.Condition{
		weather.Clear, weather.MainlyClear, weather.PartlyCloudy, weather.Overcast, weather.Fog, weather.Drizzle,
		weather.FreezingRain, weather.Rain, weather.Snow, weather.Showers, weather.Thunderstorm, weather.Unknown,
	}
	var b strings.Builder
	selected := m.cursor % len(conditions)
	visible := clamp((m.height-8)/6, 1, len(conditions))
	start := selected - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > len(conditions) {
		start = len(conditions) - visible
	}
	_, _ = fmt.Fprintf(&b, "Icon animation gallery · %d/%d\n\n", selected+1, len(conditions))
	for i := start; i < start+visible; i++ {
		cond := conditions[i]
		asciiFrame := renderIconFrame(icons.FrameDataFor(cond, true, m.tick, icons.ASCII))
		unicodeFrame := renderIconFrame(icons.FrameDataFor(cond, true, m.tick, icons.Unicode))
		frame := lipgloss.JoinHorizontal(lipgloss.Top, asciiFrame, "   ", unicodeFrame)
		label := fmt.Sprintf("%-15s", cond)
		if i == selected {
			label = ui.Selected.Render(label)
		}
		line := fmt.Sprintf("%s\n%s", label, frame)
		b.WriteString(line)
		b.WriteString("\n\n")
	}
	return ui.Panel.Width(clamp(m.width-4, 50, 110)).Render(b.String())
}

func (m Model) loadWeather() tea.Cmd {
	return func() tea.Msg {
		loc := weather.Location{Name: m.cfg.LocationName, Latitude: m.cfg.Latitude, Longitude: m.cfg.Longitude, Timezone: "auto"}
		if m.cfg.LocationName != "" {
			if found, err := m.weather.Geocode(m.ctx, m.cfg.LocationName); err == nil {
				loc = found
			}
		}
		report, err := m.weather.Forecast(m.ctx, loc, m.cfg.Units)
		return reportMsg{report: report, err: err}
	}
}

func (m Model) maybeLoadRadar() tea.Cmd {
	if m.page != pageRadar {
		return nil
	}
	if len(m.tl.Frames) == 0 && m.report.Location.Latitude != 0 {
		return m.loadRadar()
	}
	if len(m.tl.Frames) > 0 && m.radar == "" {
		return m.loadRadarImage()
	}
	return nil
}

func (m Model) loadRadar() tea.Cmd {
	return func() tea.Msg {
		m.maps.SetConfig(m.cfg)
		loc := m.report.Location
		if loc.Latitude == 0 && loc.Longitude == 0 {
			loc = weather.Location{Name: m.cfg.LocationName, Latitude: m.cfg.Latitude, Longitude: m.cfg.Longitude}
		}
		tl, err := m.maps.Timeline(m.ctx, loc, "precipitation_new", m.zoom)
		return radarMsg{timeline: tl, err: err}
	}
}

func (m Model) loadRadarImage() tea.Cmd {
	return func() tea.Msg {
		m.maps.SetConfig(m.cfg)
		if len(m.tl.Frames) == 0 {
			return radarImageMsg{}
		}
		idx := m.cursor % len(m.tl.Frames)
		width, height := m.radarSize()
		frame, err := m.maps.RenderFrame(m.ctx, m.tl.Frames[idx], width, height, m.cfg.RenderMode)
		return radarImageMsg{frame: frame, err: err}
	}
}

func (m Model) radarSize() (int, int) {
	maxWidth := clamp(m.width-8, 32, 220)
	maxHeight := clamp(m.height-14, 8, 80)
	width := maxWidth
	height := width / 2
	if height > maxHeight {
		height = maxHeight
		width = height * 2
	}
	return clamp(width, 32, 220), clamp(height, 8, 80)
}

func (m *Model) startEditing() (tea.Model, tea.Cmd) {
	field := m.cursor % settingCount()
	m.editField = field
	switch field {
	case 3:
		if m.cfg.Units == "metric" {
			m.cfg.Units = "imperial"
		} else {
			m.cfg.Units = "metric"
		}
		if err := m.persistConfig(); err != nil {
			m.status = "Save failed: " + err.Error()
		}
		return m, m.loadWeather()
	case 4:
		m.cfg.MapProvider = nextMapProvider(m.cfg.MapProvider)
		if err := m.persistConfig(); err != nil {
			m.status = "Save failed: " + err.Error()
		}
		hadNative := m.clearRadarImage()
		m.maps.SetConfig(m.cfg)
		if hadNative {
			return m, tea.Batch(tea.ClearScreen, m.loadRadar())
		}
		return m, m.loadRadar()
	case 5:
		m.cfg.RenderMode = nextRenderMode(m.cfg.RenderMode)
		if err := m.persistConfig(); err != nil {
			m.status = "Save failed: " + err.Error()
		}
		hadNative := m.clearRadarImage()
		m.maps.SetConfig(m.cfg)
		if hadNative {
			return m, tea.Batch(tea.ClearScreen, m.loadRadarImage())
		}
		return m, m.loadRadarImage()
	case 6:
		m.cfg.IconStyle = nextIconStyle(m.cfg.IconStyle)
		if err := m.persistConfig(); err != nil {
			m.status = "Save failed: " + err.Error()
		}
		return m, nil
	case 7:
		m.cfg.RadarAutoplay = !m.cfg.RadarAutoplay
		if err := m.persistConfig(); err != nil {
			m.status = "Save failed: " + err.Error()
		}
		if m.cfg.RadarAutoplay {
			return m, radarTick()
		}
		return m, nil
	case 8:
		m.cfg.Animation = !m.cfg.Animation
		if err := m.persistConfig(); err != nil {
			m.status = "Save failed: " + err.Error()
		}
		if m.cfg.Animation {
			return m, tick()
		}
		return m, nil
	}
	m.editing = true
	m.input.SetValue(settingValue(m.cfg, field))
	m.input.Focus()
	m.input.CursorEnd()
	return m, textinput.Blink
}

func (m *Model) updateEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editing = false
		m.input.Blur()
		return m, nil
	case "enter":
		applied := m.applySetting(m.editField%settingCount(), m.input.Value())
		m.editing = false
		m.input.Blur()
		if applied {
			if err := m.persistConfig(); err != nil {
				m.status = "Save failed: " + err.Error()
			}
		}
		if m.editField%settingCount() <= 4 {
			return m, tea.Batch(m.loadWeather(), m.loadRadar())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) applySetting(field int, value string) bool {
	value = strings.TrimSpace(value)
	switch field {
	case 0:
		m.cfg.LocationName = value
	case 1:
		if _, err := fmt.Sscanf(value, "%f", &m.cfg.Latitude); err != nil {
			m.status = "Invalid latitude"
			return false
		}
	case 2:
		if _, err := fmt.Sscanf(value, "%f", &m.cfg.Longitude); err != nil {
			m.status = "Invalid longitude"
			return false
		}
	case 3:
		if value == "imperial" {
			m.cfg.Units = "imperial"
		} else {
			m.cfg.Units = "metric"
		}
	case 4:
		switch value {
		case "rainviewer", "openmeteo":
			m.cfg.MapProvider = value
		default:
			m.cfg.MapProvider = "auto"
		}
	case 5:
		switch value {
		case "viu", "termimg", "blocks":
			m.cfg.RenderMode = value
		default:
			m.cfg.RenderMode = "auto"
		}
	case 6:
		switch value {
		case "unicode":
			m.cfg.IconStyle = "unicode"
		default:
			m.cfg.IconStyle = "ascii"
		}
	case 7:
		m.cfg.RadarAutoplay = truthy(value)
	}
	return true
}

func (m *Model) saveConfig() (tea.Model, tea.Cmd) {
	if err := m.persistConfig(); err != nil {
		m.status = "Save failed: " + err.Error()
		return m, nil
	}
	m.status = "Config saved"
	return m, nil
}

func (m *Model) persistConfig() error {
	if err := config.Save(m.cfg); err != nil {
		m.status = "Save failed: " + err.Error()
		return err
	}
	m.status = "Config saved"
	return nil
}

func tick() tea.Cmd {
	return tea.Tick(450*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func radarTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return radarTickMsg(t) })
}

func pageFromName(name string) page {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "hourly":
		return pageHourly
	case "daily":
		return pageDaily
	case "details":
		return pageDetails
	case "radar":
		return pageRadar
	case "settings", "config":
		return pageSettings
	case "icons", "icon-test":
		return pageIcons
	default:
		return pageCurrent
	}
}

func nextMapProvider(current string) string {
	switch current {
	case "auto":
		return "rainviewer"
	case "rainviewer":
		return "openmeteo"
	default:
		return "auto"
	}
}

func nextRenderMode(current string) string {
	switch current {
	case "auto":
		return "viu"
	case "viu":
		return "termimg"
	case "termimg":
		return "blocks"
	default:
		return "auto"
	}
}

func nextIconStyle(current string) string {
	if current == "ascii" {
		return "unicode"
	}
	return "ascii"
}

func settingLabel(field int) string {
	labels := []string{"Location", "Latitude", "Longitude", "Units", "Map provider", "Render mode", "Icon style", "Radar autoplay", "Animation"}
	return labels[field%len(labels)]
}

func settingValue(cfg config.Config, field int) string {
	switch field {
	case 0:
		return cfg.LocationName
	case 1:
		return fmt.Sprintf("%.4f", cfg.Latitude)
	case 2:
		return fmt.Sprintf("%.4f", cfg.Longitude)
	case 3:
		return cfg.Units
	case 4:
		return cfg.MapProvider
	case 5:
		return cfg.RenderMode
	case 6:
		return cfg.IconStyle
	case 7:
		if cfg.RadarAutoplay {
			return "on"
		}
		return "off"
	default:
		if cfg.Animation {
			return "on"
		}
		return "off"
	}
}

func settingCount() int {
	return 9
}

func tempUnit(units string) string {
	if units == "imperial" {
		return "°F"
	}
	return "°C"
}

func windUnit(units string) string {
	if units == "imperial" {
		return "mph"
	}
	return "km/h"
}

func clamp(v, lower, upper int) int {
	if v < lower {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}

func contentWidth(width int) int {
	if width < 44 {
		return 40
	}
	return width - 4
}

func forecastCard(icon string, cond weather.Condition, title, temp, line1, line2 string, selected bool) string {
	condition := conditionLabel(cond)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	tempStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	if selected {
		titleStyle = titleStyle.Foreground(lipgloss.Color("230"))
		metaStyle = metaStyle.Foreground(lipgloss.Color("255"))
	}
	info := strings.Join([]string{
		titleStyle.Render(title + "  " + tempStyle.Render(temp)),
		condition,
		metaStyle.Render(line1),
		metaStyle.Render(line2),
	}, "\n")
	card := lipgloss.JoinHorizontal(lipgloss.Top, icon, "  ", info)
	if selected {
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("39")).
			PaddingLeft(1).
			Render(card)
	}
	return lipgloss.NewStyle().PaddingLeft(2).Render(card)
}

func (m Model) icon(cond weather.Condition, day bool) string {
	return renderIconFrame(icons.FrameDataFor(cond, day, m.tick, iconStyle(m.cfg.IconStyle)))
}

func renderIconFrame(frame icons.FrameData) string {
	lines := make([]string, len(frame))
	for i, line := range frame {
		var b strings.Builder
		for _, seg := range line {
			b.WriteString(iconRoleStyle(seg.Role).Render(seg.Text))
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
}

func iconRoleStyle(role string) lipgloss.Style {
	switch role {
	case "sun":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	case "moon":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	case "cloud":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	case "fog":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	case "rain", "drizzle":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	case "snow":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("159"))
	case "lightning":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	case "unknown":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	default:
		return lipgloss.NewStyle()
	}
}

func iconStyle(value string) icons.Style {
	if value == "unicode" {
		return icons.Unicode
	}
	return icons.ASCII
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func conditionLabel(cond weather.Condition) string {
	return lipgloss.NewStyle().Bold(true).Foreground(conditionColor(cond)).Render(string(cond))
}

func conditionColor(cond weather.Condition) lipgloss.Color {
	switch cond {
	case weather.Clear, weather.MainlyClear:
		return lipgloss.Color("220")
	case weather.PartlyCloudy, weather.Overcast:
		return lipgloss.Color("250")
	case weather.Fog:
		return lipgloss.Color("245")
	case weather.Drizzle, weather.Rain, weather.Showers, weather.FreezingRain:
		return lipgloss.Color("39")
	case weather.Snow:
		return lipgloss.Color("153")
	case weather.Thunderstorm:
		return lipgloss.Color("171")
	default:
		return lipgloss.Color("244")
	}
}

func centeredStart(selected, visible, total int) int {
	start := selected - visible/2
	if start < 0 {
		return 0
	}
	if start+visible > total {
		return total - visible
	}
	return start
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

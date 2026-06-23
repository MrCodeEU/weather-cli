// Package icons renders animated terminal weather icons.
package icons

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"weather-cli/internal/weather"
)

const (
	// Width is the display-cell width of every icon frame.
	Width = 9
	// Height is the line count of every icon frame.
	Height = 5
)

// Style selects the icon character set.
type Style string

const (
	// ASCII uses plain ASCII characters only.
	ASCII Style = "ascii"
	// Unicode uses weather symbols and box drawing where useful.
	Unicode Style = "unicode"
)

// FrameLine is one colored segment line in an icon frame.
type FrameLine []Segment

// Segment is a run of text with a semantic weather role.
type Segment struct {
	Role string
	Text string
}

// FrameData is a fixed-size icon frame split into semantic color segments.
type FrameData []FrameLine

// Animation is a terminal weather icon split into display frames.
type Animation struct {
	Frames []FrameData
}

// For returns the animation for a weather condition.
func For(cond weather.Condition, day bool, style Style) Animation {
	if style == Unicode {
		return Animation{Frames: unicodeFrames(cond, day)}
	}
	return Animation{Frames: asciiFrames(cond, day)}
}

// Frame returns one plain-text animation frame for a condition and tick.
func Frame(cond weather.Condition, day bool, tick int) string {
	return Plain(FrameDataFor(cond, day, tick, ASCII))
}

// FrameForStyle returns one plain-text animation frame for a condition, style, and tick.
func FrameForStyle(cond weather.Condition, day bool, tick int, style Style) string {
	return Plain(FrameDataFor(cond, day, tick, style))
}

// FrameDataFor returns one segmented animation frame for a condition, style, and tick.
func FrameDataFor(cond weather.Condition, day bool, tick int, style Style) FrameData {
	a := For(cond, day, style)
	if len(a.Frames) == 0 {
		return frame(segments("unknown", "   ???   "), segments("unknown", "  ?   ?  "), segments("unknown", "   ???   "))
	}
	return a.Frames[tick%len(a.Frames)]
}

// Plain renders segmented frame data without color.
func Plain(f FrameData) string {
	lines := make([]string, len(f))
	for i, line := range f {
		var b strings.Builder
		for _, seg := range line {
			b.WriteString(seg.Text)
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
}

func asciiFrames(cond weather.Condition, day bool) []FrameData {
	switch cond {
	case weather.Clear:
		if !day {
			return []FrameData{
				frame(s("moon", "   .-.   "), s("moon", "  (   )  "), s("moon", "   `-'   "), blank(), blank()),
				frame(s("moon", "    .-.  "), s("moon", "   (   ) "), s("moon", "    `-'  "), blank(), blank()),
				frame(s("moon", "   .-.   "), s("moon", "  (   )  "), s("moon", "   `-'   "), blank(), blank()),
				frame(s("moon", "  .-.    "), s("moon", " (   )   "), s("moon", "  `-'    "), blank(), blank()),
			}
		}
		return []FrameData{
			frame(blank(), s("sun", "  \\ | /  "), s("sun", " -- O -- "), s("sun", "  / | \\  "), blank()),
			frame(blank(), s("sun", "   \\|/   "), s("sun", " -- O -- "), s("sun", "   /|\\   "), blank()),
			frame(blank(), s("sun", "  / | \\  "), s("sun", " -- O -- "), s("sun", "  \\ | /  "), blank()),
			frame(blank(), s("sun", "   /|\\   "), s("sun", " -- O -- "), s("sun", "   \\|/   "), blank()),
		}
	case weather.MainlyClear:
		return mixedSunCloud([]string{"  \\|/    ", " --O-.   ", "  /|( )  ", "    `-'  ", "         "})
	case weather.PartlyCloudy:
		return mixedSunCloud([]string{" \\ |     ", " - O--.  ", " / |(  ) ", "    `--' ", "         "})
	case weather.Overcast:
		return []FrameData{
			cloud(" .--.    ", "(    ).-.", "`--(    )", "   `--'  ", "         "),
			cloud("  .--.   ", " (    ).-", " `--(   )", "    `--' ", "         "),
			cloud(" .--.    ", "(    ).-.", "`--(    )", "   `--'  ", "         "),
			cloud("  .--.   ", " (    ).-", " `--(   )", "    `--' ", "         "),
		}
	case weather.Fog:
		return []FrameData{
			fog("  .--.   ", " (    )  ", "  `--'   ", " ~~~~~~~ ", "~~~~~~~~~"),
			fog("  .--.   ", " (    )  ", "  `--'   ", "~~~~~~~~~", " ~~~~~~~ "),
			fog("  .--.   ", " (    )  ", "  `--'   ", "  ~~~~~~~", "~~~~~~~~ "),
			fog("  .--.   ", " (    )  ", "  `--'   ", "~~~~~~~~ ", " ~~~~~~~ "),
		}
	case weather.Drizzle:
		return precip("drizzle", []string{"   ' '   ", "  ' '    ", " ' '     ", "  ' '    "})
	case weather.Rain, weather.FreezingRain, weather.Showers:
		return precip("rain", []string{"  | | |  ", " | | |   ", "  / / /  ", " / / /   "})
	case weather.Snow:
		return precip("snow", []string{"  * * *  ", " * * *   ", "  + + +  ", " + + +   "})
	case weather.Thunderstorm:
		return thunder()
	default:
		return []FrameData{frame(s("unknown", "   ???   "), s("unknown", "  ?   ?  "), s("unknown", "   ???   "), blank(), blank())}
	}
}

func unicodeFrames(cond weather.Condition, day bool) []FrameData {
	switch cond {
	case weather.Clear:
		if !day {
			return []FrameData{
				frame(s("moon", "    ◐    "), s("moon", "   ◑     "), s("moon", "    ◐    "), blank(), blank()),
				frame(s("moon", "    ◓    "), s("moon", "   ◒     "), s("moon", "    ◓    "), blank(), blank()),
				frame(s("moon", "    ◑    "), s("moon", "   ◐     "), s("moon", "    ◑    "), blank(), blank()),
				frame(s("moon", "    ◒    "), s("moon", "   ◓     "), s("moon", "    ◒    "), blank(), blank()),
			}
		}
		return []FrameData{
			frame(blank(), s("sun", "  ╲ │ ╱  "), s("sun", " ─ ☀ ─  "), s("sun", "  ╱ │ ╲  "), blank()),
			frame(blank(), s("sun", "   ╲│╱   "), s("sun", " ─ ☀ ─  "), s("sun", "   ╱│╲   "), blank()),
			frame(blank(), s("sun", "  ╱ │ ╲  "), s("sun", " ─ ☀ ─  "), s("sun", "  ╲ │ ╱  "), blank()),
			frame(blank(), s("sun", "   ╱│╲   "), s("sun", " ─ ☀ ─  "), s("sun", "   ╲│╱   "), blank()),
		}
	case weather.MainlyClear:
		return unicodeMixed([]string{"  ╲│╱    ", " ─☀─☁   ", "  ╱│☁   ", "    ☁   ", "         "})
	case weather.PartlyCloudy:
		return unicodeMixed([]string{" ╲ │     ", " ─ ☀ ☁  ", " ╱ │☁☁  ", "    ☁   ", "         "})
	case weather.Overcast:
		return []FrameData{
			cloud("  ☁☁☁   ", " ☁☁☁☁  ", "  ☁☁☁   ", "         ", "         "),
			cloud(" ☁☁☁    ", "  ☁☁☁☁ ", "   ☁☁☁  ", "         ", "         "),
			cloud("  ☁☁☁   ", " ☁☁☁☁  ", "  ☁☁☁   ", "         ", "         "),
			cloud("   ☁☁☁  ", " ☁☁☁☁  ", "  ☁☁☁   ", "         ", "         "),
		}
	case weather.Fog:
		return []FrameData{
			fog("  ☁☁☁   ", " ☁☁☁☁  ", "         ", " ≋≋≋≋≋≋ ", "≋≋≋≋≋≋≋≋≋"),
			fog("  ☁☁☁   ", " ☁☁☁☁  ", "         ", "≋≋≋≋≋≋≋≋≋", " ≋≋≋≋≋≋ "),
			fog("  ☁☁☁   ", " ☁☁☁☁  ", "         ", "  ≋≋≋≋≋≋", "≋≋≋≋≋≋≋ "),
			fog("  ☁☁☁   ", " ☁☁☁☁  ", "         ", "≋≋≋≋≋≋≋ ", " ≋≋≋≋≋≋ "),
		}
	case weather.Drizzle:
		return unicodePrecip("drizzle", []string{"   ˙ ˙   ", "  ˙ ˙    ", " ˙ ˙     ", "  ˙ ˙    "})
	case weather.Rain, weather.FreezingRain, weather.Showers:
		return unicodePrecip("rain", []string{"  ╷ ╷ ╷  ", " ╷ ╷ ╷   ", "  ╱ ╱ ╱  ", " ╱ ╱ ╱   "})
	case weather.Snow:
		return unicodePrecip("snow", []string{"  ✦ ✦ ✦  ", " ✦ ✦ ✦   ", "  ✧ ✧ ✧  ", " ✧ ✧ ✧   "})
	case weather.Thunderstorm:
		return unicodeThunder()
	default:
		return []FrameData{frame(s("unknown", "   ???   "), s("unknown", "  ?   ?  "), s("unknown", "   ???   "), blank(), blank())}
	}
}

func mixedSunCloud(lines []string) []FrameData {
	return []FrameData{
		mixed(lines),
		mixed(shift(lines, 1)),
		mixed(lines),
		mixed(shift(lines, -1)),
	}
}

func unicodeMixed(lines []string) []FrameData {
	return []FrameData{
		mixed(lines),
		mixed(shift(lines, 1)),
		mixed(lines),
		mixed(shift(lines, -1)),
	}
}

func precip(role string, drops []string) []FrameData {
	out := make([]FrameData, 0, len(drops))
	for _, drop := range drops {
		out = append(out, frame(s("cloud", "  .--.   "), s("cloud", " (    )  "), s("cloud", "  `--'   "), s(role, drop), blank()))
	}
	return out
}

func unicodePrecip(role string, drops []string) []FrameData {
	out := make([]FrameData, 0, len(drops))
	for _, drop := range drops {
		out = append(out, frame(s("cloud", "  ☁☁☁   "), s("cloud", " ☁☁☁☁  "), blank(), s(role, drop), blank()))
	}
	return out
}

func thunder() []FrameData {
	return []FrameData{
		frame(s("cloud", "  .--.   "), s("cloud", " (    )  "), s("cloud", "  `--'   "), segments("rain", "  / / ", "lightning", "Z "), blank()),
		frame(s("cloud", "  .--.   "), s("cloud", " (    )  "), s("cloud", "  `--'   "), segments("lightning", " Z", "rain", " / /   "), blank()),
		frame(s("cloud", "  .--.   "), s("cloud", " (    )  "), s("cloud", "  `--'   "), segments("rain", "  / / ", "lightning", "Z "), blank()),
		frame(s("cloud", "  .--.   "), s("cloud", " (    )  "), s("cloud", "  `--'   "), segments("lightning", "Z ", "rain", " / /   "), blank()),
	}
}

func unicodeThunder() []FrameData {
	return []FrameData{
		frame(s("cloud", "  ☁☁☁   "), s("cloud", " ☁☁☁☁  "), blank(), segments("rain", "  ╱ ╱ ", "lightning", "⚡ "), blank()),
		frame(s("cloud", "  ☁☁☁   "), s("cloud", " ☁☁☁☁  "), blank(), segments("lightning", " ⚡", "rain", " ╱ ╱   "), blank()),
		frame(s("cloud", "  ☁☁☁   "), s("cloud", " ☁☁☁☁  "), blank(), segments("rain", "  ╱ ╱ ", "lightning", "⚡ "), blank()),
		frame(s("cloud", "  ☁☁☁   "), s("cloud", " ☁☁☁☁  "), blank(), segments("lightning", "⚡ ", "rain", " ╱ ╱   "), blank()),
	}
}

func cloud(lines ...string) FrameData {
	out := make(FrameData, 0, Height)
	for _, line := range lines {
		out = append(out, s("cloud", line))
	}
	return normalize(out)
}

func fog(lines ...string) FrameData {
	out := make(FrameData, 0, Height)
	for i, line := range lines {
		role := "fog"
		if i < 3 {
			role = "cloud"
		}
		out = append(out, s(role, line))
	}
	return normalize(out)
}

func mixed(lines []string) FrameData {
	out := make(FrameData, 0, Height)
	for _, line := range lines {
		if strings.ContainsAny(line, "O☀|/\\─╲╱│()'`☁") {
			out = append(out, mixedLine(line))
			continue
		}
		out = append(out, s("sun", line))
	}
	return normalize(out)
}

func mixedLine(line string) FrameLine {
	var out FrameLine
	cloudStarted := false
	for _, r := range line {
		if strings.ContainsRune(".()`'☁", r) {
			cloudStarted = true
		}
		next := "sun"
		if cloudStarted || strings.ContainsRune("()", r) {
			next = "cloud"
		}
		if len(out) == 0 || out[len(out)-1].Role != next {
			out = append(out, Segment{Role: next, Text: string(r)})
		} else {
			out[len(out)-1].Text += string(r)
		}
	}
	return padLine(out)
}

func shift(lines []string, n int) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		if n > 0 {
			out[i] = strings.Repeat(" ", n) + strings.TrimRight(line, " ")
		} else if n < 0 {
			out[i] = strings.TrimLeft(line, " ") + strings.Repeat(" ", -n)
		} else {
			out[i] = line
		}
	}
	return out
}

func frame(lines ...FrameLine) FrameData {
	return normalize(lines)
}

func normalize(lines []FrameLine) FrameData {
	out := make(FrameData, 0, Height)
	for i := 0; i < Height; i++ {
		if i < len(lines) {
			out = append(out, padLine(lines[i]))
		} else {
			out = append(out, blank())
		}
	}
	return out
}

func s(role, text string) FrameLine {
	return padLine(FrameLine{{Role: role, Text: text}})
}

func segments(parts ...string) FrameLine {
	var line FrameLine
	for i := 0; i+1 < len(parts); i += 2 {
		line = append(line, Segment{Role: parts[i], Text: parts[i+1]})
	}
	return padLine(line)
}

func blank() FrameLine {
	return s("", "")
}

func padLine(line FrameLine) FrameLine {
	text := ""
	for _, seg := range line {
		text += seg.Text
	}
	width := runewidth.StringWidth(text)
	if width > Width {
		var clipped FrameLine
		for _, seg := range line {
			var b strings.Builder
			for _, r := range seg.Text {
				next := b.String() + string(r)
				if runewidth.StringWidth(lineText(clipped)+next) > Width {
					break
				}
				b.WriteRune(r)
			}
			if b.Len() > 0 {
				clipped = append(clipped, Segment{Role: seg.Role, Text: b.String()})
			}
			if runewidth.StringWidth(lineText(clipped)) >= Width {
				break
			}
		}
		line = clipped
		width = runewidth.StringWidth(lineText(line))
	}
	if width < Width {
		line = append(line, Segment{Text: strings.Repeat(" ", Width-width)})
	}
	return line
}

func lineText(line FrameLine) string {
	var b strings.Builder
	for _, seg := range line {
		b.WriteString(seg.Text)
	}
	return b.String()
}

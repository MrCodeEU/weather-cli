package icons

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"

	"weather-cli/internal/weather"
)

func TestAllIconFramesHaveFixedSize(t *testing.T) {
	conditions := []weather.Condition{
		weather.Clear,
		weather.MainlyClear,
		weather.PartlyCloudy,
		weather.Overcast,
		weather.Fog,
		weather.Drizzle,
		weather.FreezingRain,
		weather.Rain,
		weather.Snow,
		weather.Showers,
		weather.Thunderstorm,
		weather.Unknown,
	}
	for _, style := range []Style{ASCII, Unicode} {
		for _, cond := range conditions {
			anim := For(cond, true, style)
			if len(anim.Frames) < 4 && cond != weather.Unknown {
				t.Fatalf("%s %s has %d frames, want at least 4", style, cond, len(anim.Frames))
			}
			for i, frame := range anim.Frames {
				lines := strings.Split(Plain(frame), "\n")
				if len(lines) != Height {
					t.Fatalf("%s %s frame %d height got %d, want %d", style, cond, i, len(lines), Height)
				}
				for j, line := range lines {
					if got := runewidth.StringWidth(line); got != Width {
						t.Fatalf("%s %s frame %d line %d width got %d, want %d: %q", style, cond, i, j, got, Width, line)
					}
				}
			}
		}
	}
}

func TestUnicodeIconsUseUnicodeGlyphs(t *testing.T) {
	got := FrameForStyle(weather.Clear, true, 0, Unicode)
	if !strings.Contains(got, "☀") {
		t.Fatalf("unicode clear icon did not contain sun glyph: %q", got)
	}
}

func TestPartlyCloudyLowerSunRaysStaySunColored(t *testing.T) {
	frame := FrameDataFor(weather.PartlyCloudy, true, 0, ASCII)
	line := frame[2]
	var sawSun, sawCloud bool
	for _, seg := range line {
		if seg.Role == "sun" && strings.ContainsAny(seg.Text, "/|") {
			sawSun = true
		}
		if seg.Role == "cloud" && strings.ContainsAny(seg.Text, "()") {
			sawCloud = true
		}
	}
	if !sawSun || !sawCloud {
		t.Fatalf("partly cloudy lower line roles wrong: %+v", line)
	}
}

package weather

import "testing"

func TestConditionForOpenMeteoCodes(t *testing.T) {
	tests := map[int]Condition{
		0:  Clear,
		2:  PartlyCloudy,
		45: Fog,
		55: Drizzle,
		63: Rain,
		67: FreezingRain,
		75: Snow,
		82: Showers,
		95: Thunderstorm,
	}
	for code, want := range tests {
		if got := ConditionFor(code, true); got != want {
			t.Fatalf("code %d: got %q want %q", code, got, want)
		}
	}
}

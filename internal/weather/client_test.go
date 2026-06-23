package weather

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestForecastParsesOpenMeteoResponse(t *testing.T) {
	client := NewOpenMeteoClient(roundTrip(func(_ *http.Request) (*http.Response, error) {
		body := `{
			"current":{"time":"2026-06-21T12:00","temperature_2m":24.5,"apparent_temperature":25.1,"is_day":1,"precipitation":0.2,"weather_code":61,"cloud_cover":80,"pressure_msl":1012,"wind_speed_10m":12,"wind_direction_10m":270,"wind_gusts_10m":22},
			"hourly":{"time":["2026-06-21T12:00"],"temperature_2m":[24.5],"precipitation_probability":[70],"precipitation":[0.2],"weather_code":[61],"cloud_cover":[80],"wind_speed_10m":[12]},
			"daily":{"time":["2026-06-21"],"weather_code":[61],"temperature_2m_max":[25],"temperature_2m_min":[18],"sunrise":["2026-06-21T05:00"],"sunset":["2026-06-21T21:00"],"precipitation_sum":[3.2],"precipitation_probability_max":[80],"wind_speed_10m_max":[22]}
		}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	report, err := client.Forecast(context.Background(), Location{Name: "Vienna", Latitude: 48.2, Longitude: 16.3, Timezone: "Europe/Vienna"}, "metric")
	if err != nil {
		t.Fatal(err)
	}
	if report.Current.Condition != Rain || len(report.Hourly) != 1 || len(report.Daily) != 1 {
		t.Fatalf("unexpected parsed report: %+v", report)
	}
}

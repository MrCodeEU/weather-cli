// Package weather provides Open-Meteo forecast data and normalized weather models.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTPDoer is the subset of http.Client used by weather clients.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// NewHTTPClient returns an HTTP client with the requested timeout.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// OpenMeteoClient queries Open-Meteo geocoding and forecast APIs.
type OpenMeteoClient struct {
	http HTTPDoer
}

// NewOpenMeteoClient creates an Open-Meteo client.
func NewOpenMeteoClient(httpClient HTTPDoer) *OpenMeteoClient {
	return &OpenMeteoClient{http: httpClient}
}

// Location identifies a weather forecast location.
type Location struct {
	Name      string
	Country   string
	Latitude  float64
	Longitude float64
	Timezone  string
}

// Report is the normalized weather forecast shown in the TUI.
type Report struct {
	Location Location
	Current  Current
	Hourly   []Hour
	Daily    []Day
	Fetched  time.Time
	Units    string
}

// Current contains current weather conditions.
type Current struct {
	Time          time.Time
	Temperature   float64
	Apparent      float64
	Precipitation float64
	CloudCover    int
	Pressure      float64
	WindSpeed     float64
	WindDirection int
	WindGusts     float64
	IsDay         bool
	Code          int
	Condition     Condition
}

// Hour contains one hourly forecast entry.
type Hour struct {
	Time       time.Time
	Temp       float64
	Pop        int
	Precip     float64
	CloudCover int
	WindSpeed  float64
	Code       int
	Condition  Condition
}

// Day contains one daily forecast entry.
type Day struct {
	Date      time.Time
	TempMin   float64
	TempMax   float64
	PrecipSum float64
	PopMax    int
	WindMax   float64
	Sunrise   time.Time
	Sunset    time.Time
	Code      int
	Condition Condition
}

// Geocode resolves a location name with Open-Meteo geocoding.
func (c *OpenMeteoClient) Geocode(ctx context.Context, name string) (Location, error) {
	u := "https://geocoding-api.open-meteo.com/v1/search?count=1&language=en&format=json&name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Location{}, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return Location{}, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return Location{}, fmt.Errorf("geocoding failed: %s", res.Status)
	}
	var body struct {
		Results []struct {
			Name      string  `json:"name"`
			Country   string  `json:"country"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Timezone  string  `json:"timezone"`
		} `json:"results"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Location{}, err
	}
	if len(body.Results) == 0 {
		return Location{}, fmt.Errorf("location %q not found", name)
	}
	r := body.Results[0]
	return Location{Name: r.Name, Country: r.Country, Latitude: r.Latitude, Longitude: r.Longitude, Timezone: r.Timezone}, nil
}

// Forecast loads current, hourly, and daily forecast data.
func (c *OpenMeteoClient) Forecast(ctx context.Context, loc Location, units string) (Report, error) {
	if loc.Timezone == "" {
		loc.Timezone = "auto"
	}
	tempUnit, windUnit := "celsius", "kmh"
	if units == "imperial" {
		tempUnit, windUnit = "fahrenheit", "mph"
	}
	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(loc.Latitude, 'f', -1, 64))
	q.Set("longitude", strconv.FormatFloat(loc.Longitude, 'f', -1, 64))
	q.Set("timezone", loc.Timezone)
	q.Set("forecast_days", "7")
	q.Set("current", strings.Join([]string{"temperature_2m", "apparent_temperature", "is_day", "precipitation", "weather_code", "cloud_cover", "pressure_msl", "wind_speed_10m", "wind_direction_10m", "wind_gusts_10m"}, ","))
	q.Set("hourly", strings.Join([]string{"temperature_2m", "precipitation_probability", "precipitation", "weather_code", "cloud_cover", "wind_speed_10m"}, ","))
	q.Set("daily", strings.Join([]string{"weather_code", "temperature_2m_max", "temperature_2m_min", "sunrise", "sunset", "precipitation_sum", "precipitation_probability_max", "wind_speed_10m_max"}, ","))
	q.Set("temperature_unit", tempUnit)
	q.Set("wind_speed_unit", windUnit)
	u := "https://api.open-meteo.com/v1/forecast?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Report{}, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return Report{}, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return Report{}, fmt.Errorf("forecast failed: %s", res.Status)
	}
	var raw forecastResponse
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return Report{}, err
	}
	return convertForecast(loc, units, raw), nil
}

type forecastResponse struct {
	Current struct {
		Time          string  `json:"time"`
		Temperature   float64 `json:"temperature_2m"`
		Apparent      float64 `json:"apparent_temperature"`
		IsDay         int     `json:"is_day"`
		Precipitation float64 `json:"precipitation"`
		Code          int     `json:"weather_code"`
		CloudCover    int     `json:"cloud_cover"`
		Pressure      float64 `json:"pressure_msl"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindDirection int     `json:"wind_direction_10m"`
		WindGusts     float64 `json:"wind_gusts_10m"`
	} `json:"current"`
	Hourly struct {
		Time       []string  `json:"time"`
		Temp       []float64 `json:"temperature_2m"`
		Pop        []int     `json:"precipitation_probability"`
		Precip     []float64 `json:"precipitation"`
		Code       []int     `json:"weather_code"`
		CloudCover []int     `json:"cloud_cover"`
		WindSpeed  []float64 `json:"wind_speed_10m"`
	} `json:"hourly"`
	Daily struct {
		Time      []string  `json:"time"`
		Code      []int     `json:"weather_code"`
		TempMax   []float64 `json:"temperature_2m_max"`
		TempMin   []float64 `json:"temperature_2m_min"`
		Sunrise   []string  `json:"sunrise"`
		Sunset    []string  `json:"sunset"`
		PrecipSum []float64 `json:"precipitation_sum"`
		PopMax    []int     `json:"precipitation_probability_max"`
		WindMax   []float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

func convertForecast(loc Location, units string, raw forecastResponse) Report {
	parse := func(s string) time.Time {
		for _, layout := range []string{"2006-01-02T15:04", "2006-01-02"} {
			if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
				return t
			}
		}
		return time.Time{}
	}
	current := Current{
		Time:          parse(raw.Current.Time),
		Temperature:   raw.Current.Temperature,
		Apparent:      raw.Current.Apparent,
		Precipitation: raw.Current.Precipitation,
		CloudCover:    raw.Current.CloudCover,
		Pressure:      raw.Current.Pressure,
		WindSpeed:     raw.Current.WindSpeed,
		WindDirection: raw.Current.WindDirection,
		WindGusts:     raw.Current.WindGusts,
		IsDay:         raw.Current.IsDay == 1,
		Code:          raw.Current.Code,
		Condition:     ConditionFor(raw.Current.Code, raw.Current.IsDay == 1),
	}
	hours := make([]Hour, 0, minLen(len(raw.Hourly.Time), len(raw.Hourly.Code), 24))
	for i := 0; i < len(raw.Hourly.Time) && i < 48; i++ {
		h := Hour{Time: parse(raw.Hourly.Time[i])}
		if i < len(raw.Hourly.Temp) {
			h.Temp = raw.Hourly.Temp[i]
		}
		if i < len(raw.Hourly.Pop) {
			h.Pop = raw.Hourly.Pop[i]
		}
		if i < len(raw.Hourly.Precip) {
			h.Precip = raw.Hourly.Precip[i]
		}
		if i < len(raw.Hourly.CloudCover) {
			h.CloudCover = raw.Hourly.CloudCover[i]
		}
		if i < len(raw.Hourly.WindSpeed) {
			h.WindSpeed = raw.Hourly.WindSpeed[i]
		}
		if i < len(raw.Hourly.Code) {
			h.Code = raw.Hourly.Code[i]
			h.Condition = ConditionFor(h.Code, true)
		}
		hours = append(hours, h)
	}
	days := make([]Day, 0, len(raw.Daily.Time))
	for i := range raw.Daily.Time {
		d := Day{Date: parse(raw.Daily.Time[i])}
		if i < len(raw.Daily.Code) {
			d.Code = raw.Daily.Code[i]
			d.Condition = ConditionFor(d.Code, true)
		}
		if i < len(raw.Daily.TempMin) {
			d.TempMin = raw.Daily.TempMin[i]
		}
		if i < len(raw.Daily.TempMax) {
			d.TempMax = raw.Daily.TempMax[i]
		}
		if i < len(raw.Daily.PrecipSum) {
			d.PrecipSum = raw.Daily.PrecipSum[i]
		}
		if i < len(raw.Daily.PopMax) {
			d.PopMax = raw.Daily.PopMax[i]
		}
		if i < len(raw.Daily.WindMax) {
			d.WindMax = raw.Daily.WindMax[i]
		}
		if i < len(raw.Daily.Sunrise) {
			d.Sunrise = parse(raw.Daily.Sunrise[i])
		}
		if i < len(raw.Daily.Sunset) {
			d.Sunset = parse(raw.Daily.Sunset[i])
		}
		days = append(days, d)
	}
	return Report{Location: loc, Current: current, Hourly: hours, Daily: days, Fetched: time.Now(), Units: units}
}

func minLen(v int, rest ...int) int {
	m := v
	for _, n := range rest {
		if n < m {
			m = n
		}
	}
	return m
}

package webserver

import (
	"net/http/httptest"
	"slices"
	"testing"
)

func TestLaunchArgsIncludesValidBrowserCoordinates(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws?lat=48.2&lon=16.3", nil)
	args := launchArgs(req)
	if !slices.Contains(args, "--lat") || !slices.Contains(args, "48.2") || !slices.Contains(args, "--lon") || !slices.Contains(args, "16.3") {
		t.Fatalf("args did not include browser coordinates: %#v", args)
	}
}

func TestLaunchArgsRejectsInvalidBrowserCoordinates(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws?lat=500&lon=16.3", nil)
	args := launchArgs(req)
	if slices.Contains(args, "--lat") || slices.Contains(args, "--lon") {
		t.Fatalf("args should reject invalid coordinates: %#v", args)
	}
}

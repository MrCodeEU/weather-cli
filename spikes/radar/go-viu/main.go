package main

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"os/exec"

	"golang.org/x/term"

	"weather-cli-radar-spikes/shared"
)

func main() {
	viu, err := exec.LookPath("viu")
	if err != nil {
		fmt.Fprintln(os.Stderr, "viu is not installed")
		os.Exit(1)
	}
	ctx := context.Background()
	frames, err := shared.LoadFrames(ctx, shared.DefaultLat, shared.DefaultLon, shared.DefaultZoom)
	if err != nil {
		fatal(err)
	}
	img, err := shared.Composite(ctx, frames[0])
	if err != nil {
		fatal(err)
	}
	tmp, err := os.CreateTemp("", "radar-*.png")
	if err != nil {
		fatal(err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := png.Encode(tmp, img); err != nil {
		fatal(err)
	}
	_ = tmp.Close()
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	rw, rh := shared.FitSquareCells(w-2, h-5)
	cmd := exec.Command(viu, "-w", fmt.Sprint(rw), "-h", fmt.Sprint(rh), tmp.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("go-viu · %s · %s\n\n", frames[0].Kind, frames[0].Time.Format("15:04"))
	if err := cmd.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

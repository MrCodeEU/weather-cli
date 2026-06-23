package main

import (
	"context"
	"fmt"
	"os"

	termimg "github.com/blacktop/go-termimg"
	"golang.org/x/term"

	"weather-cli-radar-spikes/shared"
)

func main() {
	ctx := context.Background()
	frames, err := shared.LoadFrames(ctx, shared.DefaultLat, shared.DefaultLon, shared.DefaultZoom)
	if err != nil {
		fatal(err)
	}
	img, err := shared.Composite(ctx, frames[0])
	if err != nil {
		fatal(err)
	}
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	rw, rh := shared.FitSquareCells(w-2, h-5)
	out, err := termimg.New(img).Size(rw, rh).Scale(termimg.ScaleStretch).Protocol(termimg.Auto).Render()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("go-termimg · auto · %s · %s\n\n", frames[0].Kind, frames[0].Time.Format("15:04"))
	fmt.Println(out)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

package main

import (
	"context"
	"fmt"
	"os"

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
	fmt.Printf("go-halfblocks · %s · %s\n\n", frames[0].Kind, frames[0].Time.Format("15:04"))
	fmt.Println(shared.RenderHalfblocks(img, rw, rh))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

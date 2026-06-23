package main

import (
	"context"
	"fmt"

	termimg "github.com/blacktop/go-termimg"
	"github.com/rivo/tview"

	"weather-cli-radar-spikes/shared"
)

func main() {
	app := tview.NewApplication()
	view := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	view.SetBorder(true).SetTitle("go-tview-termimg")
	view.SetChangedFunc(func() { app.Draw() })
	go func() {
		ctx := context.Background()
		frames, err := shared.LoadFrames(ctx, shared.DefaultLat, shared.DefaultLon, shared.DefaultZoom)
		if err != nil {
			fmt.Fprintln(view, err)
			return
		}
		img, err := shared.Composite(ctx, frames[0])
		if err != nil {
			fmt.Fprintln(view, err)
			return
		}
		_, _, width, height := view.GetInnerRect()
		rw, rh := shared.FitSquareCells(width, height-3)
		out, err := termimg.New(img).Size(rw, rh).Scale(termimg.ScaleStretch).Protocol(termimg.Auto).Render()
		if err != nil {
			fmt.Fprintln(view, err)
			return
		}
		fmt.Fprintf(view, "frame %s %s\n\n", frames[0].Kind, frames[0].Time.Format("15:04"))
		fmt.Fprintln(view, out)
	}()
	if err := app.SetRoot(view, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

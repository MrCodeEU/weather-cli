package shared

import (
	"fmt"
	"image"
	"strings"
)

func RenderHalfblocks(img image.Image, width, height int) string {
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return ""
	}
	var b strings.Builder
	for row := 0; row < height; row++ {
		upperY := bounds.Min.Y + (row*2)*bounds.Dy()/(height*2)
		lowerY := bounds.Min.Y + (row*2+1)*bounds.Dy()/(height*2)
		for col := 0; col < width; col++ {
			x := bounds.Min.X + col*bounds.Dx()/width
			ur, ug, ub, ua := img.At(x, upperY).RGBA()
			lr, lg, lb, la := img.At(x, lowerY).RGBA()
			upperVisible := ua >= 0x2000
			lowerVisible := la >= 0x2000
			switch {
			case upperVisible && lowerVisible:
				_, _ = fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", uint8(ur>>8), uint8(ug>>8), uint8(ub>>8), uint8(lr>>8), uint8(lg>>8), uint8(lb>>8))
			case upperVisible:
				_, _ = fmt.Fprintf(&b, "\x1b[0m\x1b[38;2;%d;%d;%dm▀", uint8(ur>>8), uint8(ug>>8), uint8(ub>>8))
			case lowerVisible:
				_, _ = fmt.Fprintf(&b, "\x1b[0m\x1b[38;2;%d;%d;%dm▄", uint8(lr>>8), uint8(lg>>8), uint8(lb>>8))
			default:
				b.WriteString("\x1b[0m ")
			}
		}
		b.WriteString("\x1b[0m")
		if row != height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func FitSquareCells(width, height int) (int, int) {
	if width < 20 {
		width = 20
	}
	if height < 8 {
		height = 8
	}
	w := width
	h := w / 2
	if h > height {
		h = height
		w = h * 2
	}
	return w, h
}

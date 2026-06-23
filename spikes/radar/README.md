# Radar Renderer Spike Suite

This directory contains isolated radar-page prototypes. The goal is to compare terminal image/rendering approaches before deciding whether the main weather app should stay in Go/Bubble Tea, move to another Go TUI framework, or be rebuilt in Rust/Ratatui.

## Shared Contract

Each spike should:

- Load the same location and radar frame.
- Compose basemap + radar overlay + center marker.
- Render a radar view.
- Support terminal resize where the framework makes it practical.
- Support `+/-` zoom.
- Support previous/next frame.
- Exit cleanly with `q`.

## Spikes

- `go-halfblocks`
- `go-termimg`
- `go-tview-blocks`
- `go-tview-termimg`
- `go-chafa`
- `go-viu`
- `go-sixel`
- `rust-ratatui-image`
- `rust-ratatui-halfblocks`
- `rust-viuer`

## Status

- Go spikes compile with `go test ./...`.
- `rust-ratatui-halfblocks` and `rust-viuer` are runnable prototypes.
- `rust-ratatui-image` is now a real Ratatui `StatefulImage` prototype using the crate's protocol picker.

## Results

Record findings in `results.md` after testing in Konsole and Ghostty.

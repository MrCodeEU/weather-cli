# Radar Spike Results

| Spike | Konsole Layout | Ghostty Layout | Image Quality | Resize | Cleanup | Dependencies | Complexity | Verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| go-halfblocks | works | pending | 2/5 | works | clean | Go only | low | Baseline only; too coarse |
| go-termimg | works | pending | 4/5 | works | clean | Go module | medium | Promising |
| go-tview-blocks | not functional | pending | 0/5 | unknown | unknown | Go module | medium | Text/layout rendering failed |
| go-tview-termimg | not functional | pending | 0/5 | unknown | unknown | Go module | medium | Text/layout rendering failed |
| go-chafa | works | similar to blocks | 2/5 | works | clean | `chafa` binary | low | Not meaningfully better than built-in blocks in this TUI |
| go-viu | works | works | 4/5 | works | clean | `viu` binary | low | Good portable image result, slightly more pixelated than Sixel |
| go-sixel | works | not functional | 4.5/5 | fixed after resize patch | clean | Go module / Sixel terminal | medium | Best in Konsole, but not portable to Ghostty |
| rust-ratatui-image | implemented, retest needed | implemented, retest needed | pending | should fit area | pending | Rust crates | medium | Real StatefulImage spike now uses protocol picker |
| rust-ratatui-halfblocks | bad / no render | bad / no render | pending | pending | pending | Rust crates | medium | Less promising than Go halfblocks |
| rust-viuer | retest needed | retest needed | pending | pending | pending | Rust crates | low | User-Agent fixed after earlier 403 page image |

## Notes

- Go and Rust spikes now use the same RainViewer metadata endpoint, OSM basemap tile, radar overlay tile, and User-Agent.
- The earlier Rust `viuer` 403 screenshot was caused by the Rust shared HTTP client not setting a User-Agent for OSM tile requests.
- Sixel remains the best-looking option where the terminal supports it, but Ghostty does not appear to render it in this test.
- `ratatui-image` is the key integration spike because it reserves Ratatui buffer cells for the rendered image, unlike raw protocol dumps.

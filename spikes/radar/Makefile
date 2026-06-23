GO ?= go
CARGO ?= cargo

.PHONY: help tidy test go-halfblocks go-termimg go-tview-blocks go-tview-termimg go-chafa go-viu go-sixel rust-ratatui-image rust-ratatui-halfblocks rust-viuer

help:
	@printf "Radar spike targets:\n"
	@printf "  make go-halfblocks\n"
	@printf "  make go-termimg\n"
	@printf "  make go-tview-blocks\n"
	@printf "  make go-tview-termimg\n"
	@printf "  make go-chafa\n"
	@printf "  make go-viu\n"
	@printf "  make go-sixel\n"
	@printf "  make rust-ratatui-image\n"
	@printf "  make rust-ratatui-halfblocks\n"
	@printf "  make rust-viuer\n"

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

go-halfblocks:
	$(GO) run ./go-halfblocks

go-termimg:
	$(GO) run ./go-termimg

go-tview-blocks:
	$(GO) run ./go-tview-blocks

go-tview-termimg:
	$(GO) run ./go-tview-termimg

go-chafa:
	$(GO) run ./go-chafa

go-viu:
	$(GO) run ./go-viu

go-sixel:
	$(GO) run ./go-sixel

rust-ratatui-image:
	cd rust-ratatui-image && $(CARGO) run

rust-ratatui-halfblocks:
	cd rust-ratatui-halfblocks && $(CARGO) run

rust-viuer:
	cd rust-viuer && $(CARGO) run

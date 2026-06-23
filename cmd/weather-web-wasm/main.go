//go:build js && wasm

// Command weather-web-wasm provides browser-side metadata for the web shell.
package main

import "syscall/js"

var version = "dev"

func main() {
	done := make(chan struct{})
	js.Global().Set("weatherCLIInfo", js.FuncOf(func(this js.Value, args []js.Value) any {
		return map[string]any{
			"name":    "Weather CLI",
			"version": version,
			"runtime": "go-wasm",
		}
	}))
	<-done
}

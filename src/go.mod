module mova.local

go 1.24.4

require (
	github.com/charmbracelet/glamour v0.8.0
	github.com/lib/pq v1.10.9 // PostgreSQL
	github.com/tiktoken-go/tokenizer v0.7.0
)

require (
	github.com/charmbracelet/bubbles v1.0.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/odvcencio/gotreesitter v0.52.0
)

// PNG diagram export (mova.local/diagram, see COMMANDS.md § Diagram):
// pure-Go SVG rasterization, no cgo, added specifically for this
// feature — see diagram/png.go's header for why this pair and not
// something else.
require (
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
)

require (
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.5.0 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/goldmark v1.7.4 // indirect
	github.com/yuin/goldmark-emoji v1.0.3 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/text v0.23.0 // indirect
)

require golang.org/x/image v0.18.0 // indirect (github.com/srwiley/oksvg's math/fixed + colornames)

// Espejo temporal: en redes que bloquean golang.org (proxies
// corporativos, sandboxes de CI restringidos), "go build"/"go mod
// tidy" no pueden resolver golang.org/x/* por su redirección propia.
// Estos "replace" apuntan a los mismos módulos publicados en
// github.com/golang/* (espejo oficial) y no cambian ningún
// comportamiento. Si tu red llega a golang.org sin problema, esta
// sección es opcional y puede eliminarse sin tocar el resto del
// archivo.
replace (
	golang.org/x/image => github.com/golang/image v0.18.0
	golang.org/x/net => github.com/golang/net v0.27.0
	golang.org/x/sys => github.com/golang/sys v0.38.0
	golang.org/x/term => github.com/golang/term v0.22.0
	golang.org/x/text => github.com/golang/text v0.23.0
)

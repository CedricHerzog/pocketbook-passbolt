// A pure-Go stand-in for github.com/dennwc/inkview, which needs libinkview.
// `make preview` swaps it in to render the app's screens to PNGs on a computer.
module github.com/dennwc/inkview

go 1.27.1

require golang.org/x/image v0.47.0

require (
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

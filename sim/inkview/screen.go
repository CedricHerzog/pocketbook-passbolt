// Package ink is a stand-in for github.com/dennwc/inkview that draws into an
// in-memory image instead of the e-ink panel. It has the parts of inkview's
// API that the app uses, with the same signatures, plus SavePNG. Events,
// networking and the app loop are stubs: the preview sets the app's state
// and calls Draw itself.
package ink

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// The colors, as in inkview's graphics.go.
var (
	Black     = color.Black
	White     = color.White
	DarkGray  = color.Gray{0x55}
	LightGray = color.Gray{0xaa}
)

// screen is the panel: 1404x1872 like the InkPad 3, or PB_SIZE=WxH.
var screen = func() *image.Gray {
	w, h := 1404, 1872
	if s := os.Getenv("PB_SIZE"); s != "" {
		if _, err := fmt.Sscanf(s, "%dx%d", &w, &h); err != nil {
			panic("PB_SIZE must be WxH, like 1404x1872")
		}
	}
	return image.NewGray(image.Rect(0, 0, w, h))
}()

func ScreenSize() image.Point { return screen.Bounds().Size() }

func Screen() image.Rectangle { return image.Rectangle{Max: ScreenSize()} }

func ClearScreen() { FillArea(screen.Bounds(), White) }

// FillArea fills r with cl, reduced to the panel's 16 grays.
func FillArea(r image.Rectangle, cl color.Color) {
	draw.Draw(screen, r, image.NewUniform(panelGray(cl)), image.Point{}, draw.Src)
}

// panelGray converts cl to one of the panel's 16 levels of gray.
func panelGray(cl color.Color) color.Gray {
	g := color.GrayModel.Convert(cl).(color.Gray).Y
	return color.Gray{Y: g&0xf0 | g>>4}
}

func Repaint()                      {}
func FullUpdate()                   {}
func PartialUpdate(image.Rectangle) {}

// Paper and ink are the colors of the panel's white and black, the same as
// the blog's e-ink memes (EINK in the Makefile).
var (
	paper = color.RGBA{0xe8, 0xe5, 0xdc, 0xff}
	inkC  = color.RGBA{0x26, 0x26, 0x26, 0xff}
)

// SavePNG writes the screen to path. With eink set, white is drawn as paper
// and black as ink, and the grays in between are mixed from the two. It only
// exists in this stand-in.
func SavePNG(path string, eink bool) error { return savePNG(path, screen, eink) }

func savePNG(path string, g *image.Gray, eink bool) error {
	var img image.Image = g
	if eink {
		rgba := image.NewRGBA(g.Bounds())
		for i, v := range g.Pix {
			mix := func(a, b uint8) uint8 { return uint8((int(a)*int(v) + int(b)*(255-int(v))) / 255) }
			copy(rgba.Pix[4*i:], []uint8{mix(paper.R, inkC.R), mix(paper.G, inkC.G), mix(paper.B, inkC.B), 0xff})
		}
		img = rgba
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

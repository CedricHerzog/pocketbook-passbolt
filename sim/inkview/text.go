package ink

import (
	"embed"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The reader's fonts, as in inkview.h.
const (
	DefaultFont           = "LiberationSans"
	DefaultFontBold       = "LiberationSans-Bold"
	DefaultFontItalic     = "LiberationSans-Italic"
	DefaultFontBoldItalic = "LiberationSans-BoldItalic"
	DefaultFontMono       = "LiberationMono"
)

// fonts are copies of the reader's fonts (see fonts/README.md).
//
//go:embed fonts/*.ttf
var fonts embed.FS

type Font struct {
	face font.Face
}

// active is the font and color that SetActive chose.
var active struct {
	f  *Font
	cl color.Color
}

// OpenFont opens the font name at size pixels: <name>.ttf from $PB_FONTS if
// set, else the copy of the reader's font in fonts/.
func OpenFont(name string, size int, aa bool) *Font {
	var data []byte
	var err error
	if d := os.Getenv("PB_FONTS"); d != "" {
		data, err = os.ReadFile(filepath.Join(d, name+".ttf"))
	} else {
		data, err = fonts.ReadFile("fonts/" + name + ".ttf")
	}
	if err != nil {
		log.Printf("font %s: %v", name, err)
		return nil
	}
	ft, err := opentype.Parse(data)
	if err != nil {
		log.Printf("font %s: %v", name, err)
		return nil
	}
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		log.Printf("font %s: %v", name, err)
		return nil
	}
	return &Font{face: face}
}

func (f *Font) SetActive(cl color.Color) {
	if f != nil && f.face != nil {
		active.f, active.cl = f, cl
	}
}

func (f *Font) Close() {
	if f != nil && f.face != nil {
		f.face.Close()
		f.face = nil
	}
}

// DrawString draws s in the active font with its top-left corner at p, as
// libinkview does.
func DrawString(p image.Point, s string) {
	if active.f == nil || active.f.face == nil {
		return
	}
	face := active.f.face
	d := font.Drawer{
		Dst:  screen,
		Src:  image.NewUniform(panelGray(active.cl)),
		Face: face,
		Dot:  fixed.P(p.X, p.Y+face.Metrics().Ascent.Ceil()),
	}
	d.DrawString(s)
}

func StringWidth(s string) int {
	if active.f == nil || active.f.face == nil {
		return 0
	}
	return font.MeasureString(active.f.face, s).Ceil()
}

func CharWidth(c rune) int { return StringWidth(string(c)) }

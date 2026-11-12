package ink

import (
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

// shot is a copy of the screen, and its caption on the overview.
type shot struct {
	name string
	img  *image.Gray
}

var shots []shot

// Snapshot keeps a copy of the screen, named name, for SaveOverview. It only
// exists in this stand-in.
func Snapshot(name string) {
	img := image.NewGray(screen.Bounds())
	copy(img.Pix, screen.Pix)
	shots = append(shots, shot{name, img})
}

// SaveOverview writes the snapshots side by side to path, cols per row, each
// framed with its name above it. Each screen is shrunk scale times: with 1,
// the screens keep their full resolution and can be cropped out one by one.
// eink is as for SavePNG. It only exists in this stand-in.
func SaveOverview(path string, eink bool, cols, scale int) error {
	scale = max(1, scale)
	gap, frameW, capSize := 144/scale, max(1, 6/scale), 108/scale
	size := ScreenSize().Div(scale)
	f := OpenFont(DefaultFontBold, capSize, true)
	defer f.Close()
	capH := capSize * 2

	rows := (len(shots) + cols - 1) / cols
	cell := image.Pt(size.X+gap, size.Y+capH+gap)
	out := image.NewGray(image.Rect(0, 0, gap+cols*cell.X, gap+rows*cell.Y))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.Gray{0xd8}), image.Point{}, draw.Src)

	// Draw captions with the stand-in's text functions, on out.
	saved := screen
	screen = out
	defer func() { screen = saved }()

	for i, s := range shots {
		p := image.Pt(gap+i%cols*cell.X, gap+i/cols*cell.Y)
		f.SetActive(Black)
		DrawString(p, s.name)
		r := image.Rectangle{Min: p.Add(image.Pt(0, capH)), Max: p.Add(image.Pt(size.X, capH+size.Y))}
		FillArea(r.Inset(-frameW), Black)
		if scale == 1 {
			draw.Draw(out, r, s.img, image.Point{}, draw.Src)
		} else {
			xdraw.CatmullRom.Scale(out, r, s.img, s.img.Bounds(), xdraw.Src, nil)
		}
	}
	return savePNG(path, out, eink)
}

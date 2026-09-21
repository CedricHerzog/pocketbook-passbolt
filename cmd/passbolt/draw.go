package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"
	"unicode"

	ink "github.com/dennwc/inkview"

	"github.com/CedricHerzog/pocketbook-passbolt/internal/passbolt"
)

// unit is the screen's shorter side, set in Init. Sizes in this file are for
// the InkPad 3 (1404x1872) and px scales them to the current screen.
var unit = 1404

func px(n int) int { return n * unit / 1404 }

// font is an open font and its size in pixels.
type font struct {
	*ink.Font
	size int
}

// lineHeight is the distance between two lines of text in f.
func (f font) lineHeight() int { return f.size * 5 / 4 }

// fonts are the app's fonts, opened in Init and closed in Close.
type fonts struct {
	small font // secondary text, labels, hints
	body  font // progress lines, values, page number
	bold  font // resource names, buttons
	title font // resource name on the detail screen
	mono  font // password
}

func openFonts() fonts {
	open := func(name string, size int) font {
		size = px(size)
		f := ink.OpenFont(name, size, true)
		if f == nil {
			log.Printf("cannot open font %s", name)
		}
		return font{f, size}
	}
	return fonts{
		small: open(ink.DefaultFont, 32),
		body:  open(ink.DefaultFont, 40),
		bold:  open(ink.DefaultFontBold, 44),
		title: open(ink.DefaultFontBold, 56),
		mono:  open(ink.DefaultFontMono, 72),
	}
}

func (fs fonts) close() {
	for _, f := range []font{fs.small, fs.body, fs.bold, fs.title, fs.mono} {
		f.Close()
	}
}

// drawLoading draws the loading screen, or the error screen once a.err is
// set: the large logo, the progress lines and a Cancel or Exit button. It
// returns the progress area, the only part that changes while loading.
func (a *app) drawLoading() image.Rectangle {
	scr := ink.Screen()
	w, h := scr.Dx(), scr.Dy()
	lg := logoLarge
	if lg.size.X > w-2*px(96) {
		lg = logoSmall
	}
	top := h / 5
	lg.draw(image.Pt((w-lg.size.X)/2, top))

	btn := image.Rect(w/2-px(150), h-px(250), w/2+px(150), h-px(150))
	area := image.Rect(px(96), top+lg.size.Y+px(96), w-px(96), btn.Min.Y-px(48))

	type line struct {
		f font
		c color.Color
		s string
	}
	var lines []line
	add := func(f font, c color.Color, s string) {
		for _, l := range wrapWords(f, s, area.Dx()) {
			lines = append(lines, line{f, c, l})
		}
	}
	for _, s := range a.lines {
		add(a.fonts.body, ink.Black, s)
	}
	if a.status != "" {
		add(a.fonts.body, ink.DarkGray, a.status+"…")
	}
	label := "Cancel"
	if a.err != nil {
		lines = append(lines, line{a.fonts.body, ink.Black, ""})
		add(a.fonts.bold, ink.Black, "Error")
		add(a.fonts.body, ink.Black, clean(a.err.Error()))
		label = "Exit"
	}

	// Show the newest lines that fit.
	first, used := len(lines), 0
	for first > 0 && used+lines[first-1].f.lineHeight() <= area.Dy() {
		first--
		used += lines[first].f.lineHeight()
	}
	y := area.Min.Y
	for _, l := range lines[first:] {
		text(l.f, l.c, image.Pt(area.Min.X, y), l.s)
		y += l.f.lineHeight()
	}

	button(a.fonts.bold, btn, label, true)
	a.tap(btn, action{kind: actExit})
	return area
}

// drawList draws a page of the resource list between a header and a footer.
// It returns the list and the footer, the part that changes on a page turn.
func (a *app) drawList() image.Rectangle {
	scr := ink.Screen()
	w, h := scr.Dx(), scr.Dy()
	m, bar := px(48), px(168)

	// Header: logo, resource count and Exit.
	logoSmall.draw(image.Pt(m, (bar-logoSmall.size.Y)/2))
	exit := image.Rect(w-m-px(200), (bar-px(100))/2, w-m, (bar+px(100))/2)
	button(a.fonts.bold, exit, "Exit", true)
	a.tap(image.Rect(exit.Min.X-m, 0, w, bar), action{kind: actExit})
	count := fmt.Sprintf("%d resources", len(a.list))
	if len(a.list) == 1 {
		count = "1 resource"
	}
	if a.skipped > 0 {
		count += fmt.Sprintf(" · %d unsupported", a.skipped)
	}
	right := exit.Min.X - px(32)
	count = ellipsize(a.fonts.small, count, right-(m+logoSmall.size.X+px(32)))
	text(a.fonts.small, ink.DarkGray, image.Pt(right-width(a.fonts.small, count), (bar-a.fonts.small.size)/2), count)
	ink.FillArea(image.Rect(0, bar-px(3), w, bar), ink.Black)

	// Rows.
	list := image.Rect(0, bar, w, h-bar)
	a.rows = max(1, list.Dy()/px(150))
	rowH := list.Dy() / a.rows
	pages := max(1, (len(a.list)+a.rows-1)/a.rows)
	a.page = min(a.page, pages-1)
	if len(a.list) == 0 {
		s := "No resources"
		text(a.fonts.body, ink.DarkGray, image.Pt((w-width(a.fonts.body, s))/2, list.Min.Y+list.Dy()/3), s)
	}
	first := a.page * a.rows
	for i := first; i < min(first+a.rows, len(a.list)); i++ {
		top := list.Min.Y + (i-first)*rowH
		row := image.Rect(0, top, w, top+rowH)
		if i > first {
			ink.FillArea(image.Rect(m, top-px(1), w-m, top+px(1)), ink.LightGray)
		}
		a.drawRow(a.list[i], row)
		a.tap(row, action{kind: actOpen, i: i})
	}

	// Footer: page buttons and page number.
	fy := h - bar
	ink.FillArea(image.Rect(0, fy, w, fy+px(3)), ink.Black)
	by := fy + (bar-px(100))/2
	button(a.fonts.bold, image.Rect(m, by, m+px(300), by+px(100)), "‹ Prev", a.page > 0)
	button(a.fonts.bold, image.Rect(w-m-px(300), by, w-m, by+px(100)), "Next ›", a.page < pages-1)
	a.tap(image.Rect(0, fy, w/3, h), action{kind: actPrev})
	a.tap(image.Rect(w-w/3, fy, w, h), action{kind: actNext})
	num := fmt.Sprintf("Page %d of %d", a.page+1, pages)
	text(a.fonts.body, ink.Black, image.Pt((w-width(a.fonts.body, num))/2, fy+(bar-a.fonts.body.size)/2), num)
	return image.Rect(0, bar, w, h)
}

// drawRow draws a resource's name and, below it, its username and URI.
func (a *app) drawRow(res passbolt.Resource, r image.Rectangle) {
	m, gap := px(48), px(14)
	var sub []string
	for _, s := range []string{res.Username, res.URI} {
		if s = strings.TrimSpace(clean(s)); s != "" {
			sub = append(sub, s)
		}
	}
	y := r.Min.Y + (r.Dy()-a.fonts.bold.size-gap-a.fonts.small.size)/2
	text(a.fonts.bold, ink.Black, image.Pt(m, y), ellipsize(a.fonts.bold, name(res), r.Dx()-2*m))
	y += a.fonts.bold.size + gap
	text(a.fonts.small, ink.DarkGray, image.Pt(m, y), ellipsize(a.fonts.small, strings.Join(sub, " · "), r.Dx()-2*m))
}

// drawDetail draws the open resource: its name, username, URI and, in a box,
// its password. It returns the area the box can grow into, the part that
// changes once the password is decrypted.
func (a *app) drawDetail() image.Rectangle {
	scr := ink.Screen()
	w, h := scr.Dx(), scr.Dy()
	m, bar := px(48), px(168)
	res := a.list[a.sel]

	// Header: Back and logo.
	button(a.fonts.bold, image.Rect(m, (bar-px(100))/2, m+px(300), (bar+px(100))/2), "‹ Back", true)
	a.tap(image.Rect(0, 0, w/2, bar), action{kind: actBack})
	logoSmall.draw(image.Pt(w-m-logoSmall.size.X, (bar-logoSmall.size.Y)/2))
	ink.FillArea(image.Rect(0, bar-px(3), w, bar), ink.Black)

	y := bar + px(48)
	text(a.fonts.title, ink.Black, image.Pt(m, y), ellipsize(a.fonts.title, name(res), w-2*m))
	y += a.fonts.title.size + px(56)
	label := func(s string) {
		text(a.fonts.small, ink.DarkGray, image.Pt(m, y), s)
		y += a.fonts.small.size + px(12)
	}
	for _, f := range []struct{ label, value string }{{"Username", res.Username}, {"URI", res.URI}} {
		label(f.label)
		text(a.fonts.body, ink.Black, image.Pt(m, y), ellipsize(a.fonts.body, clean(f.value), w-2*m))
		y += a.fonts.body.size + px(40)
	}
	label("Password")

	hint := "Press any button or tap Back to return"
	hy := h - px(96) - a.fonts.small.size
	text(a.fonts.small, ink.DarkGray, image.Pt((w-width(a.fonts.small, hint))/2, hy), hint)

	// Password box.
	area := image.Rect(m, y, w-m, hy-px(48))
	pad := px(32)
	f, c := a.fonts.body, color.Color(ink.DarkGray)
	var lines []string
	switch {
	case a.decrypting:
		lines = []string{"Decrypting…"}
	case a.revealErr != nil:
		c = ink.Black
		lines = wrapWords(f, "Could not decrypt: "+clean(a.revealErr.Error()), area.Dx()-2*pad)
	case a.password == "":
		lines = []string{"(no password)"}
	default:
		f, c = a.fonts.mono, ink.Black
		lines = wrapChars(f, clean(a.password), area.Dx()-2*pad)
	}
	if n := max(1, (area.Dy()-2*pad)/f.lineHeight()); len(lines) > n {
		lines = lines[:n]
		last := []rune(lines[n-1])
		lines[n-1] = string(last[:len(last)-1]) + "…"
	}
	box := image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+2*pad+len(lines)*f.lineHeight())
	frame(box, px(3), ink.Black)
	for i, l := range lines {
		text(f, c, image.Pt(box.Min.X+pad, box.Min.Y+pad+i*f.lineHeight()), l)
	}
	return area
}

// name returns the name to show for res.
func name(res passbolt.Resource) string {
	if s := strings.TrimSpace(clean(res.Name)); s != "" {
		return s
	}
	return "(no name)"
}

// text draws s in font f and color c, with its top-left corner at p.
func text(f font, c color.Color, p image.Point, s string) {
	f.SetActive(c)
	ink.DrawString(p, s)
}

// width returns the width of s in font f.
func width(f font, s string) int {
	f.SetActive(ink.Black)
	return ink.StringWidth(s)
}

// button draws a framed button with a centered label. A disabled button is
// light gray.
func button(f font, r image.Rectangle, label string, enabled bool) {
	var c color.Color = ink.Black
	if !enabled {
		c = ink.LightGray
	}
	frame(r, px(3), c)
	text(f, c, image.Pt(r.Min.X+(r.Dx()-width(f, label))/2, r.Min.Y+(r.Dy()-f.size)/2), label)
}

// frame draws a border t pixels thick inside r.
func frame(r image.Rectangle, t int, c color.Color) {
	ink.FillArea(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+t), c)
	ink.FillArea(image.Rect(r.Min.X, r.Max.Y-t, r.Max.X, r.Max.Y), c)
	ink.FillArea(image.Rect(r.Min.X, r.Min.Y, r.Min.X+t, r.Max.Y), c)
	ink.FillArea(image.Rect(r.Max.X-t, r.Min.Y, r.Max.X, r.Max.Y), c)
}

// ellipsize shortens s to at most w pixels in font f, ending it with "…".
func ellipsize(f font, s string, w int) string {
	if width(f, s) <= w {
		return s
	}
	r := []rune(s)
	lo, hi := 0, len(r) // the longest prefix that fits is r[:lo]
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if width(f, string(r[:mid])+"…") <= w {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return string(r[:lo]) + "…"
}

// wrapWords splits s into lines of at most w pixels in font f, breaking at
// spaces. A word longer than a line is broken between characters.
func wrapWords(f font, s string, w int) []string {
	var lines []string
	line := ""
	for word := range strings.FieldsSeq(s) {
		next := word
		if line != "" {
			next = line + " " + word
		}
		if width(f, next) <= w {
			line = next
			continue
		}
		if line != "" {
			lines = append(lines, line)
		}
		parts := wrapChars(f, word, w)
		lines = append(lines, parts[:len(parts)-1]...)
		line = parts[len(parts)-1]
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// wrapChars splits s into lines of at most w pixels in font f, breaking
// between any two characters. Passwords are wrapped this way so that no space
// is lost at a line break.
func wrapChars(f font, s string, w int) []string {
	var lines []string
	var line []rune
	lw := 0
	for _, r := range s {
		// Not CharWidth, which takes a 16-bit character.
		cw := width(f, string(r))
		if lw+cw > w && len(line) > 0 {
			lines = append(lines, string(line))
			line, lw = nil, 0
		}
		line = append(line, r)
		lw += cw
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}

// clean replaces control characters, which the fonts cannot draw, with U+FFFD.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '�'
		}
		return r
	}, s)
}

//go:build sim

// The preview build renders the app's screens on a computer, with made-up
// data, using the stand-in inkview in sim/inkview: all of them on one
// overview image, and with -each one image per screen. Run it with
// `make preview`.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	ink "github.com/dennwc/inkview"

	"github.com/CedricHerzog/pocketbook-passbolt/internal/passbolt"
)

// scene is one screen state to render.
type scene struct {
	name string
	set  func(a *app)
}

var progress = []string{
	"Wi-Fi connected in 1.284s",
	"Reading config from " + configDir,
	"Private key unlocked in 2.917s",
	"Server verified in 412ms",
	"Logged in to https://passbolt.example.com in 3.106s",
}

var sample = []passbolt.Resource{
	{Name: "Bank", Username: "jane.doe", URI: "https://bank.example.com/login"},
	{Name: "Email", Username: "jane@example.com", URI: "https://mail.example.com"},
	{Name: "GitHub", Username: "janedoe", URI: "https://github.com"},
	{Name: "Home Wi-Fi", Username: "", URI: ""},
	{Name: "Hosting control panel for the family website and its mailboxes", Username: "admin", URI: "https://panel.hosting.example.net/customer/dashboard?account=42"},
	{Name: "Insurance", Username: "J-0042-1187", URI: "https://my.insurance.example"},
	{Name: "Library card", Username: "1102 3344 5566", URI: ""},
	{Name: "Mastodon", Username: "@jane@social.example", URI: "https://social.example"},
	{Name: "NAS", Username: "jane", URI: "https://192.168.1.20:5001"},
	{Name: "Phone carrier", Username: "+33 6 12 34 56 78", URI: "https://account.carrier.example"},
	{Name: "Router", Username: "admin", URI: "http://192.168.1.1"},
	{Name: "Tax portal", Username: "1 23 45 678 901 234", URI: "https://tax.example.gov"},
	{Name: "Train tickets", Username: "jane.doe@example.com", URI: "https://tickets.rail.example"},
	{Name: "", Username: "unnamed", URI: "https://example.org"},
	{Name: "VPN", Username: "jane", URI: "vpn.example.com"},
}

func listed(a *app) {
	a.lines = append(progress, fmt.Sprintf("%d resources listed in 1.532s", len(sample)), "2 resources of an unsupported type skipped")
	a.list, a.skipped = sample, 2
	a.show(screenList)
}

func detail(i int, f func(a *app)) func(a *app) {
	return func(a *app) {
		listed(a)
		a.sel = i
		a.show(screenDetail)
		f(a)
	}
}

var scenes = []scene{
	{"loading", func(a *app) {
		a.lines = progress[:3]
		a.status = "Verifying server"
	}},
	{"error", func(a *app) {
		a.lines = progress[:4]
		a.err = errors.New("logging in: api error: the user's account is disabled or the server is unreachable (Post \"https://passbolt.example.com/auth/jwt/login.json\": dial tcp: i/o timeout)")
		a.show(screenError)
	}},
	{"list", listed},
	{"list-last-page", func(a *app) {
		listed(a)
		a.page = 1 << 20 // drawList clamps it to the last page
	}},
	{"list-empty", func(a *app) {
		a.lines = append(progress, "0 resources listed in 204ms")
		a.show(screenList)
	}},
	{"detail-decrypting", detail(2, func(a *app) { a.decrypting = true })},
	{"detail", detail(2, func(a *app) { a.password = "correct-horse-battery" })},
	{"detail-long-password", detail(4, func(a *app) {
		a.password = "Xq7#vP2!mZ9@rL4$kW8^nB3&hT6*fJ1(dS5)gY0_cE-uA+iO=oU~"
	})},
	{"detail-no-password", detail(3, func(a *app) {})},
	{"detail-error", detail(0, func(a *app) {
		a.revealErr = errors.New("decrypting secret: gopenpgp: error in reading message: openpgp: incorrect key")
	})},
}

func main() {
	out := flag.String("out", "build/preview", "folder for the PNGs")
	eink := flag.Bool("eink", false, "also write the -eink versions, in the panel's paper and ink colors")
	each := flag.Bool("each", false, "also write each screen to its own <scene>.png")
	only := flag.String("only", "", "render only this scene")
	cols := flag.Int("cols", 5, "screens per row on the overview")
	scale := flag.Int("scale", 1, "shrink the screens on the overview this many times (1: full resolution)")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	size := ink.ScreenSize()
	unit = min(size.X, size.Y)
	fs := openFonts()
	defer fs.close()

	found := false
	for _, s := range scenes {
		if *only != "" && s.name != *only {
			continue
		}
		found = true
		a := &app{fonts: fs}
		s.set(a)
		a.Draw()
		ink.Snapshot(s.name)
		if *each {
			save(filepath.Join(*out, s.name), *eink, ink.SavePNG)
		}
	}
	if !found {
		log.Fatalf("no scene %q", *only)
	}
	save(filepath.Join(*out, "overview"), *eink, func(path string, eink bool) error {
		return ink.SaveOverview(path, eink, *cols, *scale)
	})
}

// save writes base.png, and base-eink.png if eink is set, with write.
func save(base string, eink bool, write func(path string, eink bool) error) {
	for _, e := range []bool{false, true} {
		if e && !eink {
			continue
		}
		path := base + ".png"
		if e {
			path = base + "-eink.png"
		}
		if err := write(path, e); err != nil {
			log.Fatal(err)
		}
		fmt.Println(path)
	}
}

// Command hello is a Hello World app for PocketBook readers. It shows the
// device model, firmware and Wi-Fi address (the IP to attach gdb to), counts
// taps, and exits on any button.
package main

import (
	"image"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"

	ink "github.com/dennwc/inkview"
)

// libinkview's event loop and its callbacks must run on the process's main thread.
func init() { runtime.LockOSThread() }

type app struct {
	title, body *ink.Font
	taps        int
	drawn       bool
}

func (a *app) Init() error {
	a.title = ink.OpenFont(ink.DefaultFontBold, 72, true)
	a.body = ink.OpenFont(ink.DefaultFont, 36, true)
	log.Printf("init: model=%q firmware=%q screen=%v wifi=%q",
		ink.DeviceModel(), ink.SoftwareVersion(), ink.ScreenSize(), wifiAddr())
	return nil
}

func (a *app) Close() error {
	log.Printf("exit after %d taps", a.taps)
	a.title.Close()
	a.body.Close()
	return nil
}

func (a *app) Draw() {
	ink.ClearScreen()
	width := ink.ScreenSize().X
	y := 250
	center := func(s string, lineHeight int) {
		ink.DrawString(image.Pt((width-ink.StringWidth(s))/2, y), s)
		y += lineHeight
	}

	a.title.SetActive(ink.Black)
	center("Hello, World!", 200)

	a.body.SetActive(ink.Black)
	center(ink.DeviceModel()+" · firmware "+ink.SoftwareVersion(), 60)
	wifi := wifiAddr()
	if wifi == "" {
		wifi = "off"
	}
	center("Wi-Fi: "+wifi, 60)
	center("Taps: "+strconv.Itoa(a.taps), 160)
	center("Tap to count · any button to exit", 60)
	center("Long-press to test crash logging", 60)

	// The first draw does a full refresh; later redraws avoid the e-ink flash.
	if a.drawn {
		ink.SoftUpdate()
	} else {
		ink.FullUpdate()
		a.drawn = true
	}
}

func (a *app) Key(e ink.KeyEvent) bool {
	if e.State == ink.KeyStateUp {
		log.Printf("key %v released, exiting", e.Key)
		ink.Exit()
	}
	return true
}

func (a *app) Pointer(e ink.PointerEvent) bool {
	switch e.State {
	case ink.PointerUp:
		a.taps++
		log.Printf("tap %d at %v", a.taps, e.Point)
		ink.Repaint()
		return true
	case ink.PointerLong:
		// Crash self-test: a deliberate panic, to check that the traceback
		// ends up in the log file.
		panic("crash self-test (long press)")
	}
	return false
}

func (a *app) Touch(ink.TouchEvent) bool        { return false }
func (a *app) Orientation(ink.Orientation) bool { return false }

// wifiAddr returns the reader's IPv4 address, or "" when Wi-Fi is off.
func wifiAddr() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipn, ok := addr.(*net.IPNet); ok && ipn.IP.To4() != nil {
				return ipn.IP.String()
			}
		}
	}
	return ""
}

func main() {
	if err := setupLogging(); err != nil {
		log.Printf("logging to %s disabled: %v", logPath, err)
	}
	log.Printf("hello starting (%s, pid %d)", runtime.Version(), os.Getpid())
	if err := ink.Run(&app{}); err != nil {
		log.Printf("run: %v", err)
		os.Exit(1)
	}
}

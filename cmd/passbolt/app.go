package main

import (
	"context"
	"image"
	"log"
	"sync"
	"time"

	ink "github.com/dennwc/inkview"
	"github.com/passbolt/go-passbolt/api"

	"github.com/CedricHerzog/pocketbook-passbolt/internal/passbolt"
)

// screen is what the app shows.
type screen int

const (
	screenLoading screen = iota // logo and progress lines
	screenError                 // the error that stopped loading; any input exits
	screenList                  // a page of resources
	screenDetail                // one resource, and its password once decrypted
)

// actionKind is what a tap or a button does.
type actionKind int

const (
	actNone actionKind = iota
	actExit
	actPrev // previous page
	actNext // next page
	actOpen // show a resource
	actBack // back to the list
)

type action struct {
	kind actionKind
	i    int // the resource to open, for actOpen
}

// target is an area of the screen that triggers act when tapped.
type target struct {
	r   image.Rectangle
	act action
}

// keys maps the buttons the app uses to what they do on the list screen.
// Other screens give them their own meaning, see Key.
var keys = map[ink.Key]actionKind{
	ink.KeyNext: actNext, ink.KeyNext2: actNext, ink.KeyRight: actNext, ink.KeyDown: actNext, ink.KeyPlus: actNext,
	ink.KeyPrev: actPrev, ink.KeyPrev2: actPrev, ink.KeyLeft: actPrev, ink.KeyUp: actPrev, ink.KeyMinus: actPrev,
	ink.KeyBack: actExit, ink.KeyMenu: actExit,
	ink.KeyOk: actNone,
}

// app is the ink.App. It loads the resources in a goroutine and shows them.
// libinkview calls Init, Close, Draw and the event handlers on the main
// thread; other goroutines only change the state and call ink.Repaint.
type app struct {
	mu         sync.Mutex // guards the fields up to closing
	screen     screen
	lines      []string    // finished progress lines, also logged
	status     string      // the step in progress, "" if none
	err        error       // what stopped loading
	client     *api.Client // logged in; Close logs it out
	list       []passbolt.Resource
	skipped    int
	page       int // page of the list shown
	rows       int // rows per page, set by Draw
	sel        int // index in list of the resource on screenDetail
	decrypting bool
	password   string // list[sel]'s password; dropped when leaving screenDetail
	revealErr  error
	targets    []target // tap targets of the last Draw
	full       bool     // the next Draw does a full, flashing refresh
	pending    bool     // a Repaint is queued
	closing    bool

	// Main thread only.
	cancel  context.CancelFunc // stops load
	wg      sync.WaitGroup     // load and reveal goroutines
	stopNet func()
	fonts   fonts
	down    image.Point // where the last tap started
}

func (a *app) Init() error {
	size := ink.ScreenSize()
	unit = min(size.X, size.Y)
	a.fonts = openFonts()
	a.status = "Connecting to Wi-Fi"
	a.Draw()

	start := time.Now()
	stop, err := ink.KeepNetwork()
	if err != nil {
		a.say("Wi-Fi not connected: %v", err)
	} else {
		a.stopNet = stop
		a.say("Wi-Fi connected in %v", time.Since(start).Round(time.Millisecond))
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.wg.Add(1)
	go a.load(ctx)
	return nil
}

// Close stops the goroutines, then logs out. It returns the error that
// stopped loading, if any, so that ink.Run returns it.
func (a *app) Close() error {
	a.mu.Lock()
	a.closing = true
	a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()

	if a.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := a.client.Logout(ctx); err != nil {
			log.Printf("logging out: %v", err)
		} else {
			log.Print("logged out")
		}
		cancel()
	}
	if a.stopNet != nil {
		a.stopNet()
	}
	a.fonts.close()
	return a.err
}

// Draw redraws the whole screen. It refreshes the panel fully when the
// screen changed, or when libinkview asked for the redraw, and otherwise only
// the part that changed.
func (a *app) Draw() {
	a.mu.Lock()
	defer a.mu.Unlock()
	full := a.full || !a.pending
	a.full, a.pending = false, false
	a.targets = a.targets[:0]

	ink.ClearScreen()
	var changed image.Rectangle
	switch a.screen {
	case screenLoading, screenError:
		changed = a.drawLoading()
	case screenList:
		changed = a.drawList()
	case screenDetail:
		changed = a.drawDetail()
	}
	if full {
		ink.FullUpdate()
	} else {
		ink.PartialUpdate(changed)
	}
}

func (a *app) Key(e ink.KeyEvent) bool {
	kind, ok := keys[e.Key]
	if !ok {
		if e.State == ink.KeyStatePress {
			log.Printf("unhandled key %#x", int(e.Key))
		}
		return false
	}
	a.mu.Lock()
	scr := a.screen
	a.mu.Unlock()

	// Act on presses, and on repeats of the page keys on the list. The
	// other events of the keys the app uses are swallowed.
	repeat := e.State == ink.KeyStateRepeat && scr == screenList && (kind == actPrev || kind == actNext)
	if e.State != ink.KeyStatePress && !repeat {
		return true
	}
	switch scr {
	case screenLoading:
		// Prev also cancels, as in inkview's RunCLI.
		if kind != actExit && e.Key != ink.KeyPrev {
			return true
		}
		kind = actExit
	case screenError:
		kind = actExit
	case screenDetail:
		kind = actBack
	}
	a.do(action{kind: kind})
	return true
}

func (a *app) Pointer(e ink.PointerEvent) bool {
	switch e.State {
	case ink.PointerDown:
		a.down = e.Point
	case ink.PointerUp:
		a.mu.Lock()
		act := a.tapAction(a.down, e.Point)
		a.mu.Unlock()
		a.do(act)
	}
	return true
}

// tapAction returns what a tap from down to up does: a swipe across the list
// turns the page, otherwise the tap must start and end on the same target.
// The caller holds a.mu.
func (a *app) tapAction(down, up image.Point) action {
	switch a.screen {
	case screenError:
		return action{kind: actExit}
	case screenList:
		d := up.Sub(down)
		if abs(d.X) > ink.ScreenSize().X/5 && abs(d.X) > 2*abs(d.Y) {
			if d.X < 0 {
				return action{kind: actNext}
			}
			return action{kind: actPrev}
		}
	}
	for _, t := range a.targets {
		if down.In(t.r) && up.In(t.r) {
			return t.act
		}
	}
	return action{}
}

func (*app) Touch(ink.TouchEvent) bool        { return false }
func (*app) Orientation(ink.Orientation) bool { return false }

// tap adds a tap target. Draw calls it with a.mu held.
func (a *app) tap(r image.Rectangle, act action) {
	a.targets = append(a.targets, target{r, act})
}

// show switches to screen s. Until the next Draw rebuilds the tap targets,
// taps do nothing. The caller holds a.mu.
func (a *app) show(s screen) {
	a.screen, a.targets = s, nil
}

// do performs act. It runs on the main thread without a.mu held, because
// ink.Exit may call Close right away.
func (a *app) do(act action) {
	switch act.kind {
	case actExit:
		ink.Exit()
	case actPrev, actNext:
		a.update(false, func() bool { return a.turn(act.kind == actNext) })
	case actOpen:
		a.open(act.i)
	case actBack:
		// A full refresh, so that no ghost of the password stays on the panel.
		a.update(true, func() bool {
			if a.screen != screenDetail {
				return false
			}
			a.password, a.revealErr, a.decrypting = "", nil, false
			a.show(screenList)
			return true
		})
	}
}

// turn moves the list one page forward or back, and reports whether the page
// changed. The caller holds a.mu.
func (a *app) turn(forward bool) bool {
	switch {
	case a.screen != screenList || a.rows == 0:
		return false
	case forward && (a.page+1)*a.rows < len(a.list):
		a.page++
	case !forward && a.page > 0:
		a.page--
	default:
		return false
	}
	return true
}

// open shows resource i and decrypts its password in the background.
func (a *app) open(i int) {
	var res passbolt.Resource
	var c *api.Client
	a.update(true, func() bool {
		if a.screen != screenList || i >= len(a.list) {
			return false
		}
		a.sel, a.password, a.revealErr, a.decrypting = i, "", nil, true
		a.show(screenDetail)
		res, c = a.list[i], a.client
		return true
	})
	if c == nil {
		return
	}
	a.wg.Add(1)
	go a.reveal(i, res, c)
}

// update runs f with a.mu held. If f reports a change, update queues a
// redraw: a full refresh if full is set, else a partial one.
func (a *app) update(full bool, f func() bool) {
	a.mu.Lock()
	repaint := false
	if f() {
		a.full = a.full || full
		repaint = !a.pending && !a.closing
		a.pending = true
	}
	a.mu.Unlock()
	if repaint {
		ink.Repaint()
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

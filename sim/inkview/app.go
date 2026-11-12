package ink

import (
	"errors"
	"image"
)

// The event types and constants, with libinkview's values.

type KeyEvent struct {
	Key   Key
	State KeyState
}

type PointerEvent struct {
	image.Point
	State PointerState
}

type TouchEvent struct {
	image.Point
	State TouchState
}

type KeyState int

const (
	KeyStateDown    KeyState = 25
	KeyStatePress   KeyState = 25
	KeyStateUp      KeyState = 26
	KeyStateRelease KeyState = 26
	KeyStateRepeat  KeyState = 28
)

type PointerState int

const (
	PointerUp   PointerState = 29
	PointerDown PointerState = 30
)

type TouchState int

type Key int

const (
	KeyBack  Key = 0x1b
	KeyOk    Key = 0x0a
	KeyUp    Key = 0x11
	KeyDown  Key = 0x12
	KeyLeft  Key = 0x13
	KeyRight Key = 0x14
	KeyMinus Key = 0x15
	KeyPlus  Key = 0x16
	KeyMenu  Key = 0x17
	KeyPrev  Key = 0x18
	KeyNext  Key = 0x19
	KeyPrev2 Key = 0x1c
	KeyNext2 Key = 0x1d
)

type Orientation int

type App interface {
	Init() error
	Close() error
	Draw()
	Key(e KeyEvent) bool
	Pointer(e PointerEvent) bool
	Touch(e TouchEvent) bool
	Orientation(o Orientation) bool
}

// Run is not supported: the preview drives the app itself.
func Run(app App) error { return errors.New("ink.Run is not available in the preview") }

func Exit() {}

func KeepNetwork() (func(), error) { return func() {}, nil }

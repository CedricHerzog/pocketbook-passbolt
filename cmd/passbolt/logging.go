package main

import (
	"log"
	"os"
	"runtime/debug"
	"syscall"
)

// logPath is on the reader's internal storage, next to the app. Read it with
// `make log` after connecting the reader over USB.
const logPath = "/mnt/ext1/applications/passbolt.log"

// maxLogSize is the size above which the log is truncated at startup.
const maxLogSize = 1 << 20

// setupLogging sends the standard logger, stdout and stderr to logPath.
// Redirecting the file descriptors also captures Go panic tracebacks and
// anything libinkview prints.
func setupLogging() error {
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if fi, err := os.Stat(logPath); err == nil && fi.Size() > maxLogSize {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(logPath, flags, 0o644)
	if err != nil {
		return err
	}
	for _, fd := range []int{1, 2} {
		if err := syscall.Dup2(int(f.Fd()), fd); err != nil {
			return err
		}
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	// Include every goroutine in crash tracebacks, not just the failing one.
	debug.SetTraceback("all")
	return nil
}

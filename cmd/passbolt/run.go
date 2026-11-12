//go:build !sim

package main

import (
	"log"
	"os"
	"runtime"

	ink "github.com/dennwc/inkview"
)

// main runs the app on the reader. The preview build (preview.go) has its own.
func main() {
	if err := setupLogging(); err != nil {
		log.Printf("logging to %s disabled: %v", logPath, err)
	}
	log.Printf("passbolt starting (%s, pid %d)", runtime.Version(), os.Getpid())

	// Run returns the error that stopped loading, already logged and shown.
	if err := ink.Run(&app{}); err != nil {
		os.Exit(1)
	}
}

// Command passbolt is a Passbolt client for PocketBook readers. It reads its
// configuration from configDir, verifies the server, logs in and shows the
// user's resources as a paged list. Tapping a resource decrypts its password
// and shows it.
package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	ink "github.com/dennwc/inkview"
)

// configDir holds config.json and private.asc, copied over USB, and
// verify.json, which the app writes on its first run.
const configDir = "/mnt/ext1/applications/passbolt"

// libinkview's event loop and its callbacks must run on the process's main thread.
func init() { runtime.LockOSThread() }

// httpClient does not verify the server's TLS certificate, so the reader needs
// no CA roots and self-signed servers work. Someone on the network could
// intercept the session; secrets and v5 metadata stay end-to-end encrypted.
func httpClient() *http.Client {
	return &http.Client{
		Timeout: time.Minute,
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

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

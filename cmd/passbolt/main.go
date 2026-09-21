// Command passbolt is a Passbolt client for PocketBook readers. It reads its
// configuration from configDir, verifies the server, logs in and lists the
// user's resources on screen. Any button or tap exits.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	ink "github.com/dennwc/inkview"
	"github.com/passbolt/go-passbolt/api"

	"github.com/CedricHerzog/pocketbook-passbolt/internal/passbolt"
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

// run writes its progress and the resource list to w, which RunCLI shows on
// screen. ctx is cancelled when the app exits.
func run(ctx context.Context, w io.Writer) error {
	// say shows a progress line and logs it. Resource details go only to the
	// screen, never to the log file.
	say := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintln(w, msg)
		log.Print(msg)
	}
	// Each step reports how long it took, measured from start.
	var start time.Time
	elapsed := func() time.Duration { return time.Since(start).Round(time.Millisecond) }

	say("Reading config from %s", configDir)
	cfg, err := passbolt.LoadConfig(configDir)
	if err != nil {
		return err
	}

	start = time.Now()
	client, err := api.NewClient(httpClient(), "", cfg.ServerAddress, cfg.UserPrivateKey, cfg.UserPassword)
	if err != nil {
		return err
	}
	say("Private key unlocked in %v", elapsed())

	start = time.Now()
	pinned, err := passbolt.Verify(ctx, client, configDir)
	if err != nil {
		return err
	}
	if pinned {
		say("Server pinned (first use) in %v", elapsed())
	} else {
		say("Server verified in %v", elapsed())
	}

	start = time.Now()
	if err := client.Login(ctx); err != nil {
		return fmt.Errorf("logging in: %w", err)
	}
	defer client.Logout(ctx)
	say("Logged in to %s in %v", cfg.ServerAddress, elapsed())

	start = time.Now()
	resources, skipped, err := passbolt.ListResources(ctx, client)
	if err != nil {
		return err
	}
	say("%d resources listed in %v", len(resources), elapsed())
	if skipped > 0 {
		say("%d resources of an unsupported type skipped", skipped)
	}
	for _, r := range resources {
		fmt.Fprintf(w, "%s  %s  %s\n", r.Name, r.Username, r.URI)
	}
	return nil
}

func main() {
	if err := setupLogging(); err != nil {
		log.Printf("logging to %s disabled: %v", logPath, err)
	}
	log.Printf("passbolt starting (%s, pid %d)", runtime.Version(), os.Getpid())

	// RunCLI's default font is too small to read on a 300 dpi screen.
	ink.DefaultFontHeight = 28
	err := ink.RunCLI(func(ctx context.Context, w io.Writer) error {
		// RunCLI shows the error on screen; this also puts it in the log.
		err := run(ctx, w)
		if err != nil {
			log.Printf("error: %v", err)
		}
		return err
	}, &ink.RunConfig{Network: true})
	if err != nil {
		os.Exit(1)
	}
}

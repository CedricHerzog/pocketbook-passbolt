package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/passbolt/go-passbolt/api"

	"github.com/CedricHerzog/pocketbook-passbolt/internal/passbolt"
)

// load logs in and lists the user's resources, showing its progress on the
// loading screen, then the list or the error. It runs in its own goroutine
// and stops early when ctx is cancelled.
func (a *app) load(ctx context.Context) {
	defer a.wg.Done()
	list, skipped, err := a.fetch(ctx)
	if err != nil {
		log.Printf("error: %v", err)
		a.update(true, func() bool {
			if ctx.Err() != nil {
				return false // exiting
			}
			a.err, a.status = err, ""
			a.show(screenError)
			return true
		})
		return
	}
	a.update(true, func() bool {
		a.list, a.skipped = list, skipped
		a.show(screenList)
		return true
	})
}

// fetch reads the config, verifies the server, logs in and lists the
// resources. Each step reports how long it took.
func (a *app) fetch(ctx context.Context) ([]passbolt.Resource, int, error) {
	var start time.Time
	elapsed := func() time.Duration { return time.Since(start).Round(time.Millisecond) }

	a.say("Reading config from %s", configDir)
	cfg, err := passbolt.LoadConfig(configDir)
	if err != nil {
		return nil, 0, err
	}

	start = time.Now()
	a.setStatus("Unlocking private key")
	client, err := api.NewClient(httpClient(), "", cfg.ServerAddress, cfg.UserPrivateKey, cfg.UserPassword)
	if err != nil {
		return nil, 0, err
	}
	a.say("Private key unlocked in %v", elapsed())

	start = time.Now()
	a.setStatus("Verifying server")
	pinned, err := passbolt.Verify(ctx, client, configDir)
	if err != nil {
		return nil, 0, err
	}
	if pinned {
		a.say("Server pinned (first use) in %v", elapsed())
	} else {
		a.say("Server verified in %v", elapsed())
	}

	start = time.Now()
	a.setStatus("Logging in")
	if err := client.Login(ctx); err != nil {
		return nil, 0, fmt.Errorf("logging in: %w", err)
	}
	a.update(false, func() bool {
		a.client = client
		return false
	})
	a.say("Logged in to %s in %v", cfg.ServerAddress, elapsed())

	start = time.Now()
	a.setStatus("Listing resources")
	list, skipped, err := passbolt.ListResources(ctx, client)
	if err != nil {
		return nil, 0, err
	}
	a.say("%d resources listed in %v", len(list), elapsed())
	if skipped > 0 {
		a.say("%d resources of an unsupported type skipped", skipped)
	}
	return list, skipped, nil
}

// reveal decrypts the password of res, the resource at index i, and shows it
// if res is still on screen. It runs in its own goroutine. It logs how long
// decryption took or why it failed, never the resource or the password.
func (a *app) reveal(i int, res passbolt.Resource, c *api.Client) {
	defer a.wg.Done()
	start := time.Now()
	password, err := res.Password(c)
	if err != nil {
		log.Printf("error: %v", err)
	} else {
		log.Printf("password decrypted in %v", time.Since(start).Round(time.Millisecond))
	}
	a.update(false, func() bool {
		if a.screen != screenDetail || a.sel != i || !a.decrypting {
			return false
		}
		a.password, a.revealErr, a.decrypting = password, err, false
		return true
	})
}

// say adds a finished step to the loading screen and to the log. It must not
// be given resource details.
func (a *app) say(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(msg)
	a.update(false, func() bool {
		a.lines = append(a.lines, msg)
		a.status = ""
		return true
	})
}

// setStatus shows the step in progress on the loading screen.
func (a *app) setStatus(s string) {
	a.update(false, func() bool {
		a.status = s
		return true
	})
}

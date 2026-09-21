// Package passbolt reads the app's configuration, verifies the Passbolt
// server, lists the user's resources and decrypts their passwords, using
// go-passbolt. It has no PocketBook dependency, so it builds and runs on any
// machine.
package passbolt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
)

// Files in the configuration directory. The JSON keys match go-passbolt-cli's
// configuration keys.
const (
	configFile = "config.json" // written by the user
	keyFile    = "private.asc" // the user's armored private key
	verifyFile = "verify.json" // written by Verify on first use
)

// Config is what the client needs to log in.
type Config struct {
	ServerAddress  string `json:"serverAddress"`
	UserPassword   string `json:"userPassword"`
	UserPrivateKey string `json:"-"` // read from keyFile
}

// LoadConfig reads configFile and keyFile from dir.
func LoadConfig(dir string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", configFile, err)
	}
	if cfg.ServerAddress == "" {
		return cfg, fmt.Errorf("%s: serverAddress is not set", configFile)
	}
	if cfg.UserPassword == "" {
		return cfg, fmt.Errorf("%s: userPassword is not set", configFile)
	}
	key, err := os.ReadFile(filepath.Join(dir, keyFile))
	if err != nil {
		return cfg, err
	}
	cfg.UserPrivateKey = string(key)
	return cfg, nil
}

// verification is the token pair from api.Client.SetupServerVerification.
type verification struct {
	Token    string `json:"serverVerifyToken"`
	EncToken string `json:"serverVerifyEncToken"`
}

// Verify checks that the server is the one pinned in verifyFile. On first use
// there is no pin yet: it pins the server and returns pinned = true. It must
// run before the client logs in.
func Verify(ctx context.Context, c *api.Client, dir string) (pinned bool, err error) {
	path := filepath.Join(dir, verifyFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		token, encToken, err := c.SetupServerVerification(ctx)
		if err != nil {
			return false, fmt.Errorf("setting up server verification: %w", err)
		}
		data, err := json.MarshalIndent(verification{token, encToken}, "", "  ")
		if err != nil {
			return false, err
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return false, fmt.Errorf("saving server verification: %w", err)
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}

	var v verification
	if err := json.Unmarshal(data, &v); err != nil {
		return false, fmt.Errorf("parsing %s: %w", verifyFile, err)
	}
	if err := c.VerifyServer(ctx, v.Token, v.EncToken); err != nil {
		return false, fmt.Errorf("verifying server: %w (if the server key changed on purpose, delete %s)", err, verifyFile)
	}
	return false, nil
}

// Resource is a Passbolt resource as shown in the list. Its secret stays
// encrypted until Password is called.
type Resource struct {
	ID, Name, Username, URI string

	secret       string // armored, encrypted for the user; "" if the server sent none
	secretString bool   // the decrypted secret is the password itself, not JSON
}

// ListResources returns the user's resources sorted by name, with their
// secrets still encrypted. Resources of a type the SDK does not support are
// left out and counted in skipped.
func ListResources(ctx context.Context, c *api.Client) (list []Resource, skipped int, err error) {
	resources, err := c.GetResources(ctx, &api.GetResourcesOptions{ContainSecret: true})
	if err != nil {
		return nil, 0, fmt.Errorf("getting resources: %w", err)
	}
	for _, r := range resources {
		rType, err := c.GetResourceTypeCached(ctx, r.ResourceTypeID)
		if err != nil {
			return nil, 0, fmt.Errorf("getting resource type: %w", err)
		}
		// v4 resources have their name, username and URI in clear; v5
		// resources have them in encrypted metadata, decrypted here.
		_, name, username, uri, _, _, err := helper.GetResourceFromDataWithOptions(c, r, api.Secret{}, *rType, false)
		if errors.Is(err, helper.ErrUnsupportedResourceType) {
			skipped++
			continue
		}
		if err != nil {
			return nil, 0, fmt.Errorf("reading resource %s: %w", r.ID, err)
		}
		res := Resource{ID: r.ID, Name: name, Username: username, URI: uri, secretString: rType.IsSecretString()}
		if len(r.Secrets) > 0 {
			res.secret = r.Secrets[0].Data
		}
		list = append(list, res)
	}
	slices.SortFunc(list, func(a, b Resource) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return list, skipped, nil
}

// Password decrypts r's secret with c's private key. It runs locally, but c
// must not be logged out: Logout clears the key.
func (r Resource) Password(c *api.Client) (string, error) {
	if r.secret == "" {
		return "", errors.New("the server sent no secret for this resource")
	}
	data, err := c.DecryptSecretWithResourceID(r.ID, r.secret)
	if err != nil {
		return "", fmt.Errorf("decrypting secret: %w", err)
	}
	return parsePassword(data, r.secretString)
}

// parsePassword returns the password in decrypted secret data: the data itself
// for string secrets, else its "password" field ("" for TOTP-only resources).
// This is how helper.GetResourceFromDataWithOptions reads it.
func parsePassword(data string, isString bool) (string, error) {
	if isString {
		return data, nil
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(data), &fields); err != nil {
		// json's errors quote the input, which is the secret.
		return "", errors.New("secret is not valid JSON")
	}
	return helper.GetStringField(fields, "password"), nil
}

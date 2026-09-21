# pocketbook-passbolt

A Passbolt client for PocketBook e-readers, written in Go with
[dennwc/inkview](https://github.com/dennwc/inkview) (cgo bindings to PocketBook's `libinkview`) and
[passbolt/go-passbolt](https://github.com/passbolt/go-passbolt).

The app (`cmd/passbolt`) verifies the server, logs in and shows your resources as a paged list
(name, username, URI). Tap a resource to decrypt its password and show it. It targets the
InkPad 3 (PB740, Allwinner B288) on firmware 6.x, and should work on other B288 readers.

## Requirements

- Docker (Docker Desktop or OrbStack). On Apple silicon, enable Rosetta for amd64 emulation,
  because the PocketBook SDK's tools are Linux x86_64 binaries.
- `make`

The build image (`docker/Dockerfile`) bundles the official
[PocketBook SDK](https://github.com/pocketbook/SDK_6.3.0) (branch `6.5`, for firmware 6.x),
Go, and `gdb-multiarch`. Build it once:

```sh
make image
```

## Build and install

```sh
make build      # release build  -> build/passbolt.app
make debug      # debug build    -> build/passbolt.app (no optimisation, with symbols)
```

Connect the reader over USB and choose **PC link** on its screen, then:

```sh
make deploy
```

This copies these files to the reader, creates the `applications/passbolt/` config folder and
ejects the reader:

| File on the reader | Purpose |
|---|---|
| `applications/passbolt.app` | the app |
| `applications/icons/passbolt.bmp`, `passbolt_f.bmp` | the launcher icon, and its selected variant |
| `applications/passbolt-gdb.app` | starts the app under gdbserver |
| `system/usr/bin/gdbserver` | the debug server, from the SDK |

### Launcher icon

On firmware 6.x, the launcher shows an app's own icon only if the app is listed in
`system/config/desktop/view.json`. Without that entry, the app keeps the generic icon. The file
documents its own format in its `_comment` fields:

- A plain string as `icon` names an icon of the theme, so an icon file must be given as
  `{ "path": ... }`, relative to the reader's storage.
- The app's `path` must be absolute, though. With the relative path of the file's own example,
  the launcher shows the icon but doesn't start the app.

To add the entry:

1. Keep a copy of `view.json`, for example as `view.json.orig` next to it.
2. Add this entry to the `applications` object, before its `_comment`:

   ```json
   "U_passbolt": {
       "path": "/mnt/ext1/applications/passbolt.app",
       "title": "Passbolt",
       "icon": { "path": "applications/icons/passbolt.bmp" },
       "focused_icon": { "path": "applications/icons/passbolt_f.bmp" }
   },
   ```

3. Add `"U_passbolt"` to the `apps` of one of the groups in `view.groups`, for example
   `@General`.

Edit the file with a plain-text editor, because TextEdit's smart quotes break JSON. Then check it
with `python3 -m json.tool < view.json` and eject the reader. If the launcher shows an error or
no longer shows your apps, restore your copy.

The `hash.txt` file next to `view.json` holds the MD5 of the firmware's version of the file. The
firmware probably uses it to detect local changes. Leave it as it is.

## Configure

The app reads its settings from `applications/passbolt/` on the reader. Copy two files there
over USB:

- `config.json`, with your server address and your passphrase:

  ```json
  {
    "serverAddress": "https://passbolt.example.com",
    "userPassword": "your passphrase"
  }
  ```

- `private.asc`, your armored private key. This is the recovery kit file that Passbolt
  offered to download during account setup. You can download it again from the Passbolt web
  app, under your profile's keys.

> **Security.** The passphrase is stored in plain text on the reader: anyone with the reader and
> a USB cable can read both it and the key. The app also does not check the server's TLS
> certificate, so someone on the same network could intercept the session. Your secrets stay
> end-to-end encrypted either way.

## Use

Turn Wi-Fi on, then open **Applications → passbolt**. Under the logo, the app shows its
progress:

1. it connects to Wi-Fi;
2. it reads `config.json` and `private.asc`, and unlocks the key;
3. it verifies the server. On the first run it pins the server and saves the pin to
   `applications/passbolt/verify.json`. Later runs check that the server still matches the pin.
   If the server key changed on purpose, delete `verify.json` to pin it again;
4. it logs in and lists your resources.

Each step shows how long it took; the timings also go to the log file. The list then shows your
resources sorted by name: one page at a time, with the name, then the username and URI.

| Screen | Tap | Buttons |
|---|---|---|
| Loading | **Cancel** exits | Back, Menu or Prev exits |
| List | a resource opens it; **‹ Prev** / **Next ›** or a swipe turns the page; **Exit** exits | page buttons turn the page; Back or Menu exits |
| Resource | **‹ Back** returns to the list | any button returns to the list |
| Error | anywhere exits | any button exits |

The app logs the codes of the buttons it does not use (`unhandled key 0x…`), so `make log` shows
what your reader's buttons send.

A password is decrypted on the reader only when you open its resource. It is never written to
disk or to the log. Leaving the resource screen does a full refresh, so the password leaves no
ghost on the screen. It stays in memory until the app exits, though, and if the reader goes to
sleep on that screen, the sleep screen may keep showing it. The app stays logged in while it
runs, and logs out when you exit.

## Debugging

### Log file

The app appends to `applications/passbolt.log` on the reader. Its standard output, standard
error and Go crash tracebacks also go to this file. The log records progress and errors, never
resource details or passwords. To read it, reconnect over USB and run:

```sh
make log
```

### gdb over Wi-Fi

1. Deploy a debug build: `make debug deploy`.
2. On the reader, turn Wi-Fi on and look up its IP address, in the Wi-Fi network details or in
   your router.
3. Open **passbolt-gdb**. The launcher looks like nothing happened: gdbserver is waiting on port
   10002, with the app paused before its first instruction.
4. On the Mac, run:

   ```sh
   make gdb IP=192.168.1.42
   ```

5. Then, in gdb:

   ```
   (gdb) break main.(*app).load
   (gdb) continue
   (gdb) bt
   (gdb) next
   (gdb) info locals
   (gdb) kill            # ends the app and gdbserver
   ```

`debug/gdbinit` loads the reader's libraries from the SDK and ignores `SIGURG`, which Go's
scheduler uses. Without that setting, gdb would stop constantly.

## Logo

The logos in `assets/` are the official Passbolt SVGs, from
[passbolt/passbolt_styleguide](https://github.com/passbolt/passbolt_styleguide) (AGPL-3.0).
The images the app and the launcher use are generated from them and committed. E-ink has no red,
so the red key is dark gray in the app and black on the launcher icon. To regenerate them, you
need `rsvg-convert` and ImageMagick on the Mac (`brew install librsvg imagemagick`):

```sh
make -B logos
```

Passbolt and its logo are trademarks of Passbolt SA. This project is not affiliated with
Passbolt.

## Layout

```
cmd/passbolt/          the app: screens and drawing, loading flow, logo, log file
cmd/passbolt/images/   in-app logos, embedded in the binary (generated by make logos)
internal/passbolt/     config, server verification, resource list, password decryption
                       (go-passbolt, no PocketBook code)
assets/                official Passbolt logo SVGs
docker/Dockerfile      SDK + Go + gdb build image
device/                files that run on the reader
device/icons/          launcher icons (generated by make logos)
debug/gdbinit          gdb settings for this target
scripts/reader.sh      deploy to / read logs from the USB-mounted reader
```

# pocketbook-passbolt

A Passbolt client for PocketBook e-readers, written in Go with
[dennwc/inkview](https://github.com/dennwc/inkview) (cgo bindings to PocketBook's `libinkview`) and
[passbolt/go-passbolt](https://github.com/passbolt/go-passbolt).

For now the app (`cmd/passbolt`) verifies the server, logs in and lists your resources (name,
username, URI). It does not show passwords. It targets the InkPad 3 (PB740, Allwinner B288) on
firmware 6.x, and should work on other B288 readers.

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
| `applications/passbolt-gdb.app` | starts the app under gdbserver |
| `system/usr/bin/gdbserver` | the debug server, from the SDK |

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

Turn Wi-Fi on, then open **Applications → passbolt**. The app:

1. reads `config.json` and `private.asc`, and unlocks the key;
2. verifies the server. On the first run it pins the server and saves the pin to
   `applications/passbolt/verify.json`. Later runs check that the server still matches the pin.
   If the server key changed on purpose, delete `verify.json` to pin it again;
3. logs in and lists your resources, one per line: name, username, URI.

Each step shows how long it took (key unlock, verify, login, listing); the timings also go to
the log file. Any button or tap exits. The screen keeps only the lines that fit, so a long list shows its
last page only.

## Debugging

### Log file

The app appends to `applications/passbolt.log` on the reader. Its standard output, standard
error and Go crash tracebacks also go to this file. The log records progress and errors, never
resource details. To read it, reconnect over USB and run:

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
   (gdb) break main.run
   (gdb) continue
   (gdb) bt
   (gdb) next
   (gdb) info locals
   (gdb) kill            # ends the app and gdbserver
   ```

`debug/gdbinit` loads the reader's libraries from the SDK and ignores `SIGURG`, which Go's
scheduler uses. Without that setting, gdb would stop constantly.

## Layout

```
cmd/passbolt/        the app: on-screen console, flow, log file
internal/passbolt/   config, server verification, resource list (go-passbolt, no PocketBook code)
docker/Dockerfile    SDK + Go + gdb build image
device/              files that run on the reader
debug/gdbinit        gdb settings for this target
scripts/reader.sh    deploy to / read logs from the USB-mounted reader
```

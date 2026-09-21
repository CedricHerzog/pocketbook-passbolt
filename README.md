# pocketbook-passbolt

A Passbolt client for PocketBook e-readers, written in Go with
[dennwc/inkview](https://github.com/dennwc/inkview) (cgo bindings to PocketBook's `libinkview`).

For now it contains a **Hello World app** (`cmd/hello`) and the build, deploy and debug setup
around it. It targets the InkPad 3 (PB740, Allwinner B288) on firmware 6.x, and should work on
other B288 readers.

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
make build      # release build  -> build/hello.app
make debug      # debug build    -> build/hello.app (no optimisation, with symbols)
```

Connect the reader over USB and choose **PC link** on its screen, then:

```sh
make deploy
```

This copies these files to the reader and ejects it:

| File on the reader | Purpose |
|---|---|
| `applications/hello.app` | the app |
| `applications/hello-gdb.app` | starts the app under gdbserver |
| `system/usr/bin/gdbserver` | the debug server, from the SDK |

Open **Applications → hello** on the reader.

The app shows the device model, firmware and Wi-Fi address. Tapping the screen increments a
counter, and any hardware button exits.

## Debugging

### Log file

The app appends to `applications/hello.log` on the reader. Its standard output, standard error
and Go crash tracebacks also go to this file. To read it, reconnect over USB and run:

```sh
make log
```

A **long press** crashes the app on purpose (`crash self-test`), to check that tracebacks
reach the log.

### gdb over Wi-Fi

1. Deploy a debug build: `make debug deploy`.
2. On the reader, turn Wi-Fi on. Open **hello** once and note the IP it shows, then exit.
3. Open **hello-gdb**. The launcher looks like nothing happened: gdbserver is waiting on port
   10002, with the app paused before its first instruction.
4. On the Mac, run:

   ```sh
   make gdb IP=192.168.1.42
   ```

5. Then, in gdb:

   ```
   (gdb) break main.(*app).Pointer
   (gdb) continue
         ... tap the reader's screen ...
   (gdb) bt
   (gdb) info locals
   (gdb) print a.taps
   (gdb) next
   (gdb) kill            # ends the app and gdbserver
   ```

`debug/gdbinit` loads the reader's libraries from the SDK and ignores `SIGURG`, which Go's
scheduler uses. Without that setting, gdb would stop constantly.

## Layout

```
cmd/hello/           the Hello World app
docker/Dockerfile    SDK + Go + gdb build image
device/              files that run on the reader
debug/gdbinit        gdb settings for this target
scripts/reader.sh    deploy to / read logs from the USB-mounted reader
```

#!/bin/sh
# Talks to a PocketBook reader mounted over USB ("PC link" on the reader).
#
#   scripts/reader.sh deploy   copy the app, its gdb launcher and gdbserver, then eject
#   scripts/reader.sh log      print the app's log file
#
# The reader's volume is detected under /Volumes; set PB_VOLUME to override.
set -eu

# The reader's storage has lowercase applications/ and system/ folders.
# Compare names exactly: macOS paths are case-insensitive, so a plain -d test
# would also match the boot disk's /Applications and /System (and
# /Volumes/Macintosh HD is a symlink to /, hence skipping symlinks too).
find_volume() {
	for v in /Volumes/*; do
		[ -L "$v" ] && continue
		entries=$(ls -1 "$v" 2>/dev/null) || continue
		if printf '%s\n' "$entries" | grep -qx applications &&
			printf '%s\n' "$entries" | grep -qx system; then
			echo "$v"
			return 0
		fi
	done
	return 1
}

vol=${PB_VOLUME:-$(find_volume || true)}
if [ -z "$vol" ]; then
	echo "No PocketBook found under /Volumes. Connect the reader over USB and choose 'PC link'." >&2
	exit 1
fi

case ${1:-} in
deploy)
	for f in build/hello.app build/gdbserver; do
		[ -f "$f" ] || { echo "$f is missing; run 'make build' or 'make debug' first." >&2; exit 1; }
	done
	mkdir -p "$vol/system/usr/bin"
	# -X and COPYFILE_DISABLE keep macOS from writing ._* files, which the
	# reader's launcher would list as broken apps.
	export COPYFILE_DISABLE=1
	cp -X build/hello.app device/hello-gdb.app "$vol/applications/"
	cp -X build/gdbserver "$vol/system/usr/bin/"
	rm -f "$vol/applications/._hello.app" "$vol/applications/._hello-gdb.app" "$vol/system/usr/bin/._gdbserver"
	sync
	echo "Deployed to $vol"
	diskutil eject "$vol"
	;;
log)
	cat "$vol/applications/hello.log"
	;;
*)
	echo "usage: $0 deploy|log" >&2
	exit 2
	;;
esac

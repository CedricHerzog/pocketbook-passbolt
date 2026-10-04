#!/bin/sh
# Talks to a PocketBook reader mounted over USB ("PC link" on the reader).
#
#   scripts/reader.sh deploy   copy the app and its launcher icons, create the
#                              app's config folder, then eject
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
	[ -f build/passbolt.app ] || { echo "build/passbolt.app is missing; run 'make build' first." >&2; exit 1; }
	# applications/passbolt/ holds the app's config.json and private.asc;
	# applications/icons/ the launcher icons that view.json points to.
	mkdir -p "$vol/applications/passbolt" "$vol/applications/icons"
	# -X and COPYFILE_DISABLE keep macOS from writing ._* files, which the
	# reader's launcher would list as broken apps.
	export COPYFILE_DISABLE=1
	cp -X build/passbolt.app "$vol/applications/"
	cp -X device/icons/passbolt.bmp device/icons/passbolt_f.bmp "$vol/applications/icons/"
	rm -f "$vol/applications/._passbolt.app" "$vol/applications/._icons" \
		"$vol/applications/icons/._passbolt.bmp" "$vol/applications/icons/._passbolt_f.bmp"
	sync
	echo "Deployed to $vol"
	diskutil eject "$vol"
	;;
*)
	echo "usage: $0 deploy" >&2
	exit 2
	;;
esac

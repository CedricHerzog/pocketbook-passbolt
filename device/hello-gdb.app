#!/bin/sh
# Starts hello.app under gdbserver, which waits for a debugger on port 10002.
# Turn Wi-Fi on first, then attach from the Mac with: make gdb IP=<reader IP>
exec /mnt/ext1/system/usr/bin/gdbserver 0.0.0.0:10002 /mnt/ext1/applications/hello.app

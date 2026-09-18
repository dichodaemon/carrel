#!/usr/bin/env bash
# udev-twang-usb -- grant world access to the EK-RA8D2 board's USB device
# (Zephyr VID 0x2fe3) and the ALSA nodes it creates, so the non-root container
# user (UID 1000) can reach them. Run once per machine on the HOST; requires
# sudo. Complements bin/udev-jlink.sh (which covers the SEGGER probe).
set -euo pipefail

sudo tee /etc/udev/rules.d/99-twang-usb.rules >/dev/null <<'EOF'
# Board's USB device (Zephyr Project VID) — world access for lsusb/pyusb.
SUBSYSTEM=="usb", ATTRS{idVendor}=="2fe3", MODE="0666"
# ALSA sound/MIDI/UMP nodes the board's UAC2+MIDI2 device creates.
# Broad on purpose: the snd nodes don't reliably carry the parent USB VID, and
# this is a single-user dev machine (same philosophy as 99-jlink.rules).
SUBSYSTEM=="snd", MODE="0666"
EOF

sudo udevadm control --reload-rules
sudo udevadm trigger

echo "twang USB + sound udev rule installed at /etc/udev/rules.d/99-twang-usb.rules"

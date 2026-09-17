#!/usr/bin/env bash
# udev-jlink -- grant world access to SEGGER J-Link probes (USB vendor 1366) so
# the non-root container user (UID 1000) can open the device node. Run once per
# machine; requires sudo. Equivalent to the rule SEGGER's own install ships
# (99-jlink.rules), but here it is explicit and reproducible.
set -euo pipefail

sudo tee /etc/udev/rules.d/99-jlink.rules >/dev/null <<'EOF'
SUBSYSTEM=="usb", ATTRS{idVendor}=="1366", MODE="0666"
EOF

sudo udevadm control --reload-rules
sudo udevadm trigger

echo "J-Link udev rule installed at /etc/udev/rules.d/99-jlink.rules"

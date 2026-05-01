#!/bin/bash
# sunbeams-drm-force: privileged DRM connector force-toggle helper.
# Installed at /usr/local/sbin/sunbeams-drm-force, mode 0700 root:root.
# Invoked via sudoers NOPASSWD by `sunbeams switch on/off` under the debugfs
# strategy. See docs/superpowers/specs/2026-04-29-gaming-mode-support-design.md
set -euo pipefail

ACTION="${1:-}"
CONNECTOR="${2:-}"

if [[ ! "$ACTION" =~ ^(on|off)$ ]]; then
    echo "bad action (expected on|off)" >&2
    exit 2
fi

# DRM connector names: HDMI-A-1, DP-1, eDP-1, DSI-1, VGA-1, DP-A-1, etc.
if [[ ! "$CONNECTOR" =~ ^[A-Za-z]+(-[A-Z])?-[0-9]+$ ]]; then
    echo "bad connector name" >&2
    exit 2
fi

shopt -s nullglob
matches=( /sys/kernel/debug/dri/*/"$CONNECTOR"/force )
if (( ${#matches[@]} == 0 )); then
    echo "no debugfs path for $CONNECTOR (debugfs may not be mounted)" >&2
    exit 3
fi
if (( ${#matches[@]} > 1 )); then
    echo "multiple debugfs paths for $CONNECTOR (multi-GPU not supported)" >&2
    exit 3
fi

echo "$ACTION" > "${matches[0]}"
udevadm trigger --subsystem-match=drm

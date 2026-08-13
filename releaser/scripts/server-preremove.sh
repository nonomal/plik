#!/bin/sh
set -e

# Stop and disable the service before removal
if [ -d /run/systemd/system ]; then
  systemctl stop plikd.service || true
  systemctl disable plikd.service || true
fi

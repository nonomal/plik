#!/bin/sh
set -e

# Reload systemd after unit file removal
if [ -d /run/systemd/system ]; then
  systemctl daemon-reload
fi

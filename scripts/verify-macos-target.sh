#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <Mach-O binary>" >&2
  exit 1
fi

minimum=$(xcrun vtool -show-build "$1" | awk '$1 == "minos" {print $2}' | sort -u)
if [[ "$minimum" != "13.0" ]]; then
  echo "Expected macOS 13.0 deployment target for $1; found: $minimum" >&2
  exit 1
fi

#!/bin/bash
# Mova Context uninstaller for macOS (double-click). Same logic as ../linux/uninstall.sh.
# NOT executed on a real Mac by the author: run with --dry-run first.
cd "$(dirname "$0")" || exit 1
../linux/uninstall.sh "$@"; rc=$?
echo; read -rp "Press Enter to close this window..." _ || true
exit $rc

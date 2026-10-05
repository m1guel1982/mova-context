#!/bin/bash
# uninstall.sh -- Mova Context uninstaller for Linux (macOS wraps it).
# Undoes what installers/linux/install.sh does: binary, PATH line, MOVA_PROJECT_ROOT line, temp leftovers.
# The repo folder (your projects/memory) is deleted ONLY if you confirm or pass --remove-repo.
# Usage: uninstall.sh [--dir <bin dir>] [--remove-repo|--keep-repo] [--purge-path] [--yes] [--dry-run]
#   --purge-path  also drop the PATH line when the bin folder still holds other programs (default: keep, it is shared)
set -uo pipefail
info() { echo "[Mova Uninstaller] $*"; }
warn() { echo "[Mova Uninstaller] WARNING: $*" >&2; }
fail() { echo "[Mova Uninstaller] ERROR: $*" >&2; exit 1; }

BIN_DIR_ARG=""; REMOVE_REPO="ask"; PURGE=0; YES=0; DRY=0
while [ $# -gt 0 ]; do
  case "$1" in
    -d|--dir) [ $# -ge 2 ] || fail "Option $1 requires a path."; BIN_DIR_ARG="$2"; shift 2 ;;
    --remove-repo) REMOVE_REPO="yes"; shift ;;
    --keep-repo) REMOVE_REPO="no"; shift ;;
    --purge-path) PURGE=1; shift ;;
    -y|--yes) YES=1; shift ;;
    -n|--dry-run) DRY=1; shift ;;
    -h|--help) sed -n 2,6p "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) fail "Unknown option: $1" ;;
  esac
done
run() { if [ "$DRY" -eq 1 ]; then info "[dry-run] $*"; else "$@"; fi; }
SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; SELF_REPO="$(cd "$SELF/../.." && pwd)"
PROFILES=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.zprofile" "$HOME/.profile")

BIN_DIR=""
if [ -n "$BIN_DIR_ARG" ]; then BIN_DIR="${BIN_DIR_ARG%/}"; else
  F="$(command -v mova 2>/dev/null || true)"
  if [ -n "$F" ] && [ -f "$F" ]; then BIN_DIR="$(dirname "$F")"; else
    GP="${GOPATH:-$HOME/go}"; GP="${GP%%:*}"
    for d in "$GP/bin" "$HOME/go/bin"; do [ -f "$d/mova" ] && { BIN_DIR="$d"; break; }; done
  fi
fi
if [ -n "$BIN_DIR" ]; then info "Binary folder: $BIN_DIR"; else warn "No installed mova binary found; cleaning the rest."; fi

REPO=""
for p in "${PROFILES[@]}"; do
  [ -f "$p" ] || continue
  v="$(grep -E '^export MOVA_PROJECT_ROOT=' "$p" | tail -n1 | sed -E 's/^export MOVA_PROJECT_ROOT=//; s/^"//; s/"$//')"
  [ -n "$v" ] && { REPO="$v"; break; }
done
[ -z "$REPO" ] && REPO="${MOVA_PROJECT_ROOT:-}"
[ -z "$REPO" ] && [ -f "$SELF_REPO/workflow.md" ] && REPO="$SELF_REPO"
info "Repo folder:   ${REPO:-(none found)}"

confirm() { [ "$YES" -eq 1 ] && return 0; [ -t 0 ] || return 1; read -rp "$1 [y/N] " a || return 1; case "$a" in y|Y|yes) return 0;; *) return 1;; esac; }

# 1. stop running mova
if command -v pgrep >/dev/null 2>&1; then
  for pid in $(pgrep -u "$(id -u)" -x mova 2>/dev/null || true); do
    [ "$pid" = "$$" ] && continue; info "Stopping mova (PID $pid)"; run kill "$pid"
  done
fi

# 2. binary
if [ -n "$BIN_DIR" ]; then
  for f in "$BIN_DIR/mova" "$BIN_DIR"/mova.new.*; do [ -e "$f" ] && { info "Deleting $f"; run rm -f "$f"; }; done
fi

# 3. shell profiles
OTHERS=0
if [ -n "$BIN_DIR" ] && [ -d "$BIN_DIR" ]; then
  OTHERS="$(find "$BIN_DIR" -mindepth 1 -maxdepth 1 ! -name mova ! -name 'mova.new.*' 2>/dev/null | wc -l)"
fi
PATH_LINE=""; [ -n "$BIN_DIR" ] && PATH_LINE="export PATH=\"$BIN_DIR:\$PATH\""
for p in "${PROFILES[@]}"; do
  [ -f "$p" ] || continue
  drop=0
  if [ -n "$PATH_LINE" ] && grep -qxF "$PATH_LINE" "$p" && { [ "$OTHERS" -eq 0 ] || [ "$PURGE" -eq 1 ]; }; then drop=1; fi
  root=0; grep -qE '^export MOVA_PROJECT_ROOT=' "$p" && root=1
  [ "$drop" -eq 1 ] || [ "$root" -eq 1 ] || continue
  info "Cleaning $p (PATH line: $drop, MOVA_PROJECT_ROOT line: $root)"
  if [ "$DRY" -eq 0 ]; then
    tmp="$(mktemp "${TMPDIR:-/tmp}/mova-un.XXXXXX")"
    grep -vE '^export MOVA_PROJECT_ROOT=' "$p" > "$tmp"
    if [ "$drop" -eq 1 ]; then grep -vxF "$PATH_LINE" "$tmp" > "$tmp.2" || true; mv "$tmp.2" "$tmp"; fi
    cat "$tmp" > "$p"; rm -f "$tmp"
    [ -s "$p" ] || { info "$p is empty now -> removed"; rm -f "$p"; }
  fi
done
[ "$OTHERS" -gt 0 ] && [ "$PURGE" -eq 0 ] && info "Kept $BIN_DIR on PATH: it holds $OTHERS other item(s). Use --purge-path to drop it."
if [ "$DRY" -eq 0 ] && [ -n "$BIN_DIR" ] && [ -d "$BIN_DIR" ] && [ -z "$(ls -A "$BIN_DIR")" ]; then info "Removing empty $BIN_DIR"; rmdir "$BIN_DIR"; fi

# 4. temp leftovers
T="${TMPDIR:-/tmp}"; T="${T%/}"
for f in "$T"/mova-trace-* "$T"/mova-build-*; do [ -e "$f" ] && { info "Deleting temp $f"; run rm -rf "$f"; }; done

# 5. repo folder (explicit)
if [ -n "$REPO" ] && [ -d "$REPO" ] && [ "$REMOVE_REPO" != "no" ]; then
  safe=1; case "$REPO" in "/"|"$HOME"|"") safe=0;; esac
  { [ -f "$REPO/workflow.md" ] && [ -d "$REPO/src" ]; } || safe=0
  if [ "$safe" -eq 0 ]; then warn "$REPO does not look like a mova repo (workflow.md + src/ required) -> NOT deleted."
  elif [ "$REMOVE_REPO" = "yes" ] || confirm "Delete repo $REPO (your projects/ and memory.md inside are lost)?"; then
    info "Deleting $REPO"; cd / && run rm -rf "$REPO"
  else info "Kept $REPO"; fi
fi

if [ "$DRY" -eq 0 ]; then
  left=0
  for p in "${PROFILES[@]}"; do [ -f "$p" ] && grep -q MOVA_PROJECT_ROOT "$p" && { warn "still in $p"; left=1; }; done
  [ -n "$BIN_DIR" ] && [ -e "$BIN_DIR/mova" ] && { warn "binary still at $BIN_DIR/mova"; left=1; }
  [ "$left" -eq 0 ] && info "Clean: no mova binary or MOVA_PROJECT_ROOT left in your profiles."
  info "Open a NEW terminal (a script cannot change the environment of the shell that launched it)."
fi
exit 0

#!/bin/bash
# install.sh — Mova Context installer for Linux.
set -euo pipefail

info()  { echo "[Mova Installer] $*"; }
fail()  { echo "[Mova Installer] ERROR: $*" >&2; exit 1; }

info "Starting installation..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

ARCH="amd64"
case "$(uname -m)" in
    arm64|aarch64) ARCH="arm64" ;;
esac

DIST_BINARY="$REPO_ROOT/dist/mova-linux-$ARCH"
BUILT_BINARY=""
TMP_BINARY=""

# Limpieza automática al salir si se usó mktemp
cleanup() {
    if [ -n "$TMP_BINARY" ] && [ -f "$TMP_BINARY" ]; then
        rm -f "$TMP_BINARY"
    fi
}
trap cleanup EXIT

if [ -f "$DIST_BINARY" ]; then
    info "Found prebuilt binary: $DIST_BINARY"
    BUILT_BINARY="$DIST_BINARY"
else
    info "No prebuilt binary found — building from source (requires Go)..."
    if ! command -v go >/dev/null 2>&1; then
        fail "Go is not installed or not on PATH. Install Go from https://go.dev/dl and run this installer again, or run 'make build-all' first."
    fi
    TMP_BINARY="$(mktemp "${TMPDIR:-/tmp}/mova.XXXXXX")"
    ( cd "$REPO_ROOT" && go build -ldflags="-s -w" -o "$TMP_BINARY" ./src/cli ) \
        || fail "Build failed. Check the Go output above."
    BUILT_BINARY="$TMP_BINARY"
    info "Build succeeded: $BUILT_BINARY"
fi

# Resolución segura de GOPATH sin que set -e rompa si Go no está instalado
GOPATH_DIR="${GOPATH:-}"
if [ -z "$GOPATH_DIR" ] && command -v go >/dev/null 2>&1; then
    GOPATH_DIR="$(go env GOPATH 2>/dev/null || echo "")"
fi
if [ -z "$GOPATH_DIR" ]; then
    GOPATH_DIR="$HOME/go"
fi

BIN_DIR="$GOPATH_DIR/bin"
mkdir -p "$BIN_DIR"

cp -f "$BUILT_BINARY" "$BIN_DIR/mova"
chmod +x "$BIN_DIR/mova"
info "Installed: $BIN_DIR/mova"

# Detección del archivo de configuración según el shell activo del usuario
PROFILE="$HOME/.bashrc"
USER_SHELL="$(basename "${SHELL:-bash}")"
if [ "$USER_SHELL" = "zsh" ] || [ -f "$HOME/.zshrc" ]; then
    PROFILE="$HOME/.zshrc"
fi

if ! grep -qF "$BIN_DIR" "$PROFILE" 2>/dev/null; then
    echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$PROFILE"
    info "Added $BIN_DIR to PATH in $PROFILE. Open a NEW terminal window for this to take effect."
else
    info "$BIN_DIR is already on your PATH ($PROFILE)."
fi

if ! grep -q "MOVA_PROJECT_ROOT" "$PROFILE" 2>/dev/null; then
    echo "export MOVA_PROJECT_ROOT=\"$REPO_ROOT\"" >> "$PROFILE"
    info "Set MOVA_PROJECT_ROOT in $PROFILE — mova now works from any folder or drive."
else
    info "MOVA_PROJECT_ROOT is already set in $PROFILE — leaving it as-is."
fi

info "Done."

export PATH="$BIN_DIR:$PATH"
export MOVA_PROJECT_ROOT="$REPO_ROOT"
cd "$REPO_ROOT"

echo ""
echo "Which console would you like to use, ready to run mova?"
echo "  [1] Continue right here (default)"
echo "  [2] Open a new terminal window"
echo "  [3] Don't open one"
read -rp "Choose 1-3 and press Enter (default: 1): " choice
choice="${choice:-1}"

open_new_terminal() {
    local cmd="cd '$REPO_ROOT'; export PATH='$BIN_DIR:\$PATH'; export MOVA_PROJECT_ROOT='$REPO_ROOT'; mova; exec \"\${SHELL:-bash}\""
    if [ -n "${TERMINAL:-}" ] && command -v "$TERMINAL" >/dev/null 2>&1; then
        "$TERMINAL" -e bash -c "$cmd" >/dev/null 2>&1 & disown; return 0
    fi
    for term in x-terminal-emulator gnome-terminal konsole xfce4-terminal tilix alacritty kitty xterm; do
        if command -v "$term" >/dev/null 2>&1; then
            case "$term" in
                gnome-terminal|tilix) "$term" -- bash -c "$cmd" >/dev/null 2>&1 & disown ;;
                *)                    "$term" -e bash -c "$cmd" >/dev/null 2>&1 & disown ;;
            esac
            return 0
        fi
    done
    return 1
}

case "$choice" in
    2)
        if open_new_terminal; then
            info "Opened a new terminal window."
        else
            info "No terminal emulator found automatically — continuing in this one instead."
        fi
        read -rp "Press Enter to close this window..." _ || true
        ;;
    3)
        info "OK — remember to open a NEW terminal window for the PATH change to apply."
        read -rp "Press Enter to close this window..." _ || true
        ;;
    *)
        info "Ready. Running mova from $REPO_ROOT — this window stays open."
        exec "${SHELL:-/bin/bash}" -i
        ;;
esac
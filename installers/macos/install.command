#!/bin/bash
# install.command — double-click entry point for the Mova Context macOS installer.
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

DIST_BINARY="$REPO_ROOT/dist/mova-macos-$ARCH"
BUILT_BINARY=""
TMP_BINARY=""

# Limpieza automática de binario temporal al finalizar
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

# Resolución segura de GOPATH en macOS sin depender de la presencia de Go
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

# Remueve el atributo de cuarentena de Gatekeeper si el binario fue descargado de la web
xattr -d com.apple.quarantine "$BIN_DIR/mova" 2>/dev/null || true

info "Installed: $BIN_DIR/mova"

# Garantiza que ~/.zshrc exista para evitar fallos de lectura/escritura
PROFILE="$HOME/.zshrc"
touch "$PROFILE"

if ! grep -qF "$BIN_DIR" "$PROFILE" 2>/dev/null; then
    echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$PROFILE"
    info "Added $BIN_DIR to PATH in $PROFILE. Open a NEW terminal window for this to take effect."
else
    info "$BIN_DIR is already on your PATH ($PROFILE)."
fi

if ! grep -q "MOVA_PROJECT_ROOT" "$PROFILE" 2>/dev/null; then
    echo "export MOVA_PROJECT_ROOT=\"$REPO_ROOT\"" >> "$PROFILE"
    info "Set MOVA_PROJECT_ROOT in $PROFILE — mova now works from any folder or volume."
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
echo "  [2] Open a new Terminal window"
echo "  [3] Don't open one"
read -rp "Choose 1-3 and press Enter (default: 1): " choice
choice="${choice:-1}"

case "$choice" in
    2)
        # Invocación con comillas escapadas para soportar rutas con espacios
        CMD_PAYLOAD="cd \"$REPO_ROOT\"; export PATH=\"$BIN_DIR:\$PATH\"; export MOVA_PROJECT_ROOT=\"$REPO_ROOT\"; mova"
        if osascript -e "tell application \"Terminal\" to do script \"$CMD_PAYLOAD\"" >/dev/null 2>&1; then
            info "Opened a new Terminal window."
        else
            info "Could not open a new Terminal window automatically — continuing in this one instead."
        fi
        read -rp "Press Enter to close this window..." _ || true
        ;;
    3)
        info "OK — remember to open a NEW terminal window for the PATH change to apply."
        read -rp "Press Enter to close this window..." _ || true
        ;;
    *)
        info "Ready. Running mova from $REPO_ROOT — this window stays open."
        exec "${SHELL:-/bin/zsh}" -i
        ;;
esac
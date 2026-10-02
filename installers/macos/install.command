#!/bin/bash
# install.command -- double-click entry point for the Mova Context macOS installer.
#
# Usage: install.command [--dir <install directory>]
#   Without --dir the installer asks for a directory; pressing Enter keeps the default
#   ($GOPATH/bin, or ~/go/bin).
set -euo pipefail

info()  { echo "[Mova Installer] $*"; }
warn()  { echo "[Mova Installer] WARNING: $*" >&2; }
fail()  { echo "[Mova Installer] ERROR: $*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Argumentos
# ---------------------------------------------------------------------------
INSTALL_DIR_ARG=""
while [ $# -gt 0 ]; do
    case "$1" in
        -d|--dir)
            [ $# -ge 2 ] || fail "Option $1 requires a path."
            INSTALL_DIR_ARG="$2"; shift 2 ;;
        --dir=*)
            INSTALL_DIR_ARG="${1#--dir=}"; shift ;;
        -h|--help)
            echo "Usage: install.command [--dir <install directory>]"; exit 0 ;;
        *)
            fail "Unknown option: $1" ;;
    esac
done

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
STAGED_BINARY=""

# Limpieza automática al salir (binario temporal de build y copia intermedia)
cleanup() {
    if [ -n "$TMP_BINARY" ] && [ -f "$TMP_BINARY" ]; then
        rm -f "$TMP_BINARY"
    fi
    if [ -n "$STAGED_BINARY" ] && [ -f "$STAGED_BINARY" ]; then
        rm -f "$STAGED_BINARY"
    fi
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Binario: prebuilt o build desde código fuente
# ---------------------------------------------------------------------------
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

# ---------------------------------------------------------------------------
# Directorio de instalación (por defecto: $GOPATH/bin)
# ---------------------------------------------------------------------------
GOPATH_DIR="${GOPATH:-}"
if [ -z "$GOPATH_DIR" ] && command -v go >/dev/null 2>&1; then
    GOPATH_DIR="$(go env GOPATH 2>/dev/null || echo "")"
fi
if [ -z "$GOPATH_DIR" ]; then
    GOPATH_DIR="$HOME/go"
fi
# GOPATH puede tener varias rutas separadas por ':' -> usar la primera
GOPATH_DIR="${GOPATH_DIR%%:*}"
DEFAULT_BIN_DIR="$GOPATH_DIR/bin"

# Quita comillas, expande ~, hace la ruta absoluta y elimina '/' final
normalize_dir() {
    local d="$1"
    d="${d%\"}"; d="${d#\"}"; d="${d%\'}"; d="${d#\'}"
    case "$d" in
        "~")   d="$HOME" ;;
        "~/"*) d="$HOME/${d:2}" ;;
    esac
    case "$d" in
        /*) ;;
        *)  d="$PWD/$d" ;;
    esac
    while [ "$d" != "/" ] && [ "${d%/}" != "$d" ]; do d="${d%/}"; done
    printf '%s' "$d"
}

# Crea el directorio si no existe y verifica que se pueda escribir
prepare_dir() {
    mkdir -p "$1" 2>/dev/null && [ -w "$1" ]
}

BIN_DIR=""
if [ -n "$INSTALL_DIR_ARG" ]; then
    BIN_DIR="$(normalize_dir "$INSTALL_DIR_ARG")"
    prepare_dir "$BIN_DIR" || fail "Cannot create or write to $BIN_DIR."
elif [ -t 0 ]; then
    echo ""
    while true; do
        GOT_INPUT=1
        read -rp "Install directory (press Enter for default: $DEFAULT_BIN_DIR): " INPUT_DIR || GOT_INPUT=0
        if [ "$GOT_INPUT" -eq 0 ] || [ -z "${INPUT_DIR//[[:space:]]/}" ]; then
            CANDIDATE="$DEFAULT_BIN_DIR"
        else
            CANDIDATE="$(normalize_dir "$INPUT_DIR")"
        fi
        if prepare_dir "$CANDIDATE"; then
            BIN_DIR="$CANDIDATE"
            break
        fi
        [ "$GOT_INPUT" -eq 1 ] || fail "Cannot create or write to $CANDIDATE."
        warn "Cannot create or write to $CANDIDATE. Try another path."
    done
else
    # Sin terminal interactiva (pipe, CI): usar la ruta por defecto
    BIN_DIR="$DEFAULT_BIN_DIR"
    prepare_dir "$BIN_DIR" || fail "Cannot create or write to $BIN_DIR."
fi
info "Install directory: $BIN_DIR"

TARGET="$BIN_DIR/mova"

# Aviso si hay otra copia de mova en el PATH que podría tapar a la nueva
EXISTING="$(command -v mova 2>/dev/null || true)"
if [ -n "$EXISTING" ] && [ "$EXISTING" != "$TARGET" ]; then
    warn "Another mova was found at $EXISTING. It may take precedence over $TARGET depending on your PATH order."
fi

# ---------------------------------------------------------------------------
# Cerrar cualquier mova en ejecución (del usuario actual) antes de reemplazarlo
# ---------------------------------------------------------------------------
stop_running_mova() {
    if ! command -v pgrep >/dev/null 2>&1; then
        warn "pgrep not available — cannot check for running mova processes."
        return 0
    fi
    local uid pids pid
    uid="$(id -u)"
    pids="$(pgrep -u "$uid" -x mova 2>/dev/null || true)"
    if [ -z "$pids" ]; then
        info "No running mova process found."
        return 0
    fi
    for pid in $pids; do
        [ "$pid" = "$$" ] && continue
        info "Stopping running mova (PID $pid)..."
        kill "$pid" 2>/dev/null || true
    done
    # Esperar hasta 5s a que terminen; si no, forzar
    local i
    for i in 1 2 3 4 5 6 7 8 9 10; do
        pgrep -u "$uid" -x mova >/dev/null 2>&1 || return 0
        sleep 0.5
    done
    warn "mova did not exit gracefully — forcing termination."
    pkill -9 -u "$uid" -x mova 2>/dev/null || true
    sleep 0.5
}
stop_running_mova

# ---------------------------------------------------------------------------
# Instalar de forma atómica: copiar a un archivo temporal y renombrar.
# 'mv' crea un inodo nuevo: evita fallos de firma de código en macOS por
# sobrescribir un ejecutable en su lugar.
# ---------------------------------------------------------------------------
STAGED_BINARY="$TARGET.new.$$"
cp -f "$BUILT_BINARY" "$STAGED_BINARY" || fail "Could not copy the binary to $BIN_DIR."
chmod +x "$STAGED_BINARY"
mv -f "$STAGED_BINARY" "$TARGET" || fail "Could not replace $TARGET."
STAGED_BINARY=""

# Remueve el atributo de cuarentena de Gatekeeper si el binario fue descargado de la web
xattr -d com.apple.quarantine "$TARGET" 2>/dev/null || true

info "Installed: $TARGET"

# ---------------------------------------------------------------------------
# PATH y MOVA_PROJECT_ROOT en el archivo de perfil del shell
# ---------------------------------------------------------------------------
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
echo "  [2] Open a new Terminal window"
echo "  [3] Don't open one"
choice=""
read -rp "Choose 1-3 and press Enter (default: 1): " choice || choice=""
choice="${choice:-1}"

open_new_terminal() {
    # Comandos con comillas simples para soportar rutas con espacios
    local payload="cd '$REPO_ROOT'; export PATH='$BIN_DIR':\$PATH; export MOVA_PROJECT_ROOT='$REPO_ROOT'; mova"
    # Escapar \ y " para incrustarlo en un string de AppleScript
    payload="${payload//\\/\\\\}"
    payload="${payload//\"/\\\"}"
    osascript -e "tell application \"Terminal\" to do script \"$payload\"" \
              -e 'tell application "Terminal" to activate' >/dev/null 2>&1
}

case "$choice" in
    2)
        if open_new_terminal; then
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
        cleanup   # 'exec' reemplaza el proceso y el trap EXIT no se ejecutaría
        exec "${SHELL:-/bin/zsh}" -i
        ;;
esac

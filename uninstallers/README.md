# Mova Context — Uninstallers

Mirror of `installers/`: undoes exactly what the installer did, nothing else.

| Platform | Run | Status |
|---|---|---|
| Linux | `uninstallers/linux/uninstall.sh` | **Tested** (isolated HOME: install → uninstall left `.bashrc` byte-identical) |
| macOS | double-click `uninstallers/macos/uninstall.command` (wraps the Linux script; the macOS installer writes `~/.zshrc`, covered) | **Not run on a Mac** — use `--dry-run` first |
| Windows | double-click `uninstallers\windows\uninstall.bat` (or `uninstall.ps1`) | **Not run** (no PowerShell available to the author of this change) — use `-DryRun` first |

## What is removed
1. Running `mova` processes of the current user.
2. The binary (`mova` / `mova.exe`, plus `mova.new.*` / `mova.exe.old` leftovers).
3. **Environment:** the `MOVA_PROJECT_ROOT` line (Linux/macOS profile) or user variable (Windows), and the `PATH` entry the installer added.
   Because the default install folder is `$GOPATH/bin` (shared with other Go tools), the `PATH` entry is **kept if the folder still holds other programs** — use `--purge-path` / `-PurgePath` to drop it anyway. An empty folder is removed.
4. Temp leftovers: `mova-trace-*`, `mova-build-*`.
5. The repo folder (`MOVA_PROJECT_ROOT`) — it holds **your** `projects/` and `memory.md`, so it is **asked, never silent**: `--remove-repo` / `-RemoveRepo` deletes, `--keep-repo` / `-KeepRepo` never does. It refuses any folder that lacks `workflow.md` + `src/`, `/`, or your home.

Options: `--dir <bin dir>` · `--yes` (no questions; does not imply remove-repo) · `--dry-run`.
A script cannot change the environment of the shell that launched it: **open a new terminal** afterwards.
Not covered: anything you copied yourself (e.g. `.mcp.json` entries in other repos — see `docs/i18n/en/MCP_INTEGRATION.md` to remove them).

# Mova Context — Installers

Double-click installers for Windows, Linux, and macOS. All three do the
same thing, following the exact same install location the project's
`make install` target already uses (`$(go env GOPATH)/bin`), so mixing
this installer and the Makefile never causes a conflict:

1. Use a prebuilt binary from `dist/` (produced by `make build-all`) if
   one exists for your OS/architecture; otherwise build one from
   source with `go build` (requires [Go](https://go.dev/dl)) —
   including any new dependency the build needs (e.g. the terminal
   interface's Bubble Tea/Lip Gloss/Bubbles libraries), downloaded
   automatically, exactly like `go build` already does for every other
   dependency. No manual install step, on any platform.
2. Copy it to `$(go env GOPATH)/bin/mova` (`mova.exe` on Windows).
3. Add that folder to your `PATH` (only if it isn't already there),
   **and set `MOVA_PROJECT_ROOT` permanently** to wherever this repo
   lives (only if it isn't already set) — this is what lets `mova` work
   from *any* folder afterward, including a different drive/volume
   entirely. See "Working from any drive/location" below.
4. **Open a console already set up to use `mova`** — no separate "now
   open a terminal yourself" step. See "Ready-to-use console" below.

## Working from any drive/location

A project's `"repo"` (in its `project.json`) can be an absolute path
pointing anywhere — a different drive on Windows (`D:\my-app`), a
different mount on Linux (`/mnt/data/my-app`), a different volume on
macOS (`/Volumes/Data/my-app`) — completely independent of where this
Mova Context repo itself lives. Every part of Mova that touches project
files (focus, save, delete, jobs, chat's natural-language edits)
already resolves an absolute `"repo"` correctly.

The one thing that needs help is finding *this* repo's `workflow.md`
when you're standing inside that external folder and just type `mova`
— by default Mova searches upward from your current directory, which
won't find anything on a separate drive. That's what step 3 above
solves: the installer sets `MOVA_PROJECT_ROOT` to this repo's path,
permanently, so `mova` finds it regardless of your current directory.
This was verified end-to-end (a project with an external `"repo"`
correctly builds context, saves reports, and deletes files there, from
an unrelated working directory) — see
[COMMANDS.md § Working across different drives/locations](../docs/i18n/en/COMMANDS.md#working-across-different-drivesocations-windowslinuxmacos)
for the full explanation and a worked example.

## Ready-to-use console

After installing, every platform asks how you'd like to start using
`mova` right away:

```text
Which console would you like to open, ready to use mova?
  [1] <the platform's own console — PowerShell / this same window>  (default)
  [2] <an alternative console — CMD / a new window>
  [3] Don't open one
```

| Platform | Option 1 (default) | Option 2 |
|---|---|---|
| Windows | Opens **PowerShell** (or `pwsh` if installed), already on `PATH`, already in the Mova folder | Opens **CMD** instead |
| macOS | **Continues in this same Terminal window** (already on `PATH`) | Opens a **new** Terminal window |
| Linux | **Continues in this same terminal** (already on `PATH`) | Opens a **new** terminal window — auto-detects `$TERMINAL`, then `x-terminal-emulator`, `gnome-terminal`, `konsole`, `xfce4-terminal`, `tilix`, `alacritty`, `kitty`, or `xterm`, whichever is installed |

Either way, `mova` works immediately in that console — no need to close
and reopen a terminal for the `PATH` change to apply, since it's set
for that specific session at launch time (on top of the permanent
`PATH` change already saved for every future session).

Choosing "Don't open one" (or if no terminal emulator can be found on
Linux) falls back to the previous behavior: a short note reminding you
to open a new terminal yourself.

## Windows

Double-click **`windows\install.bat`**. It launches `install.ps1` for
you (a `.ps1` can't be double-clicked directly on a stock Windows
install — it opens in a text editor instead).

## macOS

Double-click **`macos/install.command`** in Finder. If macOS shows a
Gatekeeper warning the first time ("cannot be opened because it is
from an unidentified developer"), right-click the file → **Open** once
to approve it, then double-click normally from then on.

## Linux

Double-click **`linux/install.sh`** — most file managers (GNOME
Files/Nautilus, Dolphin, Thunar...) will offer to **Run** it since it
ships pre-marked executable. If your file manager opens it as a text
file instead, run it from a terminal:

```bash
./installers/linux/install.sh
```

## Building prebuilt binaries first (optional)

```bash
make build-all
```

Produces `dist/mova-windows-amd64.exe`, `dist/mova-linux-amd64`,
`dist/mova-linux-arm64`, `dist/mova-macos-amd64`, and
`dist/mova-macos-arm64`. Running the installer for your platform
afterward just copies the matching one — no compiler needed on the
machine you're installing to.

## Generating installers for a new version

The installer scripts themselves (`install.bat`/`install.ps1`,
`install.command`, `install.sh`) are **version-independent** — they
never hardcode a version number or a list of files, so a new Mova
Context release does not require editing or regenerating them. What
changes between releases is only the source tree and, optionally, the
prebuilt binaries in `dist/`:

1. **Ship source only** (simplest): package the repository as-is (a
   zip, like this deliverable). The installer scripts already fall
   back to `go build` from source automatically when `dist/` is
   absent, downloading any new dependency exactly as it did for this
   change. No extra step for a new version.
2. **Ship prebuilt binaries too** (faster install, no Go toolchain
   required on the target machine): run `make build-all` from the repo
   root right before packaging the release. It cross-compiles for all
   five OS/architecture combinations above and drops them in `dist/`,
   which the installer scripts already check first. Include `dist/` in
   the release zip alongside `installers/`.
3. **A new command, dependency, or config file was added** (like the
   Job Engine, the multiagent orchestrator, or the terminal interface
   in earlier changes): still no installer change needed — steps 1-3
   in "What it does" above are generic (locate/build a binary, copy it,
   update `PATH`), so any new Go dependency is picked up automatically
   the next time `go build` runs, and any new config file (e.g.
   `config/log/logging.json`) ships simply by being part of the
   repository tree that gets zipped and installed alongside `mova`.

In short: **regenerate `dist/` with `make build-all` before a release
if you want prebuilt binaries; the installer scripts themselves never
need to change.**

#!/usr/bin/env bash
# Builds the taskd TUI and puts it on your PATH, so `taskd` works from any
# terminal. Linux (and macOS) counterpart of install-taskd.ps1.
#
#   1. Runs the Go tests (skip with --skip-tests).
#   2. Builds cmd/taskd into ~/.local/bin/taskd (override with TASKD_INSTALL_DIR).
#   3. If that folder isn't on your PATH yet, adds one line to your shell's rc
#      file, once. No sudo needed.
#
# Run it again after pulling new code to rebuild.
#
# Usage:
#   ./scripts/install-taskd.sh [--skip-tests] [--uninstall]

set -euo pipefail

INSTALL_DIR="${TASKD_INSTALL_DIR:-$HOME/.local/bin}"
BIN="$INSTALL_DIR/taskd"
MARKER="# added by taskd install-taskd.sh"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

skip_tests=0
uninstall=0
for arg in "$@"; do
    case "$arg" in
        --skip-tests) skip_tests=1 ;;
        --uninstall)  uninstall=1 ;;
        -h|--help)    sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *)            echo "unknown option: $arg (try --help)" >&2; exit 2 ;;
    esac
done

# The rc file of the shell you log in with. Other shells get a hint instead.
rc_file() {
    case "$(basename "${SHELL:-}")" in
        bash) echo "$HOME/.bashrc" ;;
        zsh)  echo "${ZDOTDIR:-$HOME}/.zshrc" ;;
        *)    echo "" ;;
    esac
}

# True if a directory is already one of the entries in $PATH.
on_path() {
    case ":$PATH:" in
        *":$1:"*) return 0 ;;
        *)        return 1 ;;
    esac
}

if [[ $uninstall -eq 1 ]]; then
    if [[ -f "$BIN" ]]; then
        rm -f "$BIN"
        echo "Removed $BIN"
    fi
    rc="$(rc_file)"
    if [[ -n "$rc" && -f "$rc" ]] && grep -qF "$MARKER" "$rc"; then
        # Delete only the line we added (it ends with the marker).
        grep -vF "$MARKER" "$rc" > "$rc.taskd.tmp" && cat "$rc.taskd.tmp" > "$rc" && rm -f "$rc.taskd.tmp"
        echo "Removed the PATH line from $rc"
    fi
    echo "taskd uninstalled."
    exit 0
fi

command -v go >/dev/null 2>&1 || { echo "Go is not on PATH. Install Go 1.26 or newer first." >&2; exit 1; }
[[ -f "$REPO_ROOT/cmd/taskd/main.go" ]] || {
    echo "cmd/taskd/main.go not found under $REPO_ROOT. Is this script in the repo's scripts folder?" >&2
    exit 1
}

cd "$REPO_ROOT"

if [[ $skip_tests -eq 0 ]]; then
    echo "Running tests..."
    go test ./... || { echo "Tests failed, not installing." >&2; exit 1; }
fi

mkdir -p "$INSTALL_DIR"

# Build to a temp file next to the target, then move it into place. The
# move is atomic, so an already running taskd is never half-overwritten.
echo "Building $BIN ..."
tmp="$(mktemp "$INSTALL_DIR/.taskd.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
# -trimpath keeps your local folder paths out of the binary.
# -ldflags "-s -w" drops debug symbols, so the binary is smaller.
go build -trimpath -ldflags '-s -w' -o "$tmp" ./cmd/taskd
chmod 0755 "$tmp"
mv -f "$tmp" "$BIN"
trap - EXIT

if on_path "$INSTALL_DIR"; then
    echo "$INSTALL_DIR is already on your PATH."
else
    rc="$(rc_file)"
    if [[ -z "$rc" ]]; then
        echo "Add $INSTALL_DIR to your PATH in your shell's config file, then open a new terminal."
    elif [[ -f "$rc" ]] && grep -qF "$MARKER" "$rc"; then
        echo "$rc already adds $INSTALL_DIR to PATH. Open a new terminal or run: source $rc"
    else
        # The format string is in single quotes, so $PATH is written as-is
        # and expands each time the rc file runs, not now.
        printf '\nexport PATH="%s:$PATH"  %s\n' "$INSTALL_DIR" "$MARKER" >> "$rc"
        echo "Added $INSTALL_DIR to PATH in $rc. Open a new terminal or run: source $rc"
    fi
fi

echo
echo "Done. Type 'taskd' to start the TUI."

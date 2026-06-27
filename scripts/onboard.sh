#!/usr/bin/env bash
# Hatch local onboarding: build the binaries, optionally install them on PATH,
# and spin up a runnable demo workspace (using the mock agent) so you can try
# everything locally — no real agent CLI required.
#
#   ./scripts/onboard.sh            # build + demo in ./demo-workspace
#   ./scripts/onboard.sh --install  # also `go install` hatch + hatch-mock
#   ./scripts/onboard.sh --no-demo  # just build (+install)
#   ./scripts/onboard.sh --demo DIR # demo in DIR
set -euo pipefail

HATCH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO_DIR="$HATCH_DIR/demo-workspace"
DO_INSTALL=0
DO_DEMO=1

while [ $# -gt 0 ]; do
  case "$1" in
    --install) DO_INSTALL=1 ;;
    --no-demo) DO_DEMO=0 ;;
    --demo) shift; DEMO_DIR="$1" ;;
    -h|--help) sed -n '2,9p' "$0"; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 1 ;;
  esac
  shift
done

step() { printf '\n\033[1;34m▶ %s\033[0m\n' "$1"; }

step "Checking Go toolchain"
if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed. Get it at https://go.dev/dl/ (need 1.24+)." >&2
  exit 1
fi
go version

step "Building hatch"
cd "$HATCH_DIR"
make build
BIN="$HATCH_DIR/bin"
echo "built: $BIN/hatch"

if [ "$DO_INSTALL" = "1" ]; then
  step "Installing to GOPATH/bin (go install)"
  make install
  echo "installed hatch to $(go env GOPATH)/bin"
  echo "ensure that dir is on your PATH:  export PATH=\"\$(go env GOPATH)/bin:\$PATH\""
fi

# Use the freshly built binaries for the demo regardless of install.
export PATH="$BIN:$PATH"

if [ "$DO_DEMO" = "1" ]; then
  step "Creating demo workspace at $DEMO_DIR"
  rm -rf "$DEMO_DIR"
  mkdir -p "$DEMO_DIR"
  cd "$DEMO_DIR"
  git init -q
  hatch init --local -w scrum >/dev/null   # --local: workspace của riêng demo repo này

  step "Compiling the SSOT (protocol + per-agent MCP registration)"
  hatch compile
  echo
  echo "  → CLAUDE.md / AGENTS.md / GEMINI.md carry the chat protocol + workflow."
  echo "  → .mcp.json registers the 'hatch' MCP server for Claude Code."

  step "Opening a task thread (a thread = a task)"
  # In real use a coding agent does this through the Hatch MCP server; here we
  # post as a human operator so the demo needs no agent CLI.
  hatch msg --from human:operator -c '#export-csv' \
    "@codex hãy stream CSV để giảm bộ nhớ" >/dev/null
  echo "  posted to #export-csv"

  step "Read-only status (threads + roster)"
  hatch status

  step "Read the thread"
  hatch thread '#export-csv' | sed 's/^/  /'

  printf '\n\033[1;32m✔ Demo ready.\033[0m  Workspace: %s\n' "$DEMO_DIR"
  cat <<EOF

How it works for real: open a coding agent (e.g. Claude Code) IN this workspace.
The compiled CLAUDE.md + .mcp.json wire it to the shared chat — it opens a
thread per task, @mentions teammates, and records knowledge, all over MCP.

Watch the squad (read-only, from that dir, with $BIN on PATH):
  hatch board                      # mission control: threads + chat + ledger
  hatch chat                       # Slack-style conversation view
  hatch search streaming           # recall messages
  hatch mcp --as claude-code       # the MCP server an agent connects to

Docs: $HATCH_DIR/docs/overview.md, $HATCH_DIR/docs/20-embedded-harness-pivot.md
EOF
fi

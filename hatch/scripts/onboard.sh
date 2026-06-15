#!/usr/bin/env bash
# Hatch local onboarding: build the binaries, optionally install them on PATH,
# and spin up a runnable demo workspace (using the mock agent) so you can try
# everything locally — no real agent CLI required.
#
#   ./scripts/onboard.sh            # build + demo in ./.hatch-demo
#   ./scripts/onboard.sh --install  # also `go install` hatch + hatch-mock
#   ./scripts/onboard.sh --no-demo  # just build (+install)
#   ./scripts/onboard.sh --demo DIR # demo in DIR
set -euo pipefail

HATCH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO_DIR="$HATCH_DIR/.hatch-demo"
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

step "Building hatch + hatch-mock"
cd "$HATCH_DIR"
make build
BIN="$HATCH_DIR/bin"
echo "built: $BIN/hatch, $BIN/hatch-mock"

if [ "$DO_INSTALL" = "1" ]; then
  step "Installing to GOPATH/bin (go install)"
  make install
  go build -o "$(go env GOPATH)/bin/hatch-mock" ./cmd/hatch-mock
  echo "installed hatch + hatch-mock to $(go env GOPATH)/bin"
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
  hatch init -w scrum >/dev/null
  # Add a dedicated `mock` agent so the demo runs at ZERO token cost — WITHOUT
  # changing the real agents (claude/codex/agy/kiro stay exactly as scaffolded).
  awk '
    /^agents:/ && !ins { print; print "  - id: mock"; print "    kind: mock"; print "    roles: [implementer, tester]"; ins=1; next }
    { print }
  ' .hatch/registry.yaml > .hatch/registry.yaml.tmp && mv .hatch/registry.yaml.tmp .hatch/registry.yaml
  hatch compile >/dev/null 2>&1

  step "Seeding a ticket and running the mock agent (real agents untouched)"
  hatch ticket new --title "Export báo cáo ra CSV" --role implementer --priority P1 >/dev/null
  hatch ticket claim T-001 --agent mock --why "demo" >/dev/null
  hatch run T-001 --agent mock >/dev/null 2>&1 || true

  step "Board"
  hatch status
  step "Run transcript (hatch logs)"
  hatch logs T-001 | sed 's/^/  /'
  step "Cost + perf"
  hatch cost T-001
  hatch perf 2>/dev/null | sed 's/^/  /' | head -4

  printf '\n\033[1;32m✔ Demo ready.\033[0m  Workspace: %s\n' "$DEMO_DIR"
  cat <<EOF

Try next (from that dir, with $BIN on PATH):
  hatch board                      # the multi-pane TUI (board + live output + activity)
  hatch ticket move T-001 --to review --by implementer --agent codex --why done --skip-gates
  hatch msg --from codex -c '#design' "@claude-code chốt streaming nhé"
  hatch convene --topic "Thiết kế export" --agents codex,claude-code --rounds 1
  hatch workload ; hatch budget

Docs: $HATCH_DIR/docs/overview.md
EOF
fi

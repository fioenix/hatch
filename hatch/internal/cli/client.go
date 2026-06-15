package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/config"
)

// resolveClientKind maps a user-facing --client alias to a registry agent kind.
func resolveClientKind(alias string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(alias)) {
	case "cc", "claude", "claude-code", "claudecode":
		return "claude", true
	case "codex":
		return "codex", true
	case "agy", "antigravity":
		return "agy", true
	case "kiro", "kiro-cli":
		return "kiro", true
	default:
		return "", false
	}
}

// agentIDForKind returns the first registry agent of a kind (the identity the
// MCP server posts as for that client).
func agentIDForKind(ws *config.Workspace, kind string) (string, bool) {
	for _, a := range ws.Registry.Agents {
		if a.Kind == kind {
			return a.ID, true
		}
	}
	return "", false
}

// setupClient wires one client so it reaches the Hatch chat + KB over MCP under
// its own identity (`hatch mcp --as <id>`). Repo-local configs are written in
// place; for clients whose config lives in $HOME (Codex, agy) it uses the
// client's own CLI when available, else merges/points at the home file.
// Running `hatch init --client <x>` is the user's explicit consent to set up x.
func setupClient(cmd *cobra.Command, ws *config.Workspace, repoRoot, alias string, dryRun bool) error {
	out := cmd.OutOrStdout()
	kind, ok := resolveClientKind(alias)
	if !ok {
		return fmt.Errorf("unknown client %q (use: cc | codex | agy | kiro)", alias)
	}
	id, ok := agentIDForKind(ws, kind)
	if !ok {
		return fmt.Errorf("no %s-kind agent in registry.yaml — add one, then re-run", kind)
	}
	args := []string{"mcp", "--as", id}

	switch kind {
	case "claude":
		p := filepath.Join(repoRoot, ".mcp.json")
		if err := writeServerJSON(p, id, dryRun); err != nil {
			return err
		}
		say(out, dryRun, "claude: project MCP config %s → server 'hatch' (--as %s)", rel(repoRoot, p), id)
		fmt.Fprintf(out, "  để cài plugin (skill + /hatch) cho Claude Code:\n")
		fmt.Fprintf(out, "    /plugin marketplace add fioenix/overclaud\n")
		fmt.Fprintf(out, "    /plugin install hatch@hatch\n")
		fmt.Fprintf(out, "  (project-scope .mcp.json ở trên đã đủ để Claude Code nạp MCP server.)\n")

	case "kiro":
		p := filepath.Join(repoRoot, ".kiro", "settings", "mcp.json")
		if err := writeServerJSON(p, id, dryRun); err != nil {
			return err
		}
		say(out, dryRun, "kiro: project MCP config %s → server 'hatch' (--as %s)", rel(repoRoot, p), id)

	case "codex":
		// Codex owns ~/.codex/config.toml; let its own CLI edit it.
		if path, err := exec.LookPath("codex"); err == nil && !dryRun {
			cargs := append([]string{"mcp", "add", "hatch", "--"}, append([]string{"hatch"}, args...)...)
			c := exec.Command(path, cargs...)
			if b, err := c.CombinedOutput(); err != nil {
				fmt.Fprintf(out, "codex: `codex mcp add` lỗi (%v): %s\n", err, strings.TrimSpace(string(b)))
				fmt.Fprintf(out, "  dán tay khối ở %s vào ~/.codex/config.toml\n", rel(repoRoot, filepath.Join(repoRoot, ".hatch", "mcp", id+".codex.toml")))
			} else {
				fmt.Fprintf(out, "codex: đã `codex mcp add hatch -- hatch mcp --as %s` (→ ~/.codex/config.toml)\n", id)
			}
		} else {
			say(out, dryRun, "codex: chạy `codex mcp add hatch -- hatch mcp --as %s`", id)
			fmt.Fprintf(out, "  (hoặc dán %s vào ~/.codex/config.toml)\n", rel(repoRoot, filepath.Join(repoRoot, ".hatch", "mcp", id+".codex.toml")))
		}

	case "agy":
		// Antigravity CLI loads MCP from a HOME-level JSON (mcpServers shape).
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("agy: cannot resolve home dir: %w", err)
		}
		homeCfg := filepath.Join(home, ".gemini", "config", "mcp_config.json")
		if err := writeServerJSON(homeCfg, id, dryRun); err != nil {
			return err
		}
		say(out, dryRun, "agy: home MCP config %s → server 'hatch' (--as %s)", homeCfg, id)
		// Workspace-scope copy too (some agy versions read .agents/mcp_config.json).
		wsCfg := filepath.Join(repoRoot, ".agents", "mcp_config.json")
		if err := writeServerJSON(wsCfg, id, dryRun); err != nil {
			return err
		}
		say(out, dryRun, "agy: workspace MCP config %s (dự phòng)", rel(repoRoot, wsCfg))
	}
	return nil
}

// writeServerJSON merges {"mcpServers":{"hatch":{command,args}}} into a JSON
// config, preserving any other servers/keys. Creates parents if needed.
func writeServerJSON(path, agentID string, dryRun bool) error {
	root := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &root); err != nil {
			return fmt.Errorf("%s is not valid JSON (edit or remove it): %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["hatch"] = map[string]any{
		"command": "hatch",
		"args":    []any{"mcp", "--as", agentID},
	}
	root["mcpServers"] = servers
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func say(out io.Writer, dryRun bool, format string, a ...any) {
	prefix := "✓ "
	if dryRun {
		prefix = "[dry-run] "
	}
	fmt.Fprintf(out, prefix+format+"\n", a...)
}

func rel(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}

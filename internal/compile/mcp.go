package compile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fioenix/hatch/internal/config"
)

// mcpServerName is the key the Hatch server registers under in every agent's
// MCP config.
const mcpServerName = "hatch"

// mcpBinary is the command agents launch to reach the shared chat + KB. It is
// expected on PATH; agents speak MCP to it over stdio.
const mcpBinary = "hatch"

// writeMCPConfigs registers the `hatch mcp --as <agent>` server for the agents
// whose only wiring point is the repo, so each reaches the shared chat + KB
// under its own identity.
//
// Only Kiro gets a repo file (`.kiro/settings/mcp.json`) — it reads MCP solely
// from the project. Claude is wired by its user-global plugin (installed via
// `hatch setup`), and Codex/agy by their $HOME config (also `hatch setup`); for
// those two we still drop a paste-ready snippet under `.hatch/mcp/` as a manual
// fallback. Hatch never edits files outside the repo.
func writeMCPConfigs(ws *config.Workspace, repoRoot string) ([]string, error) {
	var written []string
	for _, a := range ws.Registry.Agents {
		switch a.Kind {
		case "kiro":
			p := filepath.Join(repoRoot, ".kiro", "settings", "mcp.json")
			if err := mergeJSONServer(p, a.ID); err != nil {
				return written, err
			}
			written = append(written, p)
		case "codex":
			// Snippets are reference docs, kept with the SSOT (not scattered into
			// the working repo, which would look like a local .hatch override).
			p := filepath.Join(ws.Layout.MCPSnippets(), a.ID+".codex.toml")
			if err := writeFile(p, codexSnippet(a.ID)); err != nil {
				return written, err
			}
			written = append(written, p)
		case "agy", "antigravity":
			p := filepath.Join(ws.Layout.MCPSnippets(), a.ID+".agy.md")
			if err := writeFile(p, agySnippet(a.ID)); err != nil {
				return written, err
			}
			written = append(written, p)
		}
	}
	return written, nil
}

// serverEntry is the MCP stdio launch spec shared by the JSON configs.
func serverEntry(agentID string) map[string]any {
	return map[string]any{
		"command": mcpBinary,
		"args":    []any{"mcp", "--as", agentID},
	}
}

// mergeJSONServer sets mcpServers.hatch in a JSON MCP config, preserving any
// other servers the user added. It creates the file (and parents) if absent.
func mergeJSONServer(path, agentID string) error {
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
	servers[mcpServerName] = serverEntry(agentID)
	root["mcpServers"] = servers

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, string(out)+"\n")
}

func codexSnippet(agentID string) string {
	return fmt.Sprintf(`# Hatch MCP cho Codex — dán khối này vào ~/.codex/config.toml
# (Codex đọc config ở $CODEX_HOME, ngoài repo; Hatch không tự sửa file đó.)

[mcp_servers.%s]
command = "%s"
args = ["mcp", "--as", "%s"]
`, mcpServerName, mcpBinary, agentID)
}

func agySnippet(agentID string) string {
	entry := map[string]any{
		"mcpServers": map[string]any{mcpServerName: serverEntry(agentID)},
	}
	body, _ := json.MarshalIndent(entry, "", "  ")
	return fmt.Sprintf("# Hatch MCP cho agy (Antigravity CLI — KHÁC Gemini CLI legacy)\n\n"+
		"Antigravity CLI nạp MCP server CHỈ từ file HOME-level riêng (không inline trong settings):\n"+
		"  • hiện hành: `~/.gemini/config/mcp_config.json`\n"+
		"  • cũ: `~/.gemini/antigravity-cli/mcp_config.json` (nay là symlink tới file trên)\n"+
		"Project-local `.antigravitycli/mcp_config.json` bị phát hiện nhưng BỎ QUA (issue #60).\n"+
		"Cách nhanh: `hatch init --client agy`. Hoặc trộn khối `mcpServers` dưới đây vào file HOME:\n\n"+
		"```json\n%s\n```\n", string(body))
}

// writeFile writes content, creating parent dirs.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

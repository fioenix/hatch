package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/fioenix/hatch/internal/config"
	"github.com/fioenix/hatch/internal/mcpserver"
)

func newMCPCmd() *cobra.Command {
	var as string
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the Hatch MCP server (chat + KB tools) over stdio for a coding agent",
		Long: "Expose the shared chat (bus) and knowledge base to an MCP-capable agent.\n" +
			"Each agent runs its own instance with its identity, e.g.\n  hatch mcp --as claude-code\n" +
			"If --as is omitted, it resolves from $HATCH_AGENT, else the first\n" +
			"Claude-kind agent in the registry (so the Claude plugin needs no config).\n" +
			"Wire it into the agent's MCP config (claude .mcp.json / codex config.toml / .kiro/settings/mcp.json).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if as == "" {
				as = resolveIdentity(ws)
			}
			if as == "" {
				return fmt.Errorf("could not resolve identity: pass --as <agent-id>, set $HATCH_AGENT, or add a claude-kind agent to registry.yaml")
			}
			if _, ok := ws.Registry.AgentByID(as); !ok {
				return fmt.Errorf("unknown agent %q (not in registry.yaml)", as)
			}
			srv := mcpserver.New(ws, as, Version)
			// Stdio transport: the agent launches this process and speaks MCP on
			// stdin/stdout, so nothing may be printed to stdout outside the protocol.
			return srv.Run(context.Background(), &mcp.StdioTransport{})
		},
	}
	cmd.Flags().StringVar(&as, "as", "", "agent id this server acts as (default: $HATCH_AGENT or first claude-kind agent)")
	return cmd
}

// resolveIdentity picks the agent this MCP server posts as when --as is omitted:
// $HATCH_AGENT if set, otherwise the first Claude-kind agent in the registry
// (the agent the Claude plugin runs inside).
func resolveIdentity(ws *config.Workspace) string {
	if env := os.Getenv("HATCH_AGENT"); env != "" {
		return env
	}
	for _, a := range ws.Registry.Agents {
		if a.Kind == "claude" {
			return a.ID
		}
	}
	return ""
}

package cli

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/mcpserver"
)

func newMCPCmd() *cobra.Command {
	var as string
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the Hatch MCP server (chat + KB tools) over stdio for a coding agent",
		Long: "Expose the shared chat (bus) and knowledge base to an MCP-capable agent.\n" +
			"Each agent runs its own instance with its identity, e.g.\n  hatch mcp --as claude-code\n" +
			"Wire it into the agent's MCP config (claude .mcp.json / codex config.toml / .kiro/settings/mcp.json).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if as == "" {
				return fmt.Errorf("--as <agent-id> is required (the identity this MCP server posts as)")
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
	cmd.Flags().StringVar(&as, "as", "", "agent id this server acts as (required)")
	return cmd
}

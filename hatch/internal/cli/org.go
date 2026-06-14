package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Print the org chart (reporting lines) + delegation-of-authority",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			// roots = roles with no manager; print the tree depth-first.
			children := map[string][]model.Role{}
			var roots []model.Role
			for _, r := range ws.Registry.Roles {
				if r.ReportsTo == "" {
					roots = append(roots, r)
				} else {
					children[r.ReportsTo] = append(children[r.ReportsTo], r)
				}
			}
			sortRoles(roots)
			fmt.Fprintln(out, "Org chart:")
			for _, r := range roots {
				printRole(out, ws, children, r, 0)
			}
			return nil
		},
	}
	return cmd
}

func printRole(out interface{ Write([]byte) (int, error) }, ws *config.Workspace, children map[string][]model.Role, r model.Role, depth int) {
	indent := strings.Repeat("  ", depth)
	agents := agentIDs(ws.Registry.AgentsForRole(r.ID))
	auth := ""
	if r.Authority != nil {
		var parts []string
		if r.Authority.CanApprove {
			parts = append(parts, "approve")
		}
		if r.Authority.BudgetAuthorityUSD > 0 {
			parts = append(parts, fmt.Sprintf("budget≤$%.0f", r.Authority.BudgetAuthorityUSD))
		}
		if len(r.Authority.DecisionScope) > 0 {
			parts = append(parts, "scope:"+strings.Join(r.Authority.DecisionScope, "/"))
		}
		if len(parts) > 0 {
			auth = "  [" + strings.Join(parts, " · ") + "]"
		}
	}
	who := "—"
	if len(agents) > 0 {
		who = strings.Join(agents, ", ")
	}
	fmt.Fprintf(out, "%s• %s (%s)%s\n", indent, r.ID, who, auth)
	kids := children[r.ID]
	sortRoles(kids)
	for _, c := range kids {
		printRole(out, ws, children, c, depth+1)
	}
}

func sortRoles(rs []model.Role) {
	sort.Slice(rs, func(i, j int) bool { return rs[i].ID < rs[j].ID })
}

func agentIDs(as []model.Agent) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.ID
	}
	return out
}

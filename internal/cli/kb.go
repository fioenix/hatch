package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/hatch/internal/config"
	"github.com/fioenix/hatch/internal/model"
	"github.com/fioenix/hatch/internal/store"
)

func newKBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kb",
		Short: "Shared Knowledge Base: add, query, reindex, link (Obsidian-aware)",
	}
	cmd.AddCommand(newKBAddCmd(), newKBQueryCmd(), newKBIndexCmd(),
		newKBLinkCmd(), newKBBacklinksCmd(), newKBGraphCmd(), newKBOpenCmd())
	return cmd
}

// kbFor builds a KB configured from the registry (vault location + wikilinks).
func kbFor(ws *config.Workspace) *store.KB {
	cfg := ws.Registry.KB
	wikilinks := cfg.Wikilinks || cfg.Mode == "obsidian"
	if cfg.Vault != "" {
		root := cfg.Vault
		if !filepath.IsAbs(root) {
			root = filepath.Join(ws.Layout.RepoRoot(), root)
		}
		return store.NewKBVault(ws.Layout, root, wikilinks)
	}
	kb := store.NewKB(ws.Layout)
	kb.Wikilinks = wikilinks
	return kb
}

func newKBAddCmd() *cobra.Command {
	var typ, title, tags, related, author, body string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a KB entry (body from --body or stdin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			switch typ {
			case model.KBDecision, model.KBDomain, model.KBLearning:
			default:
				return fmt.Errorf("--type must be one of: decision, domain, learning")
			}
			if body == "" {
				if in, _ := io.ReadAll(cmd.InOrStdin()); len(in) > 0 {
					body = string(in)
				}
			}
			kb := kbFor(ws)
			id := kb.NextID(typ)
			entry := model.KBEntry{
				ID:      id,
				Type:    typ,
				Title:   title,
				Tags:    splitCSV(tags),
				Related: splitCSV(related),
				Author:  author,
				Created: time.Now().Format(time.RFC3339),
				Body:    strings.TrimSpace(body),
			}
			if typ == model.KBDecision {
				entry.Status = "accepted"
			}
			p, err := kb.Add(entry)
			if err != nil {
				return err
			}
			if err := kb.RebuildIndex(); err != nil {
				return err
			}
			_ = store.NewLedger(ws.Layout).Append(model.Entry{
				Agent: orHuman(author), Ticket: "-", Action: model.ActNote,
				Why: "KB add: " + title, Note: entry.Path,
			})
			fmt.Fprintf(cmd.OutOrStdout(), "Added %s → %s\n", id, p)
			return nil
		},
	}
	cmd.Flags().StringVar(&typ, "type", "learning", "decision | domain | learning")
	cmd.Flags().StringVar(&title, "title", "", "entry title (required)")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	cmd.Flags().StringVar(&related, "related", "", "comma-separated related ids/paths")
	cmd.Flags().StringVar(&author, "author", "", "author (agent id or human:<name>)")
	cmd.Flags().StringVar(&body, "body", "", "entry body (else read from stdin)")
	return cmd
}

func newKBQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [tags...]",
		Short: "List KB entries, optionally filtered by tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			entries, err := kbFor(ws).Query(args)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no matching KB entries")
				return nil
			}
			for _, e := range entries {
				tags := ""
				if len(e.Tags) > 0 {
					tags = " [" + strings.Join(e.Tags, ", ") + "]"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-10s %-9s %s%s\n  %s\n", e.ID, e.Type, e.Title, tags, e.Path)
			}
			return nil
		},
	}
	return cmd
}

func newKBIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Rebuild kb/index.md from current entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if err := kbFor(ws).RebuildIndex(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "rebuilt kb/index.md")
			return nil
		},
	}
	return cmd
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func orHuman(a string) string {
	if a == "" {
		return "human:operator"
	}
	return a
}

func newKBLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link <from-id> <to-id>",
		Short: "Link two KB notes (adds to `related`; renders as [[wikilink]])",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			kb := kbFor(ws)
			if err := kb.Link(args[0], args[1]); err != nil {
				return err
			}
			_ = kb.RebuildIndex()
			fmt.Fprintf(cmd.OutOrStdout(), "%s → %s linked\n", args[0], args[1])
			return nil
		},
	}
}

func newKBBacklinksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backlinks <note>",
		Short: "List notes that link to a note (id or name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			bl, err := kbFor(ws).Backlinks(args[0])
			if err != nil {
				return err
			}
			if len(bl) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no backlinks")
			}
			for _, id := range bl {
				fmt.Fprintln(cmd.OutOrStdout(), id)
			}
			return nil
		},
	}
}

func newKBGraphCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "graph",
		Short: "Print the KB link graph (note → linked notes)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			g, err := kbFor(ws).Graph()
			if err != nil {
				return err
			}
			if len(g) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no links yet")
			}
			for from, tos := range g {
				fmt.Fprintf(cmd.OutOrStdout(), "%s → %s\n", from, strings.Join(tos, ", "))
			}
			return nil
		},
	}
}

func newKBOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <note>",
		Short: "Print an obsidian:// URI to open a note in the app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), kbFor(ws).ObsidianURI(args[0]))
			return nil
		},
	}
}

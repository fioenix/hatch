package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newKBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kb",
		Short: "Shared Knowledge Base: add, query, reindex",
	}
	cmd.AddCommand(newKBAddCmd(), newKBQueryCmd(), newKBIndexCmd())
	return cmd
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
			kb := store.NewKB(ws.Layout)
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
			entries, err := store.NewKB(ws.Layout).Query(args)
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
			if err := store.NewKB(ws.Layout).RebuildIndex(); err != nil {
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

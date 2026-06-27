package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/fioenix/hatch/internal/docs"
)

func newDocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doc",
		Short: "Scaffold and lint documents against per-project templates/specs",
	}
	cmd.AddCommand(newDocTypesCmd(), newDocNewCmd(), newDocLintCmd())
	return cmd
}

func newDocTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List available document types + their framework",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			ts, err := docs.Types(ws.Layout)
			if err != nil {
				return err
			}
			if len(ts) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no doc templates (.hatch/templates/docs/)")
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "TYPE\tFRAMEWORK\tREQUIRED SECTIONS")
			for _, t := range ts {
				fmt.Fprintf(tw, "%s\t%s\t%v\n", t.Type, t.Framework, t.RequiredSections)
			}
			tw.Flush()
			return nil
		},
	}
}

func newDocNewCmd() *cobra.Command {
	var title, out string
	cmd := &cobra.Command{
		Use:   "new <type>",
		Short: "Scaffold a document from its template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			tmpls, err := docs.Load(ws.Layout)
			if err != nil {
				return err
			}
			t, ok := tmpls[args[0]]
			if !ok {
				return fmt.Errorf("unknown doc type %q (see `hatch doc types`)", args[0])
			}
			content, err := docs.New(t, title)
			if err != nil {
				return err
			}
			if out == "" {
				out = args[0] + "-" + docs.Slug(title) + ".md"
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil && filepath.Dir(out) != "." {
				return err
			}
			if err := os.WriteFile(out, content, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s (%s)\n", out, t.Framework)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "document title (required)")
	cmd.Flags().StringVar(&out, "out", "", "output path (default: <type>-<slug>.md)")
	return cmd
}

func newDocLintCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "lint [file]",
		Short: "Check a document has the sections/frontmatter its spec requires",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			tmpls, err := docs.Load(ws.Layout)
			if err != nil {
				return err
			}
			var files []string
			if all {
				_ = filepath.Walk(ws.Layout.RepoRoot(), func(p string, fi os.FileInfo, err error) error {
					if err == nil && !fi.IsDir() && filepath.Ext(p) == ".md" {
						files = append(files, p)
					}
					return nil
				})
			} else {
				if len(args) != 1 {
					return fmt.Errorf("provide a file or use --all")
				}
				files = args
			}
			out := cmd.OutOrStdout()
			total := 0
			for _, f := range files {
				raw, err := os.ReadFile(f)
				if err != nil {
					return err
				}
				dt, probs, err := docs.Lint(raw, tmpls)
				if err != nil {
					return err
				}
				if all && dt == "" {
					continue // not a typed document; skip in bulk mode
				}
				if len(probs) == 0 {
					fmt.Fprintf(out, "✓ %s (%s)\n", f, dt)
					continue
				}
				for _, p := range probs {
					fmt.Fprintf(out, "✗ %s: %s\n", f, p)
					total++
				}
			}
			if total > 0 {
				return fmt.Errorf("%d doc spec problem(s)", total)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "lint every typed .md document in the repo")
	return cmd
}

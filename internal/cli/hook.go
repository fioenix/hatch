package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const preCommitHook = `#!/bin/sh
# Installed by ` + "`hatch hook install`" + `. Keeps SSOT, board and compiled
# files consistent before each commit.
set -e
if ! command -v hatch >/dev/null 2>&1; then
    echo "hatch not on PATH; skipping hatch pre-commit checks" >&2
    exit 0
fi
hatch validate
hatch compile --check
`

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "hook", Short: "Manage git hooks for this workspace"}
	cmd.AddCommand(newHookInstallCmd())
	return cmd
}

func newHookInstallCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a pre-commit hook (validate + compile --check)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			hooksDir := filepath.Join(ws.Layout.RepoRoot(), ".git", "hooks")
			if fi, err := os.Stat(filepath.Dir(hooksDir)); err != nil || !fi.IsDir() {
				return fmt.Errorf("not a git repository (no .git at %s)", ws.Layout.RepoRoot())
			}
			if err := os.MkdirAll(hooksDir, 0o755); err != nil {
				return err
			}
			path := filepath.Join(hooksDir, "pre-commit")
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", path)
			}
			if err := os.WriteFile(path, []byte(preCommitHook), 0o755); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "installed pre-commit hook → %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing pre-commit hook")
	return cmd
}

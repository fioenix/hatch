package orchestrator

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeDir is where per-ticket git worktrees are created (under .hatch).
const WorktreeDir = ".hatch/.worktrees"

// AddWorktree creates an isolated git worktree for a ticket on the given branch,
// creating the branch if needed. Returns the worktree path.
func AddWorktree(repoRoot, ticketID, branch string) (string, error) {
	path := filepath.Join(repoRoot, WorktreeDir, ticketID)
	if branch == "" {
		branch = "hatch/" + ticketID
	}
	// Create the branch if it does not exist, then add the worktree.
	args := []string{"worktree", "add"}
	if !branchExists(repoRoot, branch) {
		args = append(args, "-b", branch)
	}
	args = append(args, path)
	if branchExists(repoRoot, branch) {
		args = append(args, branch)
	}
	if out, err := git(repoRoot, args...); err != nil {
		return "", fmt.Errorf("git worktree add: %v: %s", err, out)
	}
	return path, nil
}

// RemoveWorktree tears down a ticket worktree.
func RemoveWorktree(repoRoot, path string) error {
	if out, err := git(repoRoot, "worktree", "remove", "--force", path); err != nil {
		return fmt.Errorf("git worktree remove: %v: %s", err, out)
	}
	return nil
}

func branchExists(repoRoot, branch string) bool {
	_, err := git(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

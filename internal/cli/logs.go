package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var follow, all bool
	cmd := &cobra.Command{
		Use:   "logs <ticket>",
		Short: "Show an agent run transcript (raw output); --follow to tail live",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			dir := ws.Layout.Runs(args[0])
			files, err := runLogs(dir)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no runs for %s\n", args[0])
				return nil
			}
			out := cmd.OutOrStdout()
			if all {
				for _, f := range files {
					fmt.Fprintf(out, "\n=== %s ===\n", filepath.Base(f))
					raw, _ := os.ReadFile(f)
					out.Write(raw)
				}
				return nil
			}
			latest := files[len(files)-1]
			if follow {
				return tail(out, latest)
			}
			raw, err := os.ReadFile(latest)
			if err != nil {
				return err
			}
			out.Write(raw)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "tail the latest run live")
	cmd.Flags().BoolVar(&all, "all", false, "print every run transcript")
	return cmd
}

func runLogs(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out) // timestamp-prefixed names sort chronologically
	return out, nil
}

// tail streams existing content then polls for appended bytes.
func tail(out io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if err == io.EOF {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if err != nil {
			return err
		}
	}
}

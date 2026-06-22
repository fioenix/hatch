package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/daemon"
	"github.com/fioenix/overclaud/hatch/internal/roster"
	"github.com/fioenix/overclaud/hatch/internal/session"
	"github.com/fioenix/overclaud/hatch/internal/wake"
)

func newDaemonCmd() *cobra.Command {
	var interval time.Duration
	var once bool
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the wake daemon: deliver chat @mentions to teammates and wake them",
		Long: "Tails the shared chat bus and wakes the teammate a message is addressed to, " +
			"resuming their CLI session so they keep their memory. This is delivery only — it " +
			"never assigns work. Wakes are paced by the wake policy (coalesce, debounce, depth, " +
			"rate, loop-break) and unresolvable cascades escalate to the boss's DM. Ctrl-C to stop.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			d := daemon.New(
				bus.New(ws.Layout),
				roster.New(ws.Layout),
				daemon.ExecRunner{
					RepoRoot: ws.Layout.RepoRoot(),
					Stdout:   out,
					Sessions: session.New(ws.Layout),
				},
				wake.Config{},
			)

			tick := func() {
				dispatched, esc, err := d.Tick(time.Now())
				if err != nil {
					fmt.Fprintln(out, "tick error:", err)
					return
				}
				for _, x := range dispatched {
					fmt.Fprintf(out, "woke %s (%s, %d msg)\n", x.Agent, x.Reason, len(x.Payload))
				}
				for _, e := range esc {
					fmt.Fprintf(out, "escalated to %s: %s (%s)\n", e.To, e.Cause, e.Episode)
				}
			}

			if once {
				tick()
				return nil
			}

			fmt.Fprintf(out, "hatch watch: delivering chat every %s (Ctrl-C to stop)\n", interval)
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				select {
				case <-sig:
					fmt.Fprintln(out, "\nhatch watch: stopped")
					return nil
				case <-t.C:
					tick()
				}
			}
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 3*time.Second, "how often to check the bus for new messages")
	cmd.Flags().BoolVar(&once, "once", false, "process one delivery cycle and exit (for scripts/CI)")
	return cmd
}

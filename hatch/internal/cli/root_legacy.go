//go:build hatch_legacy

package cli

import "github.com/spf13/cobra"

// addLegacyCommands registers the archived self-driving operator commands when
// built with `-tags hatch_legacy`. These predate the embedded-harness pivot:
// Hatch drove agents (run/plan/watch/tick), enforced the workflow engine
// (gate/escalate/ticket), and ran ceremonies/coordination as spawn loops
// (ask/convene/pair/mob), plus presence/oncall/cost/metrics tracking.
func addLegacyCommands(root *cobra.Command) {
	root.AddCommand(
		newRunCmd(),
		newPlanCmd(),
		newWatchCmd(),
		newTickCmd(),
		newTicketCmd(),
		newGateCmd(),
		newEscalateCmd(),
		newStandupCmd(),
		newCeremonyCmd(),
		newAskCmd(),
		newConveneCmd(),
		newPairCmd(),
		newMobCmd(),
		newPresenceCmd(),
		newOncallCmd(),
		newCostCmd(),
		newBudgetCmd(),
		newWorkloadCmd(),
		newPerfCmd(),
		newReportCmd(),
	)
}

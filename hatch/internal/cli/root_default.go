//go:build !hatch_legacy

package cli

import "github.com/spf13/cobra"

// addLegacyCommands is a no-op in the default build. The self-driving operator
// commands live behind the `hatch_legacy` build tag (see root_legacy.go).
func addLegacyCommands(*cobra.Command) {}

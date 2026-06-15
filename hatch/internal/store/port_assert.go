package store

import "github.com/fioenix/overclaud/hatch/internal/port"

// Compile-time guarantees that the filesystem store satisfies the use-case
// ports it is wired into.
var (
	_ port.Board  = (*Board)(nil)
	_ port.Ledger = (*Ledger)(nil)
)

package store

import "github.com/fioenix/hatch/internal/port"

// Compile-time guarantees that the filesystem store satisfies the use-case
// ports it is wired into.
var (
	_ port.Ledger = (*Ledger)(nil)
	_ port.KB     = (*KB)(nil)
)

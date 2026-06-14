// Package decide turns a meeting decision into a recorded ADR in the Knowledge
// Base, closing the loop from convene (DECISION:) to durable knowledge.
package decide

import (
	"time"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// Record writes a decision as an accepted ADR in kb/decisions/, rebuilds the
// KB index, and notes it in the ledger. `channel` (the meeting thread) is
// linked for traceability.
func Record(ws *config.Workspace, channel, title, author, body string) (model.KBEntry, error) {
	kb := store.NewKB(ws.Layout)
	entry := model.KBEntry{
		ID:      kb.NextID(model.KBDecision),
		Type:    model.KBDecision,
		Title:   title,
		Author:  author,
		Status:  "accepted",
		Created: time.Now().Format(time.RFC3339),
		Body:    body,
	}
	if channel != "" {
		entry.Related = []string{channel}
	}
	if _, err := kb.Add(entry); err != nil {
		return model.KBEntry{}, err
	}
	if err := kb.RebuildIndex(); err != nil {
		return model.KBEntry{}, err
	}
	_ = store.NewLedger(ws.Layout).Append(model.Entry{
		Agent: author, Ticket: "-", Action: model.ActNote,
		Why: "decision recorded: " + title, Note: entry.ID,
	})
	return entry, nil
}

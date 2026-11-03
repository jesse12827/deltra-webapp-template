// Package audit records audit events in the application database and the log
// (Web-App Development Stack v2.0, Section 12). Audit events are append-only:
// the application role may insert and read them, never change or delete them.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"deltra/webapp/internal/dbgen"
	"deltra/webapp/internal/requestid"
)

// Event is one auditable action.
type Event struct {
	UserID     *int64 // nil when no user is signed in (for example, a failed sign-in)
	Action     string // dotted verb, e.g. "item.create", "signin.denied"
	RecordType string // e.g. "item"; empty when not about a record
	RecordID   *int64
	Changes    map[string]Change // for updates: changed fields only
	Detail     map[string]string // other context; never secrets
}

// Change is a field's value before and after an update.
type Change struct {
	Old any `json:"old"`
	New any `json:"new"`
}

// Record writes e using q. Pass the Queries bound to the transaction that makes
// the change, so the change and its audit event commit or fail together.
func Record(ctx context.Context, q *dbgen.Queries, e Event) error {
	var payload []byte
	if len(e.Changes) > 0 || len(e.Detail) > 0 {
		b, err := json.Marshal(struct {
			Changes map[string]Change `json:"changes,omitempty"`
			Detail  map[string]string `json:"detail,omitempty"`
		}{e.Changes, e.Detail})
		if err != nil {
			return fmt.Errorf("encoding audit event: %w", err)
		}
		payload = b
	}
	reqID := requestid.From(ctx)
	err := q.InsertAuditEvent(ctx, dbgen.InsertAuditEventParams{
		UserID:     e.UserID,
		Action:     e.Action,
		RecordType: e.RecordType,
		RecordID:   e.RecordID,
		Changes:    payload,
		RequestID:  reqID,
	})
	if err != nil {
		return fmt.Errorf("recording audit event %s: %w", e.Action, err)
	}
	slog.InfoContext(ctx, "audit",
		"action", e.Action, "record_type", e.RecordType, "record_id", e.RecordID,
		"user_id", e.UserID, "request_id", reqID)
	return nil
}

// Diff returns the fields whose values differ between before and after.
func Diff(before, after map[string]any) map[string]Change {
	out := map[string]Change{}
	for k, nv := range after {
		if ov := before[k]; fmt.Sprint(ov) != fmt.Sprint(nv) {
			out[k] = Change{Old: ov, New: nv}
		}
	}
	return out
}

// Ptr returns a pointer to v, for the optional UserID and RecordID fields.
func Ptr(v int64) *int64 { return &v }

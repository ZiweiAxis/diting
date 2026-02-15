package audit

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"diting/internal/models"
)

func TestJSONLStore_AppendAndQueryByTraceID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	store, err := NewJSONLStore(path, nil)
	if err != nil {
		t.Fatalf("NewJSONLStore: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	now := time.Now()
	e1 := &models.Evidence{
		TraceID:        "trace-1",
		AgentID:        "agent-a",
		PolicyRuleID:   "r1",
		DecisionReason: "allow by rule",
		Decision:       "allow",
		Timestamp:      now,
	}
	e2 := &models.Evidence{
		TraceID:        "trace-1",
		SpanID:         "span-2",
		Decision:       "review",
		Timestamp:      now.Add(time.Second),
	}
	e3 := &models.Evidence{
		TraceID:   "trace-2",
		Decision:  "deny",
		Timestamp: now,
	}

	if err := store.Append(ctx, e1); err != nil {
		t.Fatalf("Append e1: %v", err)
	}
	if err := store.Append(ctx, e2); err != nil {
		t.Fatalf("Append e2: %v", err)
	}
	if err := store.Append(ctx, e3); err != nil {
		t.Fatalf("Append e3: %v", err)
	}

	list1, err := store.QueryByTraceID(ctx, "trace-1")
	if err != nil {
		t.Fatalf("QueryByTraceID trace-1: %v", err)
	}
	if len(list1) != 2 {
		t.Errorf("trace-1: expected 2, got %d", len(list1))
	}
	list2, err := store.QueryByTraceID(ctx, "trace-2")
	if err != nil {
		t.Fatalf("QueryByTraceID trace-2: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("trace-2: expected 1, got %d", len(list2))
	}
	list0, err := store.QueryByTraceID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("QueryByTraceID nonexistent: %v", err)
	}
	if len(list0) != 0 {
		t.Errorf("nonexistent: expected 0, got %d", len(list0))
	}
}

func TestJSONLStore_AppendNil(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(filepath.Join(dir, "a.jsonl"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Append(context.Background(), nil); err != nil {
		t.Errorf("Append(nil) should not error: %v", err)
	}
}

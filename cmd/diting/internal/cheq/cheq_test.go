package cheq

import (
	"context"
	"testing"
	"time"

	"diting/internal/models"
)

func TestEngineImpl_CreateGetByIDSubmit(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	eng := NewEngineImpl(store, 300, nil, nil, "any") // no delivery in test
	ctx := context.Background()

	in := &CreateInput{
		TraceID:      "trace-1",
		Resource:     "/api/data",
		Action:       "write",
		Summary:      "test summary",
		ConfirmerIDs: []string{"user-1"},
		Type:         "operation_approval",
	}
	obj, err := eng.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj.ID == "" {
		t.Error("Create: expected non-empty ID")
	}
	if obj.Status != models.ConfirmationStatusPending {
		t.Errorf("Create: status = %v", obj.Status)
	}

	got, err := eng.GetByID(ctx, obj.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != obj.ID || got.TraceID != "trace-1" {
		t.Errorf("GetByID: got %+v", got)
	}

	if err := eng.Submit(ctx, obj.ID, true, ""); err != nil {
		t.Fatalf("Submit approved: %v", err)
	}
	got2, _ := eng.GetByID(ctx, obj.ID)
	if got2.Status != models.ConfirmationStatusApproved {
		t.Errorf("after Submit(approved): status = %v", got2.Status)
	}

	if err := eng.Submit(ctx, obj.ID, false, ""); err != ErrAlreadyProcessed {
		t.Errorf("Submit again: want ErrAlreadyProcessed, got %v", err)
	}
}

func TestEngineImpl_SubmitRejected(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewJSONStore(dir)
	eng := NewEngineImpl(store, 300, nil, nil, "any")
	ctx := context.Background()

	obj, _ := eng.Create(ctx, &CreateInput{
		TraceID: "t2", Resource: "/r", Action: "a", Summary: "s",
		ConfirmerIDs: []string{"u1"}, Type: "op",
	})
	if err := eng.Submit(ctx, obj.ID, false, ""); err != nil {
		t.Fatalf("Submit rejected: %v", err)
	}
	got, _ := eng.GetByID(ctx, obj.ID)
	if got.Status != models.ConfirmationStatusRejected {
		t.Errorf("status = %v", got.Status)
	}
}

func TestEngineImpl_GetByID_NotFound(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewJSONStore(dir)
	eng := NewEngineImpl(store, 300, nil, nil, "any")
	ctx := context.Background()

	got, err := eng.GetByID(ctx, "nonexistent-id")
	if err != nil {
		t.Fatalf("GetByID not found: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestEngineImpl_Submit_NotFound(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewJSONStore(dir)
	eng := NewEngineImpl(store, 300, nil, nil, "any")
	ctx := context.Background()

	err := eng.Submit(ctx, "nonexistent-id", true, "")
	if err != ErrNotFound {
		t.Errorf("Submit not found: want ErrNotFound, got %v", err)
	}
}

func TestEngineImpl_Create_NilInput(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewJSONStore(dir)
	eng := NewEngineImpl(store, 300, nil, nil, "any")
	ctx := context.Background()

	_, err := eng.Create(ctx, nil)
	if err == nil {
		t.Error("Create(nil): expected error")
	}
}

func TestEngineImpl_Submit_AllPolicy(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewJSONStore(dir)
	eng := NewEngineImpl(store, 300, nil, nil, "all")
	ctx := context.Background()

	obj, _ := eng.Create(ctx, &CreateInput{
		TraceID: "t-all", Resource: "/r", Action: "a", Summary: "s",
		ConfirmerIDs: []string{"u1", "u2"}, Type: "op",
	})
	// 仅 u1 批准，未达全部
	if err := eng.Submit(ctx, obj.ID, true, "u1"); err != nil {
		t.Fatalf("Submit u1: %v", err)
	}
	got1, _ := eng.GetByID(ctx, obj.ID)
	if got1.Status == models.ConfirmationStatusApproved {
		t.Error("expected still pending after u1 only")
	}
	// u2 批准，达全部
	if err := eng.Submit(ctx, obj.ID, true, "u2"); err != nil {
		t.Fatalf("Submit u2: %v", err)
	}
	got2, _ := eng.GetByID(ctx, obj.ID)
	if got2.Status != models.ConfirmationStatusApproved {
		t.Errorf("expected Approved after all, got %v", got2.Status)
	}
}

func TestEngineImpl_Submit_Expired(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewJSONStore(dir)
	eng := NewEngineImpl(store, 1, nil, nil, "any") // 1 second timeout
	ctx := context.Background()

	obj, _ := eng.Create(ctx, &CreateInput{
		TraceID: "t-exp", Resource: "/r", Action: "a", Summary: "s",
		ConfirmerIDs: []string{"u1"}, Type: "op",
	})
	time.Sleep(1100 * time.Millisecond)
	err := eng.Submit(ctx, obj.ID, true, "")
	if err != ErrExpired {
		t.Errorf("Submit expired: want ErrExpired, got %v", err)
	}
}

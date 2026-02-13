package chain

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalStore_AppendBatch_GetMerkleProof(t *testing.T) {
	ctx := context.Background()
	s := NewLocalStore()
	defer s.Close()

	leaves := []TraceLeaf{
		{TraceID: "trace-1", Hash: "aa"},
		{TraceID: "trace-2", Hash: "bb"},
	}
	batch := &BatchRecord{BatchID: "batch-1"}
	root, err := s.AppendBatch(ctx, batch, leaves)
	if err != nil {
		t.Fatal(err)
	}
	if root == "" {
		t.Fatal("empty merkle root")
	}

	proof, err := s.GetMerkleProof(ctx, "trace-1")
	if err != nil {
		t.Fatal(err)
	}
	if proof.BatchID != "batch-1" || proof.MerkleRoot != root || proof.LeafHash != "aa" {
		t.Errorf("proof: batch=%s root=%s leaf=%s", proof.BatchID, proof.MerkleRoot, proof.LeafHash)
	}
}

func TestLocalStore_PutDID_GetDID_Persist(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "chain")
	s := NewLocalStoreWithPath(dir)
	defer s.Close()

	doc := &DIDDocument{
		ID:                     "did:ziwei:local:abc123",
		PublicKey:              "pk",
		EnvironmentFingerprint: "fp",
		Status:                 DIDStatusActive,
	}
	if err := s.PutDID(ctx, doc); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetDID(ctx, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != doc.ID || got.PublicKey != doc.PublicKey {
		t.Errorf("got %+v", got)
	}

	// 新实例从目录读
	s2 := NewLocalStoreWithPath(dir)
	got2, err := s2.GetDID(ctx, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.ID != doc.ID {
		t.Errorf("persisted read: got %+v", got2)
	}
	s2.Close()
}

func TestLocalStore_AppendBatch_Persist(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "chain")
	s := NewLocalStoreWithPath(dir)
	defer s.Close()

	leaves := []TraceLeaf{
		{TraceID: "tr1", Hash: "h1"},
		{TraceID: "tr2", Hash: "h2"},
	}
	batch := &BatchRecord{BatchID: "b1", Timestamp: time.Now().UTC()}
	root, err := s.AppendBatch(ctx, batch, leaves)
	if err != nil {
		t.Fatal(err)
	}
	if root == "" {
		t.Fatal("empty root")
	}

	// 新实例从目录读 proof
	s2 := NewLocalStoreWithPath(dir)
	proof, err := s2.GetMerkleProof(ctx, "tr1")
	if err != nil {
		t.Fatal(err)
	}
	if proof.MerkleRoot != root || proof.BatchID != "b1" {
		t.Errorf("got %+v", proof)
	}
	s2.Close()
}

func TestLocalStore_GetDID_NotFound(t *testing.T) {
	s := NewLocalStore()
	defer s.Close()
	_, err := s.GetDID(context.Background(), "did:not:exist")
	if err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestLocalStore_GetMerkleProof_NotFound(t *testing.T) {
	s := NewLocalStore()
	defer s.Close()
	_, err := s.GetMerkleProof(context.Background(), "no-such-trace")
	if err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestSanitize(t *testing.T) {
	if sanitize("did:ziwei:x:y") == "" {
		t.Error("sanitize should replace colons")
	}
}

// 确保目录创建
func TestNewLocalStoreWithPath_createsDirs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chain")
	s := NewLocalStoreWithPath(dir)
	defer s.Close()
	for _, sub := range []string{"dids", "batches", "proofs"} {
		p := filepath.Join(dir, sub)
		if fi, err := os.Stat(p); err != nil || !fi.IsDir() {
			t.Errorf("expected dir %s: %v", p, err)
		}
	}
}

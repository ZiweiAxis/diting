package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestBuildMerkleTree(t *testing.T) {
	leaves := []TraceLeaf{
		{TraceID: "t1", Hash: hash("a")},
		{TraceID: "t2", Hash: hash("b")},
		{TraceID: "t3", Hash: hash("c")},
	}
	root, proofs := BuildMerkleTree(leaves)
	if root == "" {
		t.Fatal("empty root")
	}
	if len(proofs) != 3 {
		t.Fatalf("want 3 proofs, got %d", len(proofs))
	}
	for i, p := range proofs {
		if p.LeafHash != leaves[i].Hash {
			t.Errorf("proof[%d].LeafHash = %s, want %s", i, p.LeafHash, leaves[i].Hash)
		}
	}
}

func hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

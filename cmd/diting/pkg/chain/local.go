package chain

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

// LocalStore 为可插拔存储后端：内存 + 可选目录持久化（Story 10.2）。
// basePath 为空时仅内存；非空时 DID 与存证批次写入目录（dids/、batches/、proofs/）。
type LocalStore struct {
	mu       sync.RWMutex
	didDocs  map[string]*DIDDocument
	proofs   map[string]*MerkleProof // traceID -> proof，basePath 为空时使用
	basePath string
}

// NewLocalStore 创建仅内存的 LocalStore（兼容 10.1）。
func NewLocalStore() *LocalStore {
	return NewLocalStoreWithPath("")
}

// PutDID 实现 Backend.PutDID。持久化到 dids/<did>.json（当 basePath 非空）并写入内存。
func (s *LocalStore) PutDID(ctx context.Context, doc *DIDDocument) error {
	_ = ctx
	if doc == nil {
		return errors.New("chain: nil DIDDocument")
	}
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	s.didDocs[doc.ID] = doc
	if s.basePath != "" {
		p := s.didPath(doc.ID)
		b, _ := json.MarshalIndent(doc, "", "  ")
		if err := os.WriteFile(p, b, 0644); err != nil {
			return err
		}
	}
	return nil
}

// GetDID 实现 Backend.GetDID。先查内存；若 basePath 非空且未命中则尝试从文件读。
func (s *LocalStore) GetDID(ctx context.Context, did string) (*DIDDocument, error) {
	_ = ctx
	s.mu.RLock()
	if doc, ok := s.didDocs[did]; ok {
		s.mu.RUnlock()
		return doc, nil
	}
	s.mu.RUnlock()
	if s.basePath != "" {
		p := s.didPath(did)
		b, err := os.ReadFile(p)
		if err == nil {
			var doc DIDDocument
			if json.Unmarshal(b, &doc) == nil {
				s.mu.Lock()
				s.didDocs[did] = &doc
				s.mu.Unlock()
				return &doc, nil
			}
		}
	}
	return nil, ErrNotFound
}

// AppendBatch 实现 Backend.AppendBatch：构建 Merkle 树，持久化批次与每条 trace 的验真数据。
func (s *LocalStore) AppendBatch(ctx context.Context, batch *BatchRecord, traceLeaves []TraceLeaf) (string, error) {
	_ = ctx
	if batch == nil {
		return "", errors.New("chain: nil BatchRecord")
	}
	if len(traceLeaves) == 0 {
		return "", errors.New("chain: empty trace leaves")
	}
	rootHash, proofs := BuildMerkleTree(traceLeaves)
	batch.MerkleRoot = rootHash
	if batch.Timestamp.IsZero() {
		batch.Timestamp = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.basePath != "" {
		batchPath := s.batchPath(batch.BatchID)
		b, _ := json.MarshalIndent(batch, "", "  ")
		if err := os.WriteFile(batchPath, b, 0644); err != nil {
			return "", err
		}
		for i := range traceLeaves {
			traceID := traceLeaves[i].TraceID
			proof := &MerkleProof{
				TraceID:    traceID,
				BatchID:    batch.BatchID,
				MerkleRoot: rootHash,
				LeafHash:   proofs[i].LeafHash,
				Siblings:   proofs[i].Siblings,
			}
			p := s.proofPath(traceID)
			b, _ := json.MarshalIndent(proof, "", "  ")
			if err := os.WriteFile(p, b, 0644); err != nil {
				return "", err
			}
		}
	} else {
		for i := range traceLeaves {
			traceID := traceLeaves[i].TraceID
			s.proofs[traceID] = &MerkleProof{
				TraceID:    traceID,
				BatchID:    batch.BatchID,
				MerkleRoot: rootHash,
				LeafHash:   proofs[i].LeafHash,
				Siblings:   proofs[i].Siblings,
			}
		}
	}
	return rootHash, nil
}

// GetMerkleProof 实现 Backend.GetMerkleProof。basePath 非空时从文件读，否则从内存 proof 读。
func (s *LocalStore) GetMerkleProof(ctx context.Context, traceID string) (*MerkleProof, error) {
	_ = ctx
	if s.basePath != "" {
		p := s.proofPath(traceID)
		b, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		var proof MerkleProof
		if json.Unmarshal(b, &proof) != nil {
			return nil, errors.New("chain: invalid proof file")
		}
		return &proof, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if proof, ok := s.proofs[traceID]; ok {
		return proof, nil
	}
	return nil, ErrNotFound
}

// Close 实现 Backend.Close。
func (s *LocalStore) Close() error {
	return nil
}

// NewLedger 基于给定 Backend 构造 Ledger。一期使用 LocalStore；后续可换为其他 Backend。
func NewLedger(be Backend) Ledger {
	return &ledgerImpl{backend: be}
}

type ledgerImpl struct {
	backend Backend
}

func (l *ledgerImpl) PutDID(ctx context.Context, doc *DIDDocument) (string, error) {
	if err := l.backend.PutDID(ctx, doc); err != nil {
		return "", err
	}
	return doc.ID + "@" + doc.UpdatedAt.Format("20060102150405"), nil
}

func (l *ledgerImpl) GetDID(ctx context.Context, did string) (*DIDDocument, error) {
	return l.backend.GetDID(ctx, did)
}

func (l *ledgerImpl) AppendBatch(ctx context.Context, batchID string, traceIDToHash map[string]string) (string, error) {
	traceLeaves := make([]TraceLeaf, 0, len(traceIDToHash))
	for tid, h := range traceIDToHash {
		traceLeaves = append(traceLeaves, TraceLeaf{TraceID: tid, Hash: h})
	}
	batch := &BatchRecord{BatchID: batchID}
	merkleRoot, err := l.backend.AppendBatch(ctx, batch, traceLeaves)
	if err != nil {
		return "", err
	}
	return merkleRoot, nil
}

func (l *ledgerImpl) GetMerkleProof(ctx context.Context, traceID string) (*MerkleProof, error) {
	return l.backend.GetMerkleProof(ctx, traceID)
}

func (l *ledgerImpl) Healthy(ctx context.Context) error {
	// 简单检查：可扩展为 Ping 存储。
	return nil
}

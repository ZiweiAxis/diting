package chain

import (
	"os"
	"path/filepath"
	"strings"
)

// NewLocalStoreWithPath 创建支持目录持久化的 LocalStore。basePath 为空时仅内存存储。
func NewLocalStoreWithPath(basePath string) *LocalStore {
	s := &LocalStore{
		didDocs:  make(map[string]*DIDDocument),
		proofs:   make(map[string]*MerkleProof),
		basePath: strings.TrimSuffix(basePath, string(os.PathSeparator)),
	}
	if s.basePath != "" {
		_ = os.MkdirAll(filepath.Join(s.basePath, "dids"), 0755)
		_ = os.MkdirAll(filepath.Join(s.basePath, "batches"), 0755)
		_ = os.MkdirAll(filepath.Join(s.basePath, "proofs"), 0755)
	}
	return s
}

// sanitize 将 DID 或 trace_id 转为安全文件名（替换 : / \ 为 _）。
func sanitize(id string) string {
	return strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(id)
}

func (s *LocalStore) didPath(did string) string {
	return filepath.Join(s.basePath, "dids", sanitize(did)+".json")
}

func (s *LocalStore) batchPath(batchID string) string {
	return filepath.Join(s.basePath, "batches", sanitize(batchID)+".json")
}

func (s *LocalStore) proofPath(traceID string) string {
	return filepath.Join(s.basePath, "proofs", sanitize(traceID)+".json")
}


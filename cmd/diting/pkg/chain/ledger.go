package chain

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("chain: not found")
	ErrStorageClosed = errors.New("chain: storage closed")
)

// Ledger 定义链上 DID 与存证批次的读写与验真查询，与 I-016 设计一致。
// 具体存储由 Backend 实现（如最小链、LevelDB、或后续 Fabric 适配）。
type Ledger interface {
	// DID
	PutDID(ctx context.Context, doc *DIDDocument) (txIDOrVersion string, err error)
	GetDID(ctx context.Context, did string) (*DIDDocument, error)

	// 存证批次：追加一批 trace 哈希，构建 Merkle 树并持久化；traceIDToHash 为 trace_id -> 叶节点哈希。
	AppendBatch(ctx context.Context, batchID string, traceIDToHash map[string]string) (merkleRoot string, err error)

	// 验真：根据 trace_id 返回 Merkle 路径与链上根，供客户端 3 秒内验真。
	GetMerkleProof(ctx context.Context, traceID string) (*MerkleProof, error)

	// 健康：存储是否可写、最近一批是否成功（可选）。
	Healthy(ctx context.Context) error
}

// Backend 为可插拔存储后端接口，Ledger 实现可依赖此接口持久化。
// 一期可用 LocalStore（如目录+文件或 LevelDB）；二期可换为 Fabric/长安链适配器。
type Backend interface {
	// DID
	PutDID(ctx context.Context, doc *DIDDocument) error
	GetDID(ctx context.Context, did string) (*DIDDocument, error)

	// 存证：写入批次元数据与 trace_id -> (batch_id, leaf_index, siblings) 映射；返回本批次的 Merkle 根。
	AppendBatch(ctx context.Context, batch *BatchRecord, traceLeaves []TraceLeaf) (merkleRoot string, err error)
	GetMerkleProof(ctx context.Context, traceID string) (*MerkleProof, error)

	Close() error
}

// TraceLeaf 表示存证批次中的一条叶节点：trace_id 与其哈希。
type TraceLeaf struct {
	TraceID string
	Hash    string
}

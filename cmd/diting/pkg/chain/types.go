// Package chain 提供谛听私有链子模块的 Ledger 抽象与数据类型，与紫微技术方案 §3.5、§3.6 及 I-016 设计一致。
package chain

import "time"

// DIDStatus 表示 DID 文档状态。
type DIDStatus string

const (
	DIDStatusActive   DIDStatus = "active"
	DIDStatusRevoked  DIDStatus = "revoked"
	DIDStatusPending  DIDStatus = "pending"
)

// DIDDocument 表示链上 DID 文档（与 I-016 §4 一致，仅指纹与公钥，无敏感数据）。
type DIDDocument struct {
	ID                     string    `json:"id"`                       // DID，如 did:ziwei:<chain_id>:<hash>
	PublicKey              string    `json:"publicKey"`                 // 公钥材料（PEM 或 JWK 等）
	EnvironmentFingerprint string    `json:"environmentFingerprint"`   // 部署环境指纹哈希
	Owner                  string    `json:"owner,omitempty"`           // 所有者 DID 或标识
	Status                 DIDStatus `json:"status"`                   // active / revoked
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

// BatchRecord 表示链上存证批次元数据（与 I-016 §4 一致）。
type BatchRecord struct {
	BatchID    string    `json:"batch_id"`
	MerkleRoot string    `json:"merkle_root"`   // 十六进制或 base64
	Timestamp  time.Time `json:"timestamp"`
	SignerDID  string    `json:"signer_did,omitempty"`
}

// MerkleProof 供验真使用：给定 trace_id 返回所在批次的根与路径，便于客户端重算比对。
type MerkleProof struct {
	TraceID    string   `json:"trace_id"`
	BatchID    string   `json:"batch_id"`
	MerkleRoot string   `json:"merkle_root"`
	LeafHash   string   `json:"leaf_hash"`   // 叶节点哈希
	Siblings   []string `json:"siblings"`   // Merkle 路径上的兄弟节点（由叶到根顺序）
}

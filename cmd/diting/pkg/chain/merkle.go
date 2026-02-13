package chain

import (
	"crypto/sha256"
	"encoding/hex"
)

// BuildMerkleTree 根据叶节点哈希列表构建 Merkle 树，返回根哈希与每个叶节点的验真路径（兄弟节点哈希，由叶到根）。
// 与紫微技术方案 §3.5 一致：SHA3-256 哈希；一期用 sha256 简化，可后续改为 sha3。
func BuildMerkleTree(leaves []TraceLeaf) (rootHash string, proofs []MerkleProofPath) {
	if len(leaves) == 0 {
		return "", nil
	}
	// 叶层：每个叶节点哈希
	hashes := make([]string, len(leaves))
	for i := range leaves {
		hashes[i] = leaves[i].Hash
	}
	// 构建树并记录每一层的节点，以便为每个叶节点生成 path
	type node struct {
		hash   string
		left   *node
		right  *node
		leafIdx int // 仅叶节点有效，-1 表示非叶
	}
	nodes := make([]*node, len(hashes))
	for i, h := range hashes {
		nodes[i] = &node{hash: h, leafIdx: i}
	}
	layer := nodes
	var allLayers [][]*node
	allLayers = append(allLayers, layer)
	for len(layer) > 1 {
		next := make([]*node, 0, (len(layer)+1)/2)
		for i := 0; i < len(layer); i += 2 {
			left := layer[i]
			right := left
			if i+1 < len(layer) {
				right = layer[i+1]
			}
			parentHash := hashPair(left.hash, right.hash)
			next = append(next, &node{hash: parentHash, left: left, right: right, leafIdx: -1})
		}
		layer = next
		allLayers = append(allLayers, layer)
	}
	rootHash = layer[0].hash

	// 为每个叶节点生成 sibling path（从叶到根的兄弟哈希序列）
	proofs = make([]MerkleProofPath, len(leaves))
	for leafIdx := 0; leafIdx < len(leaves); leafIdx++ {
		var path []string
		idx := leafIdx
		for L := 0; L < len(allLayers)-1; L++ {
			row := allLayers[L]
			siblingIdx := idx ^ 1
			if siblingIdx < len(row) {
				path = append(path, row[siblingIdx].hash)
			}
			idx = idx / 2
		}
		proofs[leafIdx] = MerkleProofPath{LeafHash: leaves[leafIdx].Hash, Siblings: path}
	}
	return rootHash, proofs
}

func hashPair(a, b string) string {
	h := sha256.New()
	h.Write([]byte(a))
	h.Write([]byte(b))
	return hex.EncodeToString(h.Sum(nil))
}

// MerkleProofPath 表示单个叶节点的验真路径（叶哈希 + 兄弟序列）。
type MerkleProofPath struct {
	LeafHash string
	Siblings []string
}

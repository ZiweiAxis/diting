// Package chain 提供 /chain/* 的 HTTP Handler，暴露 DID 与存证 API（I-016 §3）。
package chain

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"diting/pkg/chain"
)

// Server 暴露链上 DID 与存证 HTTP API。
type Server struct {
	Ledger chain.Ledger
}

// NewServer 构造 chain HTTP Server。
func NewServer(ledger chain.Ledger) *Server {
	return &Server{Ledger: ledger}
}

// Handler 返回挂载在 /chain 下的 Handler（调用方需 StripPrefix("/chain", s.Handler())）。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/did/register", s.handleDIDRegister)
	mux.HandleFunc("/did/", s.handleDIDGet)
	mux.HandleFunc("/audit/batch", s.handleAuditBatch)
	mux.HandleFunc("/audit/verify", s.handleAuditVerify)
	mux.HandleFunc("/health", s.handleHealth)
	return mux
}

// DIDRegisterRequest POST /chain/did/register 请求体。
type DIDRegisterRequest struct {
	ID                     string         `json:"id"`
	PublicKey              string         `json:"publicKey"`
	EnvironmentFingerprint string         `json:"environmentFingerprint"`
	Owner                  string         `json:"owner,omitempty"`
	Status                 chain.DIDStatus `json:"status,omitempty"`
}

// DIDRegisterResponse 返回交易 ID 或版本。
type DIDRegisterResponse struct {
	TxID string `json:"tx_id"`
}

func (s *Server) handleDIDRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req DIDRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.ID == "" || req.PublicKey == "" {
		http.Error(w, "id and publicKey required", http.StatusBadRequest)
		return
	}
	status := req.Status
	if status == "" {
		status = chain.DIDStatusActive
	}
	doc := &chain.DIDDocument{
		ID:                     req.ID,
		PublicKey:              req.PublicKey,
		EnvironmentFingerprint: req.EnvironmentFingerprint,
		Owner:                  req.Owner,
		Status:                 status,
	}
	txID, err := s.Ledger.PutDID(r.Context(), doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(DIDRegisterResponse{TxID: txID})
}

func (s *Server) handleDIDGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	did := strings.TrimPrefix(r.URL.Path, "/did/")
	if did == "" {
		http.Error(w, "did path required", http.StatusBadRequest)
		return
	}
	doc, err := s.Ledger.GetDID(r.Context(), did)
	if err != nil {
		if err == chain.ErrNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}

// AuditBatchRequest POST /chain/audit/batch 请求体。
type AuditBatchRequest struct {
	BatchID     string            `json:"batch_id"`
	TraceIDHash map[string]string `json:"trace_id_hash"` // trace_id -> 叶节点哈希
}

// AuditBatchResponse 返回链上交易 ID（此处为 merkle_root 或 batch_id）。
type AuditBatchResponse struct {
	TxID       string `json:"tx_id"`
	MerkleRoot string `json:"merkle_root"`
}

func (s *Server) handleAuditBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AuditBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.BatchID == "" || len(req.TraceIDHash) == 0 {
		http.Error(w, "batch_id and trace_id_hash required", http.StatusBadRequest)
		return
	}
	merkleRoot, err := s.Ledger.AppendBatch(r.Context(), req.BatchID, req.TraceIDHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(AuditBatchResponse{TxID: merkleRoot, MerkleRoot: merkleRoot})
}

func (s *Server) handleAuditVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		http.Error(w, "trace_id query required", http.StatusBadRequest)
		return
	}
	proof, err := s.Ledger.GetMerkleProof(r.Context(), traceID)
	if err != nil {
		if err == chain.ErrNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(proof)
}

// HealthResponse GET /chain/health。
type HealthResponse struct {
	OK    bool   `json:"ok"`
	Since string `json:"since,omitempty"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := s.Ledger.Healthy(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(HealthResponse{OK: false})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{OK: true})
}

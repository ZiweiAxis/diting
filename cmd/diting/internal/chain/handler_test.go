package chain

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"diting/pkg/chain"
)

func TestHandler_DIDRegister_Get(t *testing.T) {
	ledger := chain.NewLedger(chain.NewLocalStore())
	srv := NewServer(ledger)
	h := srv.Handler()

	// POST /did/register
	body := []byte(`{"id":"did:ziwei:local:abc","publicKey":"pk1","environmentFingerprint":"fp1"}`)
	req := httptest.NewRequest(http.MethodPost, "/did/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /did/register: code=%d body=%s", w.Code, w.Body.String())
	}
	var res DIDRegisterResponse
	if json.NewDecoder(w.Body).Decode(&res) != nil || res.TxID == "" {
		t.Fatalf("POST /did/register: invalid response %s", w.Body.String())
	}

	// GET /did/did:ziwei:local:abc
	req2 := httptest.NewRequest(http.MethodGet, "/did/did:ziwei:local:abc", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET /did/...: code=%d", w2.Code)
	}
	var doc chain.DIDDocument
	if json.NewDecoder(w2.Body).Decode(&doc) != nil || doc.ID != "did:ziwei:local:abc" {
		t.Fatalf("GET /did/...: invalid doc %s", w2.Body.String())
	}
}

func TestHandler_AuditBatch_Verify(t *testing.T) {
	ledger := chain.NewLedger(chain.NewLocalStore())
	srv := NewServer(ledger)
	h := srv.Handler()

	body := []byte(`{"batch_id":"b1","trace_id_hash":{"t1":"h1","t2":"h2"}}`)
	req := httptest.NewRequest(http.MethodPost, "/audit/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /audit/batch: code=%d body=%s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/audit/verify?trace_id=t1", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET /audit/verify: code=%d", w2.Code)
	}
}

func TestHandler_Health(t *testing.T) {
	ledger := chain.NewLedger(chain.NewLocalStore())
	srv := NewServer(ledger)
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health: code=%d", w.Code)
	}
	var res HealthResponse
	if json.NewDecoder(w.Body).Decode(&res) != nil || !res.OK {
		t.Fatalf("GET /health: %s", w.Body.String())
	}
}

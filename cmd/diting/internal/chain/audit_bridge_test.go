package chain

import (
	"context"
	"testing"
	"time"

	"diting/internal/audit"
	"diting/internal/models"
	chainpkg "diting/pkg/chain"
)

func TestAuditChainBridge_Append_QueryByTraceID(t *testing.T) {
	inner := audit.NewStubStore()
	ledger := chainpkg.NewLedger(chainpkg.NewLocalStore())
	bridge := NewAuditChainBridge(inner, ledger, 2, 10*time.Millisecond)
	bridge.Start()
	defer bridge.Stop()

	ctx := context.Background()
	e := &models.Evidence{TraceID: "trace-1", Decision: "allow", Timestamp: time.Now()}
	if err := bridge.Append(ctx, e); err != nil {
		t.Fatal(err)
	}
	// 委托查询
	list, err := bridge.QueryByTraceID(ctx, "trace-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].TraceID != "trace-1" {
		t.Fatalf("QueryByTraceID: got %v", list)
	}
}

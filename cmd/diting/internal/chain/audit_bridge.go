// 审计→链存证桥接：Append 后异步将 Trace 哈希提交链子模块（Story 10.6）。

package chain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"
	"time"

	"diting/internal/audit"
	"diting/internal/models"
	chainpkg "diting/pkg/chain"
)

// AuditChainBridge 包装 audit.Store，在 Append 成功后异步将 (trace_id, hash) 提交链子模块批次存证。
// 不阻塞主审计写入；失败仅打日志，可并入下一批。
type AuditChainBridge struct {
	inner     audit.Store
	ledger    chainpkg.Ledger
	batchSize int
	interval  time.Duration
	ch        chan traceHash
	done      chan struct{}
	wg        sync.WaitGroup
}

type traceHash struct {
	TraceID string
	Hash    string
}

// NewAuditChainBridge 创建桥接；调用 Start() 启动后台刷盘，关闭时调用 Stop()。
func NewAuditChainBridge(inner audit.Store, ledger chainpkg.Ledger, batchSize int, interval time.Duration) *AuditChainBridge {
	if batchSize <= 0 {
		batchSize = 50
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &AuditChainBridge{
		inner:     inner,
		ledger:    ledger,
		batchSize: batchSize,
		interval:  interval,
		ch:        make(chan traceHash, 500),
		done:      make(chan struct{}),
	}
}

// Start 启动后台 goroutine：按批次或定时调用 Ledger.AppendBatch。
func (b *AuditChainBridge) Start() {
	b.wg.Add(1)
	go b.flushLoop()
}

// Stop 停止后台并等待当前批提交完成。
func (b *AuditChainBridge) Stop() {
	close(b.done)
	b.wg.Wait()
}

func (b *AuditChainBridge) flushLoop() {
	defer b.wg.Done()
	buf := make(map[string]string) // trace_id -> hash
	tick := time.NewTicker(b.interval)
	defer tick.Stop()
	flush := func() {
		if len(buf) == 0 {
			return
		}
		batchID := "audit-" + time.Now().UTC().Format("20060102150405")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := b.ledger.AppendBatch(ctx, batchID, buf)
		cancel()
		if err != nil {
			log.Printf("[diting] chain audit batch failed (batch_id=%s): %v", batchID, err)
			return
		}
		buf = make(map[string]string)
	}
	for {
		select {
		case <-b.done:
			flush()
			return
		case th := <-b.ch:
			buf[th.TraceID] = th.Hash
			if len(buf) >= b.batchSize {
				flush()
			}
		case <-tick.C:
			flush()
		}
	}
}

// Append 先写内层 Store，再异步投递 (trace_id, hash) 到链批次（不阻塞）。
func (b *AuditChainBridge) Append(ctx context.Context, e *models.Evidence) error {
	if err := b.inner.Append(ctx, e); err != nil {
		return err
	}
	if e == nil || e.TraceID == "" {
		return nil
	}
	data, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	h := sha256.Sum256(data)
	hashStr := hex.EncodeToString(h[:])
	select {
	case b.ch <- traceHash{TraceID: e.TraceID, Hash: hashStr}:
	default:
		// 通道满则丢弃，不阻塞审计
	}
	return nil
}

// QueryByTraceID 委托内层。
func (b *AuditChainBridge) QueryByTraceID(ctx context.Context, traceID string) ([]*models.Evidence, error) {
	return b.inner.QueryByTraceID(ctx, traceID)
}

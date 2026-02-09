package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"diting/internal/models"
)

// JSONLStore 追加写 JSONL 文件，并按 trace_id 查询（线性扫描）。
type JSONLStore struct {
	path string
	mu   sync.Mutex
	f    *os.File
}

// NewJSONLStore 创建或打开 path 对应的 JSONL 文件；目录不存在会创建。
func NewJSONLStore(path string) (*JSONLStore, error) {
	if path == "" {
		return nil, os.ErrInvalid
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &JSONLStore{path: path, f: f}, nil
}

// Append 追加一行 JSON（Evidence）。
func (s *JSONLStore) Append(ctx context.Context, e *models.Evidence) error {
	if e == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = s.f.Write(data)
	return err
}

// QueryByTraceID 读取整个文件并过滤 trace_id（线性扫描，适合 MVP）。
func (s *JSONLStore) QueryByTraceID(ctx context.Context, traceID string) ([]*models.Evidence, error) {
	s.mu.Lock()
	_ = s.f.Sync()
	s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*models.Evidence
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var e models.Evidence
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		if e.TraceID == traceID {
			out = append(out, &e)
		}
	}
	return out, nil
}

// Close 关闭底层文件。
func (s *JSONLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}

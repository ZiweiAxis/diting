package cheq

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"diting/internal/models"
)

// JSONStore 将每个 ConfirmationObject 存为单独 JSON 文件：<dir>/<id>.json。
type JSONStore struct {
	dir string
	mu  sync.Mutex
}

// NewJSONStore 使用 dir 作为存储目录；不存在则创建。
func NewJSONStore(dir string) (*JSONStore, error) {
	if dir == "" {
		return nil, os.ErrInvalid
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &JSONStore{dir: dir}, nil
}

func (s *JSONStore) path(id string) string { return filepath.Join(s.dir, id+".json") }

// Put 写入对象；同 id 覆盖。
func (s *JSONStore) Put(ctx context.Context, obj *models.ConfirmationObject) error {
	if obj == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(obj.ID), data, 0644)
}

// Get 读取对象；不存在返回 nil, nil。
func (s *JSONStore) Get(ctx context.Context, id string) (*models.ConfirmationObject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var obj models.ConfirmationObject
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

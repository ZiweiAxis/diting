// Package proxy 提供 GetSandboxProfile 接口（Story 8.1）；按 resource 返回边界与 Hot Cache。
package proxy

import (
	"encoding/json"
	"net/http"
)

// SandboxProfile 与 proto SandboxProfile 对齐的 JSON 结构（MVP 最小实现）。
type SandboxProfile struct {
	ProfileID         string              `json:"profile_id,omitempty"`
	Version           string              `json:"version,omitempty"`
	Boundary          *SandboxBoundary     `json:"boundary,omitempty"`
	HotCacheActions   []HotCacheAction     `json:"hot_cache_actions,omitempty"`
	SudoHotCache      []SudoHotCacheEntry  `json:"sudo_hot_cache,omitempty"`
	DegradationPolicy string               `json:"degradation_policy,omitempty"` // FAIL_OPEN | FAIL_CLOSE
}

// SandboxBoundary 沙箱边界（网络、文件系统等）。
type SandboxBoundary struct {
	NetworkEnabled   bool     `json:"network_enabled"`
	FsWritablePaths  []string `json:"fs_writable_paths,omitempty"`
	SyscallPreset    string   `json:"syscall_preset,omitempty"`
	MaxMemoryMB      int64    `json:"max_memory_mb,omitempty"`
	ReadonlyRoot     bool     `json:"readonly_root,omitempty"`
}

// HotCacheAction 本地快速放行规则。
type HotCacheAction struct {
	Executable      string   `json:"executable,omitempty"`
	ArgvAllowlist   []string `json:"argv_allowlist,omitempty"`
	RunAsSudo       bool     `json:"run_as_sudo,omitempty"`
}

// SudoHotCacheEntry Sudo 预授权条目。
type SudoHotCacheEntry struct {
	CommandPattern string `json:"command_pattern,omitempty"`
	RunAsUser      string `json:"run_as_user,omitempty"`
	Description    string `json:"description,omitempty"`
}

// sandboxProfileHandler 处理 GET /auth/sandbox-profile?resource=xxx，返回最小 SandboxProfile（Story 8.1）。
func (s *Server) sandboxProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		resource := r.URL.Query().Get("resource")
		if resource == "" {
			resource = "local://default"
		}
		profile := SandboxProfile{
			ProfileID:         "default",
			Version:           "1",
			Boundary:          &SandboxBoundary{NetworkEnabled: true, SyscallPreset: "default"},
			HotCacheActions:   nil,
			SudoHotCache:      nil,
			DegradationPolicy: "FAIL_CLOSE",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(profile)
	}
}

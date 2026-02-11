package proxy

import (
	"net/http/httptest"
	"testing"

	"diting/internal/audit"
	"diting/internal/cheq"
	"diting/internal/config"
	"diting/internal/delivery"
	"diting/internal/ownership"
	"diting/internal/policy"

	"github.com/gorilla/websocket"
)

func TestAuthStream_InitAndAuthAllow(t *testing.T) {
	cfg := &config.Config{}
	cfg.Proxy.ListenAddr = ":0"
	cfg.Proxy.Upstream = "http://localhost:9999"
	cfg.Proxy.AllowedAPIKeys = nil
	cfg.CHEQ.TimeoutSeconds = 60
	srv := NewServer(cfg, &policy.StubEngine{}, cheq.NewStubEngine(), &delivery.StubProvider{}, audit.NewStubStore(), &ownership.StubResolver{}, false)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	wsURL := "ws" + ts.URL[4:] + "/auth/stream"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer conn.Close()

	// 握手 init
	if err := conn.WriteJSON(AuthStreamRequest{
		RequestID: "req-1",
		Init:      &AuthStreamInit{ClientID: "test", Resource: "local://host"},
	}); err != nil {
		t.Fatalf("write init: %v", err)
	}
	var resp AuthStreamResponse
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read init resp: %v", err)
	}
	if resp.Pong != "pong" {
		t.Errorf("expected pong, got %q", resp.Pong)
	}

	// 鉴权 auth（Stub 策略恒 allow）
	if err := conn.WriteJSON(AuthStreamRequest{
		RequestID: "req-2",
		Auth: &ExecAuthRequest{
			Subject:     "test",
			Action:      "exec:run",
			Resource:    "local://host",
			CommandLine: "echo ok",
		},
	}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read auth resp: %v", err)
	}
	if resp.Immediate == nil {
		t.Fatal("expected immediate response")
	}
	if resp.Immediate.Decision != "allow" {
		t.Errorf("expected decision allow, got %q", resp.Immediate.Decision)
	}
}

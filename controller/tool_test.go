package controller

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

const testChatPath = "/v1/chat/completions"

// allowLocalUpstream opens the SSRF filter for the loopback test server.
func allowLocalUpstream(t *testing.T, port int) {
	fs := system_setting.GetFetchSetting()
	oldPrivate, oldPorts := fs.AllowPrivateIp, fs.AllowedPorts
	fs.AllowPrivateIp = true
	fs.AllowedPorts = append(append([]string{}, fs.AllowedPorts...), strconv.Itoa(port))
	t.Cleanup(func() { fs.AllowPrivateIp, fs.AllowedPorts = oldPrivate, oldPorts })
}

// newUpstream starts a loopback server on a random port and returns its base URL.
func newUpstream(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := httptest.NewUnstartedServer(handler)
	if err := srv.Listener.Close(); err != nil {
		t.Fatalf("close default listener: %v", err)
	}
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)
	port := ln.Addr().(*net.TCPAddr).Port
	allowLocalUpstream(t, port)
	return "http://127.0.0.1:" + strconv.Itoa(port)
}

func newToolRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	service.InitHttpClient()
	r := gin.New()
	r.Any("/tool/stream/*target", ToolProxyStream)
	r.Any("/tool/nonstream/*target", ToolProxyNonStream)
	return r
}

func sseChunk(content string, finish any) string {
	data, err := common.Marshal(map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion.chunk", "created": 1700000000, "model": "gpt-4",
		"choices": []any{map[string]any{
			"index": 0, "delta": map[string]any{"content": content}, "finish_reason": finish,
		}},
	})
	if err != nil {
		panic(err)
	}
	return "data: " + string(data) + "\n\n"
}

func streamUpstream() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, s := range []string{"Hello", ", ", "World"} {
			fmt.Fprint(w, sseChunk(s, nil))
		}
		last := "stop"
		fmt.Fprint(w, sseChunk("", &last))
		fmt.Fprint(w, "data: [DONE]\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

const nonStreamBody = `{"id":"chatcmpl-2","object":"chat.completion","created":1700000000,"model":"gpt-4",` +
	`"choices":[{"index":0,"message":{"role":"assistant","content":"Hi there"},"finish_reason":"stop"}]}`

func nonStreamUpstream() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(nonStreamBody))
	}
}

func doTool(t *testing.T, r *gin.Engine, mode, target string, stream bool) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":%t}`, stream)
	req := httptest.NewRequest(http.MethodPost, "/tool/"+mode+"/"+target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestToolParseTarget(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/https://api.openai.com/v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		{"http://api.openai.com/v1/chat/completions", "http://api.openai.com/v1/chat/completions"},
		{"/http:/api.openai.com/v1/chat/completions", "http://api.openai.com/v1/chat/completions"},
		{"/https:/api.openai.com/v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		{"/api.openai.com/v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		{"/https%3A%2F%2Fapi.openai.com%2Fv1%2Fchat%2Fcompletions", "https://api.openai.com/v1/chat/completions"},
	}
	for _, c := range cases {
		got, err := toolParseTarget(c.in, "")
		if err != nil {
			t.Fatalf("toolParseTarget(%q) error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("toolParseTarget(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToolParseTargetRejectsGarbage(t *testing.T) {
	for _, in := range []string{"", "/"} {
		if _, err := toolParseTarget(in, ""); err == nil {
			t.Errorf("toolParseTarget(%q) = nil error, want error", in)
		}
	}
}

func TestToolNonStreamUpstreamServedAsStream(t *testing.T) {
	up := newUpstream(t, nonStreamUpstream())
	w := doTool(t, newToolRouter(), "nonstream", url.PathEscape(up+testChatPath), true)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"content":"Hi there"`) {
		t.Errorf("stream missing content: %s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Errorf("stream missing [DONE]: %s", body)
	}
	if !strings.Contains(body, `"finish_reason":"stop"`) {
		t.Errorf("stream missing finish_reason: %s", body)
	}
}

func TestToolStreamUpstreamServedAsNonStream(t *testing.T) {
	up := newUpstream(t, streamUpstream())
	w := doTool(t, newToolRouter(), "stream", url.PathEscape(up+testChatPath), false)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"content":"Hello, World"`) {
		t.Errorf("aggregated content wrong: %s", body)
	}
	if !strings.Contains(body, `"object":"chat.completion"`) {
		t.Errorf("object should be chat.completion: %s", body)
	}
	if strings.Contains(body, "chat.completion.chunk") {
		t.Errorf("should not leak chunks: %s", body)
	}
}

func TestToolMatchingModePassesThrough(t *testing.T) {
	up := newUpstream(t, streamUpstream())
	w := doTool(t, newToolRouter(), "stream", url.PathEscape(up+testChatPath), true)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "data: [DONE]") {
		t.Errorf("expected raw SSE passthrough: %s", body)
	}
}

// Users may write the target with literal slashes instead of percent-encoding them.
func TestToolLiteralTargetURL(t *testing.T) {
	up := newUpstream(t, nonStreamUpstream())
	w := doTool(t, newToolRouter(), "nonstream", up+testChatPath, true)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"content":"Hi there"`) {
		t.Errorf("content missing: %s", body)
	}
}

func TestToolNonChatRequestPassesThrough(t *testing.T) {
	var gotPath, gotAuth string
	up := newUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	})

	r := newToolRouter()
	req := httptest.NewRequest(http.MethodGet, "/tool/nonstream/"+url.PathEscape(up+"/v1/models"), nil)
	req.Header.Set("Authorization", "Bearer sk-test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if gotPath != "/v1/models" {
		t.Errorf("upstream path = %q, want /v1/models", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization not forwarded upstream, got %q", gotAuth)
	}
}

func TestToolInvalidTargetReturns400(t *testing.T) {
	w := doTool(t, newToolRouter(), "stream", "", true)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (body: %s)", w.Code, w.Body.String())
	}
}

func TestToolBlockedTargetReturns403(t *testing.T) {
	fs := system_setting.GetFetchSetting()
	oldPrivate := fs.AllowPrivateIp
	fs.AllowPrivateIp = false
	t.Cleanup(func() { fs.AllowPrivateIp = oldPrivate })

	w := doTool(t, newToolRouter(), "nonstream", url.PathEscape("http://127.0.0.1:1234"+testChatPath), false)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

func TestToolUpstreamErrorIsForwarded(t *testing.T) {
	up := newUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	})

	w := doTool(t, newToolRouter(), "nonstream", url.PathEscape(up+testChatPath), true)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (body: %s)", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, "bad key") {
		t.Errorf("upstream error body not forwarded: %s", body)
	}
}

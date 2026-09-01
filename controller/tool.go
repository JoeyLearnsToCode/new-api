package controller

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// Public (no-auth) tool API: wraps an OpenAI-compatible upstream that only speaks
// one of stream/non-stream into an endpoint that speaks both.
// Same behaviour as the Cloudflare Worker in cf_worker.js.
//
//	/tool/stream/<target_url>     upstream only supports streaming
//	/tool/nonstream/<target_url>  upstream only supports non-streaming
//
// Non chat-completion traffic (models, embeddings, plain GET, ...) is proxied as-is.

const (
	toolUpstreamStream    = "stream"
	toolUpstreamNonStream = "nonstream"
)

// Hop-by-hop and client-scoped headers that must not reach the upstream.
var toolSkipRequestHeaders = map[string]bool{
	"host": true, "connection": true, "keep-alive": true, "proxy-authenticate": true,
	"proxy-authorization": true, "te": true, "trailer": true, "transfer-encoding": true,
	"upgrade": true, "content-length": true,
	// Let the transport negotiate encoding so it transparently decodes for us.
	"accept-encoding": true,
	"x-forwarded-for": true, "x-real-ip": true,
	"cf-ray": true, "cf-connecting-ip": true, "cf-ipcountry": true, "cf-visitor": true,
}

var toolSkipResponseHeaders = map[string]bool{
	"connection": true, "keep-alive": true, "transfer-encoding": true, "content-length": true,
}

// ToolProxyStream adapts an upstream that only supports streaming.
func ToolProxyStream(c *gin.Context) {
	toolProxy(c, toolUpstreamStream)
}

// ToolProxyNonStream adapts an upstream that only supports non-streaming.
func ToolProxyNonStream(c *gin.Context) {
	toolProxy(c, toolUpstreamNonStream)
}

func toolProxy(c *gin.Context, upstreamMode string) {
	targetURL, err := toolParseTarget(c.Param("target"), c.Request.URL.RawQuery)
	if err != nil {
		toolError(c, http.StatusBadRequest, "invalid_request_error",
			"Invalid URL format. Usage: /tool/stream/<target_url> or /tool/nonstream/<target_url>")
		return
	}

	if err := toolValidateTarget(targetURL); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("tool proxy target blocked: %s (%s)", targetURL, err.Error()))
		toolError(c, http.StatusForbidden, "invalid_request_error", "Target URL is not allowed")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		toolError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}

	chatBody, isChat := toolParseChatBody(c.Request.Method, body)
	if !isChat {
		toolPassthrough(c, targetURL, body)
		return
	}

	clientWantsStream := chatBody["stream"] == true
	chatBody["stream"] = upstreamMode == toolUpstreamStream
	upstreamBody, err := common.Marshal(chatBody)
	if err != nil {
		toolError(c, http.StatusInternalServerError, "server_error", "Failed to encode request body")
		return
	}

	resp, err := toolDoRequest(c, http.MethodPost, targetURL, upstreamBody, true)
	if err != nil {
		toolError(c, http.StatusBadGateway, "server_error", fmt.Sprintf("Upstream request failed: %s", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		toolCopyResponse(c, resp)
		return
	}

	switch {
	case clientWantsStream && upstreamMode == toolUpstreamNonStream:
		toolStreamify(c, resp)
	case !clientWantsStream && upstreamMode == toolUpstreamStream:
		toolCollect(c, resp)
	default:
		toolCopyResponse(c, resp)
	}
}

// toolParseTarget rebuilds the target URL from the catch-all path segment.
func toolParseTarget(raw, rawQuery string) (string, error) {
	target := strings.TrimPrefix(raw, "/")
	// A fully encoded target (https%3A%2F%2F...) needs one more decode.
	if unescaped, err := url.PathUnescape(target); err == nil && unescaped != target && strings.Contains(unescaped, "://") {
		target = unescaped
	}
	switch {
	case strings.HasPrefix(target, "http://"), strings.HasPrefix(target, "https://"):
	case strings.HasPrefix(target, "http:/"), strings.HasPrefix(target, "https:/"):
		// "http:/host/path" — restore the collapsed double slash.
		i := strings.Index(target, ":/")
		target = target[:i] + "://" + strings.TrimPrefix(target[i+2:], "/")
	default:
		target = "https://" + target
	}

	u, err := url.Parse(target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("invalid target url: %q", target)
	}
	if rawQuery != "" {
		if u.RawQuery == "" {
			u.RawQuery = rawQuery
		} else {
			u.RawQuery += "&" + rawQuery
		}
	}
	return u.String(), nil
}

// toolValidateTarget applies the project's existing SSRF/fetch restrictions.
func toolValidateTarget(targetURL string) error {
	fs := system_setting.GetFetchSetting()
	return common.ValidateURLWithFetchSetting(targetURL, fs.EnableSSRFProtection, fs.AllowPrivateIp,
		fs.DomainFilterMode, fs.IpFilterMode, fs.DomainList, fs.IpList, fs.AllowedPorts, fs.ApplyIPFilterForDomain)
}

// toolParseChatBody reports whether this is a chat completion request, mirroring
// the worker's rule: POST + valid JSON object carrying a messages array.
func toolParseChatBody(method string, body []byte) (map[string]any, bool) {
	if method != http.MethodPost || len(bytes.TrimSpace(body)) == 0 {
		return nil, false
	}
	var parsed map[string]any
	if err := common.Unmarshal(body, &parsed); err != nil {
		return nil, false
	}
	if _, ok := parsed["messages"].([]any); !ok {
		return nil, false
	}
	return parsed, true
}

func toolDoRequest(c *gin.Context, method, targetURL string, body []byte, forceJSON bool) (*http.Response, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, targetURL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, vs := range c.Request.Header {
		if toolSkipRequestHeaders[strings.ToLower(k)] {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if forceJSON {
		req.Header.Set("Content-Type", "application/json")
	} else if ct := c.Request.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	return service.GetHttpClient().Do(req)
}

// toolPassthrough proxies non chat requests untouched.
func toolPassthrough(c *gin.Context, targetURL string, body []byte) {
	resp, err := toolDoRequest(c, c.Request.Method, targetURL, body, false)
	if err != nil {
		toolError(c, http.StatusBadGateway, "server_error", fmt.Sprintf("Upstream request failed: %s", err.Error()))
		return
	}
	defer resp.Body.Close()
	toolCopyResponse(c, resp)
}

func toolCopyResponse(c *gin.Context, resp *http.Response) {
	for k, vs := range resp.Header {
		if toolSkipResponseHeaders[strings.ToLower(k)] {
			continue
		}
		for _, v := range vs {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

// toolStreamify turns a non-stream upstream response into an SSE stream.
func toolStreamify(c *gin.Context, resp *http.Response) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		toolError(c, http.StatusBadGateway, "server_error", "Failed to read upstream response")
		return
	}
	var chat toolChatResponse
	if err := common.Unmarshal(raw, &chat); err != nil {
		toolError(c, http.StatusBadGateway, "server_error", "Upstream returned a non-JSON response")
		return
	}

	id, created, model := chat.ID, chat.Created, chat.Model
	if id == "" {
		id = "chatcmpl-" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	if created == 0 {
		created = time.Now().Unix()
	}
	finishReason := "stop"
	content := ""
	if len(chat.Choices) > 0 {
		content = chat.Choices[0].Message.Content
		if chat.Choices[0].FinishReason != "" {
			finishReason = chat.Choices[0].FinishReason
		}
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, _ := c.Writer.(http.Flusher)
	write := func(v any) {
		data, err := common.Marshal(v)
		if err != nil {
			return
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
	}
	base := func(delta gin.H, finish any) gin.H {
		return gin.H{
			"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
			"choices": []gin.H{{"index": 0, "delta": delta, "finish_reason": finish}},
		}
	}

	write(base(gin.H{"role": "assistant", "content": ""}, nil))
	if content != "" {
		write(base(gin.H{"content": content}, nil))
	}
	write(base(gin.H{}, finishReason))
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// toolCollect aggregates an SSE upstream response into a single chat completion.
func toolCollect(c *gin.Context, resp *http.Response) {
	var (
		content      strings.Builder
		id, model    string
		created      int64
		finishReason string
	)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(line[len("data:"):])
		if data == "[DONE]" || data == "DONE" {
			break
		}
		var chunk toolStreamChunk
		if err := common.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if id == "" {
			id, created, model = chunk.ID, chunk.Created, chunk.Model
		}
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
			if chunk.Choices[0].FinishReason != nil {
				finishReason = *chunk.Choices[0].FinishReason
				if finishReason == "stop" {
					break
				}
			}
		}
	}

	if id == "" {
		id = "chatcmpl-" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	if created == 0 {
		created = time.Now().Unix()
	}
	if model == "" {
		model = "unknown"
	}
	if finishReason == "" {
		finishReason = "stop"
	}

	c.JSON(http.StatusOK, gin.H{
		"id": id, "object": "chat.completion", "created": created, "model": model,
		"choices": []gin.H{{
			"index":         0,
			"message":       gin.H{"role": "assistant", "content": content.String()},
			"finish_reason": finishReason,
		}},
		"usage": gin.H{"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0},
	})
}

type toolChatResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type toolStreamChunk struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func toolError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

package dto

type ChannelSettings struct {
	ForceFormat            bool   `json:"force_format,omitempty"`
	ThinkingToContent      bool   `json:"thinking_to_content,omitempty"`
	Proxy                  string `json:"proxy"`
	PassThroughBodyEnabled bool   `json:"pass_through_body_enabled,omitempty"`
	SystemPrompt           string `json:"system_prompt,omitempty"`
	SystemPromptOverride   bool   `json:"system_prompt_override,omitempty"`
	StreamSupport          string `json:"stream_support,omitempty"` // 流式支持配置：BOTH-支持流式和非流式（默认），STREAM_ONLY-仅支持流式，NON_STREAM_ONLY-仅支持非流式
	ExpirationTime         string `json:"expiration_time,omitempty"` // RFC3339 format with timezone, e.g., "2006-01-02T15:04:05Z07:00"
}

type VertexKeyType string

const (
	VertexKeyTypeJSON   VertexKeyType = "json"
	VertexKeyTypeAPIKey VertexKeyType = "api_key"
)

type ChannelOtherSettings struct {
	AzureResponsesVersion string        `json:"azure_responses_version,omitempty"`
	VertexKeyType         VertexKeyType `json:"vertex_key_type,omitempty"` // "json" or "api_key"
}

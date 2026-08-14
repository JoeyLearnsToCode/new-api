package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestGetFullRequestURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		requestURL  string
		channelType int
		want        string
	}{
		{
			name:        "openai default base url",
			baseURL:     "https://api.openai.com",
			requestURL:  "/v1/models",
			channelType: constant.ChannelTypeOpenAI,
			want:        "https://api.openai.com/v1/models",
		},
		{
			name:        "openai base url with trailing slash skips v1",
			baseURL:     "https://foo.bar/v2/",
			requestURL:  "/v1/models",
			channelType: constant.ChannelTypeOpenAI,
			want:        "https://foo.bar/v2/models",
		},
		{
			name:        "openai base url with trailing slash skips v1 for chat completions",
			baseURL:     "https://foo.bar/v2/",
			requestURL:  "/v1/chat/completions",
			channelType: constant.ChannelTypeOpenAI,
			want:        "https://foo.bar/v2/chat/completions",
		},
		{
			name:        "openai base url with trailing slash keeps non-v1 path",
			baseURL:     "https://foo.bar/v2/",
			requestURL:  "/openai/deployments/gpt-4/chat/completions",
			channelType: constant.ChannelTypeAzure,
			want:        "https://foo.bar/v2//openai/deployments/gpt-4/chat/completions",
		},
		{
			name:        "openai base url with trailing slash keeps v1beta path",
			baseURL:     "https://foo.bar/v2/",
			requestURL:  "/v1beta/models",
			channelType: constant.ChannelTypeOpenAI,
			want:        "https://foo.bar/v2//v1beta/models",
		},
		{
			name:        "openai base url with trailing slash and v1 only path",
			baseURL:     "https://foo.bar/v2/",
			requestURL:  "/v1",
			channelType: constant.ChannelTypeOpenAI,
			want:        "https://foo.bar/v2",
		},
		{
			name:        "cloudflare openai still skips v1",
			baseURL:     "https://gateway.ai.cloudflare.com/v1/account/gateway",
			requestURL:  "/v1/models",
			channelType: constant.ChannelTypeOpenAI,
			want:        "https://gateway.ai.cloudflare.com/v1/account/gateway/models",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFullRequestURL(tt.baseURL, tt.requestURL, tt.channelType); got != tt.want {
				t.Errorf("GetFullRequestURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
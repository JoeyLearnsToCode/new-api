package dto

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// SystemOneRequestPart 是 multipart 请求中承载 JSON 体的 part 名，由 TypeSafe 约定
const SystemOneRequestPart = "request"

// SystemOneRequest 是 TypeSafe System One API（POST /v1/systemone）的请求体。
// questions / assets 保留原始 JSON，避免上游新增字段时被丢弃。
type SystemOneRequest struct {
	Model     string                     `json:"model"`
	State     any                        `json:"state"`
	Questions map[string]json.RawMessage `json:"questions"`
	Assets    map[string]json.RawMessage `json:"assets,omitempty"`
}

// SystemOneAsset 是 assets 中单个素材，只取网关需要的字段
type SystemOneAsset struct {
	Type       string `json:"type"`
	Format     string `json:"format"`
	DataBase64 string `json:"data_base64"`
	Part       string `json:"part"`
}

func (r *SystemOneRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *SystemOneRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

// ParseAssets 解析 assets，解析失败的素材直接跳过
func (r *SystemOneRequest) ParseAssets() map[string]SystemOneAsset {
	assets := make(map[string]SystemOneAsset, len(r.Assets))
	for key, raw := range r.Assets {
		var asset SystemOneAsset
		if err := common.Unmarshal(raw, &asset); err != nil {
			continue
		}
		assets[key] = asset
	}
	return assets
}

// HasAssets 判断请求是否携带素材（图片等），用于自动开启 TypeSafe-Danger-Zone-Images。
func (r *SystemOneRequest) HasAssets() bool {
	return len(r.Assets) > 0
}

// GetTokenCountMeta 把 state 与 questions 拼成文本，用于 token 估算与敏感词检测；
// data_base64 形式的图片素材按图片计入 Files。
func (r *SystemOneRequest) GetTokenCountMeta() *types.TokenCountMeta {
	texts := make([]string, 0, len(r.Questions)+1)
	if r.State != nil {
		if stateBytes, err := common.Marshal(r.State); err == nil {
			texts = append(texts, string(stateBytes))
		}
	}
	for _, question := range r.Questions {
		texts = append(texts, string(question))
	}

	files := make([]*types.FileMeta, 0, len(r.Assets))
	for _, asset := range r.ParseAssets() {
		if asset.DataBase64 == "" {
			continue
		}
		mimeType := ""
		if asset.Format != "" {
			mimeType = "image/" + asset.Format
		}
		files = append(files, &types.FileMeta{
			FileType: types.FileTypeImage,
			MimeType: mimeType,
			Source:   types.NewBase64FileSource(asset.DataBase64, mimeType),
		})
	}

	return &types.TokenCountMeta{
		CombineText:   strings.Join(texts, "\n"),
		MessagesCount: len(r.Questions),
		Files:         files,
	}
}

type SystemOneUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type SystemOneResponse struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   SystemOneUsage             `json:"usage"`
}

// ToUsage 把 TypeSafe 的 usage 映射为 new-api 的通用用量：
// input_tokens -> PromptTokens，output_tokens -> CompletionTokens。
func (r *SystemOneResponse) ToUsage() *Usage {
	return &Usage{
		PromptTokens:     r.Usage.InputTokens,
		CompletionTokens: r.Usage.OutputTokens,
		TotalTokens:      r.Usage.InputTokens + r.Usage.OutputTokens,
	}
}

# DTO 模块文档

## 概述

DTO（Data Transfer Object）模块负责定义 One-API 系统中的数据传输对象，主要用于 API 请求和响应的数据结构定义。该模块是系统与各种 AI 服务提供商（OpenAI、Claude、Gemini 等）进行数据交换的核心组件。

## 模块结构

```
dto/
├── audio.go               # 音频相关的 DTO 定义
├── channel_settings.go    # 渠道设置相关 DTO
├── claude.go             # Claude API 相关 DTO
├── embedding.go          # 向量嵌入相关 DTO
├── error.go              # 错误响应 DTO
├── gemini.go             # Gemini API 相关 DTO
├── midjourney.go         # Midjourney API 相关 DTO
├── notify.go             # 通知相关 DTO
├── openai_image.go       # OpenAI 图像生成 DTO
├── openai_request.go     # OpenAI 请求 DTO（核心）
├── openai_response.go    # OpenAI 响应 DTO（核心）
├── playground.go         # 测试环境相关 DTO
├── pricing.go            # 定价相关 DTO
├── ratio_sync.go         # 比例同步相关 DTO
├── realtime.go           # 实时通信 DTO
├── request_common.go     # 通用请求接口定义
├── rerank.go             # 重排序相关 DTO
├── sensitive.go          # 敏感词检测 DTO
├── suno.go               # Suno API 相关 DTO
├── task.go               # 任务相关 DTO
├── user_settings.go      # 用户设置 DTO
└── video.go              # 视频相关 DTO
```

## 核心组件

### 1. 请求接口抽象 (request_common.go)

定义了所有 DTO 请求对象的通用接口：

```go
type Request interface {
    GetTokenCountMeta() *types.TokenCountMeta  // 获取 token 计数元数据
    IsStream(c *gin.Context) bool              // 判断是否为流式请求
    SetModelName(modelName string)             // 设置模型名称
}

type BaseRequest struct{}  // 基础请求结构体，提供默认实现
```

### 2. OpenAI 请求 DTO (openai_request.go)

#### GeneralOpenAIRequest
系统最重要的请求结构体，兼容 OpenAI API 格式并扩展支持其他服务商：

```go
type GeneralOpenAIRequest struct {
    Model               string            `json:"model,omitempty"`
    Messages            []Message         `json:"messages,omitempty"`
    Stream              bool              `json:"stream,omitempty"`
    MaxTokens           uint              `json:"max_tokens,omitempty"`
    MaxCompletionTokens uint              `json:"max_completion_tokens,omitempty"`
    Temperature         *float64          `json:"temperature,omitempty"`
    TopP                float64           `json:"top_p,omitempty"`
    Tools               []ToolCallRequest `json:"tools,omitempty"`
    // ... 更多字段支持各种服务商的特殊参数
}
```

**核心方法：**
- `GetTokenCountMeta()`: 计算请求的 token 使用情况
- `IsStream()`: 判断是否为流式请求
- `ParseInput()`: 解析输入内容

#### Message 结构体
消息对象，支持多模态内容：

```go
type Message struct {
    Role             string          `json:"role"`              // 角色：system/user/assistant
    Content          any             `json:"content"`           // 内容（文本或多媒体）
    Name             *string         `json:"name,omitempty"`    // 消息名称
    ToolCalls        json.RawMessage `json:"tool_calls,omitempty"` // 工具调用
    parsedContent    []MediaContent  // 解析后的媒体内容
}
```

**内容类型支持：**
- `ContentTypeText`: 文本内容
- `ContentTypeImageURL`: 图像 URL
- `ContentTypeInputAudio`: 输入音频
- `ContentTypeFile`: 文件
- `ContentTypeVideoUrl`: 视频 URL

#### MediaContent 多媒体内容结构
```go
type MediaContent struct {
    Type       string `json:"type"`                      // 内容类型
    Text       string `json:"text,omitempty"`            // 文本内容
    ImageUrl   any    `json:"image_url,omitempty"`       // 图像 URL
    InputAudio any    `json:"input_audio,omitempty"`     // 音频数据
    File       any    `json:"file,omitempty"`            // 文件数据
    VideoUrl   any    `json:"video_url,omitempty"`       // 视频 URL
}
```

### 3. OpenAI 响应 DTO (openai_response.go)

#### 流式响应结构
```go
type ChatCompletionsStreamResponse struct {
    Id      string                                `json:"id"`
    Object  string                                `json:"object"`
    Created int64                                 `json:"created"`
    Model   string                                `json:"model"`
    Choices []ChatCompletionsStreamResponseChoice `json:"choices"`
    Usage   *Usage                                `json:"usage"`
}
```

#### 使用统计结构
```go
type Usage struct {
    PromptTokens         int `json:"prompt_tokens"`
    CompletionTokens     int `json:"completion_tokens"`
    TotalTokens          int `json:"total_tokens"`
    PromptCacheHitTokens int `json:"prompt_cache_hit_tokens,omitempty"`
    
    // 详细 token 统计
    PromptTokensDetails    InputTokenDetails  `json:"prompt_tokens_details"`
    CompletionTokenDetails OutputTokenDetails `json:"completion_tokens_details"`
}
```

### 4. 错误处理 DTO (error.go)

#### OpenAIError
标准化的 OpenAI 错误格式：

```go
type OpenAIError struct {
    Message string `json:"message"`  // 错误信息
    Type    string `json:"type"`     // 错误类型
    Param   string `json:"param"`    // 相关参数
    Code    any    `json:"code"`     // 错误代码
}
```

#### GeneralErrorResponse
通用错误响应，兼容多种服务商的错误格式：

```go
type GeneralErrorResponse struct {
    Error    types.OpenAIError `json:"error"`
    Message  string            `json:"message"`
    Msg      string            `json:"msg"`
    Err      string            `json:"err"`
    ErrorMsg string            `json:"error_msg"`
}
```

### 5. 特定服务商 DTO

#### Claude DTO (claude.go)
```go
type ClaudeMediaMessage struct {
    Type         string               `json:"type,omitempty"`
    Text         *string              `json:"text,omitempty"`
    Model        string               `json:"model,omitempty"`
    Source       *ClaudeMessageSource `json:"source,omitempty"`
    Usage        *ClaudeUsage         `json:"usage,omitempty"`
    // ... Claude 特有字段
}
```

#### Gemini DTO (gemini.go)
```go
type GeminiChatRequest struct {
    Contents           []GeminiChatContent        `json:"contents"`
    SafetySettings     []GeminiChatSafetySettings `json:"safetySettings,omitempty"`
    GenerationConfig   GeminiChatGenerationConfig `json:"generationConfig,omitempty"`
    Tools              json.RawMessage            `json:"tools,omitempty"`
    SystemInstructions *GeminiChatContent         `json:"systemInstruction,omitempty"`
}
```

#### 嵌入向量 DTO (embedding.go)
```go
type EmbeddingRequest struct {
    Model            string   `json:"model"`
    Input            any      `json:"input"`
    EncodingFormat   string   `json:"encoding_format,omitempty"`
    Dimensions       int      `json:"dimensions,omitempty"`
    User             string   `json:"user,omitempty"`
}
```

## 关键特性

### 1. 多服务商兼容性
- **统一接口**: 通过 `GeneralOpenAIRequest` 统一不同服务商的请求格式
- **扩展字段**: 支持各服务商的特有参数（ExtraBody、SearchParameters 等）
- **灵活解析**: 动态处理不同格式的响应数据

### 2. 多模态支持
- **文本**: 标准文本消息处理
- **图像**: 支持 URL 和 base64 格式图像
- **音频**: 支持音频输入和处理
- **视频**: 支持视频内容识别
- **文件**: 支持通用文件处理

### 3. Token 计算
- **智能统计**: 准确计算不同内容类型的 token 消耗
- **多媒体计算**: 支持图像、音频、视频的 token 计算
- **工具调用**: 包含工具调用的 token 开销

### 4. 流式响应处理
- **实时传输**: 支持流式响应的实时处理
- **状态管理**: 跟踪流式响应的完成状态
- **工具调用**: 处理流式响应中的工具调用

## 使用示例

### 创建聊天请求
```go
request := &dto.GeneralOpenAIRequest{
    Model: "gpt-4",
    Messages: []dto.Message{
        {
            Role: "user",
            Content: "Hello, how are you?",
        },
    },
    Stream: true,
    MaxTokens: 1000,
    Temperature: &temperature,
}

// 获取 token 统计
tokenMeta := request.GetTokenCountMeta()
fmt.Printf("预计消耗 tokens: %d", tokenMeta.MaxTokens)
```

### 处理多模态消息
```go
message := dto.Message{
    Role: "user",
    Content: []dto.MediaContent{
        {
            Type: dto.ContentTypeText,
            Text: "请描述这张图片",
        },
        {
            Type: dto.ContentTypeImageURL,
            ImageUrl: &dto.MessageImageUrl{
                Url: "https://example.com/image.jpg",
                Detail: "high",
            },
        },
    },
}
```

### 错误处理
```go
func handleError(err error) {
    if openaiErr, ok := err.(*dto.OpenAIError); ok {
        log.Printf("OpenAI 错误: %s (类型: %s)", openaiErr.Message, openaiErr.Type)
    }
}
```

## 设计原则

### 1. 向后兼容性
- 保持与 OpenAI API 的完全兼容
- 新增字段使用 `omitempty` 标签
- 支持旧版本客户端

### 2. 扩展性
- 使用 `json.RawMessage` 处理未知字段
- 支持动态添加新的服务商参数
- 模块化的结构体设计

### 3. 性能优化
- 延迟解析：只在需要时解析复杂结构
- 内存复用：避免不必要的内存分配
- 高效序列化：优化 JSON 序列化性能

### 4. 类型安全
- 强类型定义避免运行时错误
- 接口抽象提供统一的操作方式
- 详细的字段标签和验证

## 依赖关系

### 内部依赖
- `one-api/types`: 基础类型定义
- `one-api/common`: 通用工具函数

### 外部依赖
- `github.com/gin-gonic/gin`: Web 框架，用于上下文处理
- `encoding/json`: JSON 序列化和反序列化

## 最佳实践

### 1. DTO 使用规范
```go
// 好的做法：使用接口进行类型断言
func ProcessRequest(req dto.Request) error {
    tokenMeta := req.GetTokenCountMeta()
    if req.IsStream(c) {
        return handleStreamRequest(req)
    }
    return handleNormalRequest(req)
}

// 避免：直接使用具体类型
func ProcessRequest(req *dto.GeneralOpenAIRequest) error {
    // 这样会降低代码的灵活性
}
```

### 2. 错误处理规范
```go
// 统一错误处理
func handleAPIError(err any) *dto.OpenAIError {
    return dto.GetOpenAIError(err)
}
```

### 3. 多模态内容处理
```go
// 安全的内容解析
func parseMessageContent(msg *dto.Message) []dto.MediaContent {
    return msg.ParseContent()  // 使用内置的解析方法
}
```

## 注意事项

1. **内存管理**: 大型多媒体内容可能占用大量内存，需要及时释放
2. **并发安全**: DTO 对象在并发环境中使用时需要注意线程安全
3. **版本兼容**: 新增字段时要考虑与旧版本的兼容性
4. **性能影响**: 复杂的 DTO 结构可能影响序列化性能

## 总结

DTO 模块是 One-API 系统的数据交换核心，通过统一的数据结构定义，实现了对多种 AI 服务商 API 的兼容支持。模块设计充分考虑了扩展性、性能和类型安全，为整个系统提供了稳定可靠的数据传输基础。
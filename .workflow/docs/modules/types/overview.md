# Types 模块文档

## 概述

Types 模块定义了 One-API 系统中的核心类型、常量和数据结构。该模块作为整个系统的类型基础，提供了错误处理、文件数据管理、请求元数据、中继格式定义等关键类型，确保系统各组件之间的类型一致性和数据安全性。

## 模块结构

```
types/
├── channel_error.go    # 渠道错误类型定义
├── error.go           # 错误处理核心类型
├── file_data.go       # 文件数据类型定义
├── price_data.go      # 定价数据类型
├── relay_format.go    # 中继格式常量定义
├── request_meta.go    # 请求元数据类型
└── set.go            # 泛型集合类型
```

## 核心组件

### 1. 错误处理系统 (error.go)

这是系统错误处理的核心，定义了统一的错误类型和处理机制。

#### OpenAIError 标准错误格式
```go
type OpenAIError struct {
    Message string `json:"message"`  // 错误信息
    Type    string `json:"type"`     // 错误类型
    Param   string `json:"param"`    // 相关参数
    Code    any    `json:"code"`     // 错误代码
}
```

#### ClaudeError Claude 专用错误格式
```go
type ClaudeError struct {
    Type    string `json:"type,omitempty"`    // 错误类型
    Message string `json:"message,omitempty"` // 错误信息
}
```

#### ErrorType 错误类型常量
```go
type ErrorType string

const (
    ErrorTypeNewAPIError     ErrorType = "new_api_error"     // 系统内部错误
    ErrorTypeOpenAIError     ErrorType = "openai_error"      // OpenAI API 错误
    ErrorTypeClaudeError     ErrorType = "claude_error"      // Claude API 错误
    ErrorTypeMidjourneyError ErrorType = "midjourney_error"  // Midjourney API 错误
    ErrorTypeGeminiError     ErrorType = "gemini_error"      // Gemini API 错误
    ErrorTypeRerankError     ErrorType = "rerank_error"      // 重排序错误
    ErrorTypeUpstreamError   ErrorType = "upstream_error"    // 上游服务错误
)
```

#### ErrorCode 详细错误代码
```go
type ErrorCode string

const (
    // 请求相关错误
    ErrorCodeInvalidRequest         ErrorCode = "invalid_request"
    ErrorCodeSensitiveWordsDetected ErrorCode = "sensitive_words_detected"
    
    // 系统内部错误
    ErrorCodeCountTokenFailed   ErrorCode = "count_token_failed"
    ErrorCodeModelPriceError    ErrorCode = "model_price_error"
    ErrorCodeInvalidApiType     ErrorCode = "invalid_api_type"
    
    // 渠道相关错误
    ErrorCodeChannelNoAvailableKey        ErrorCode = "channel:no_available_key"
    ErrorCodeChannelParamOverrideInvalid  ErrorCode = "channel:param_override_invalid"
    ErrorCodeChannelHeaderOverrideInvalid ErrorCode = "channel:header_override_invalid"
    
    // 响应相关错误
    ErrorCodeReadResponseBodyFailed ErrorCode = "read_response_body_failed"
    ErrorCodeBadResponseStatusCode  ErrorCode = "bad_response_status_code"
    ErrorCodeEmptyResponse          ErrorCode = "empty_response"
    
    // 配额相关错误
    ErrorCodeInsufficientUserQuota      ErrorCode = "insufficient_user_quota"
    ErrorCodePreConsumeTokenQuotaFailed ErrorCode = "pre_consume_token_quota_failed"
)
```

#### NewAPIError 统一错误结构
```go
type NewAPIError struct {
    Err            error     // 原始错误
    RelayError     any       // 转发的错误对象
    skipRetry      bool      // 是否跳过重试
    recordErrorLog *bool     // 是否记录错误日志
    errorType      ErrorType // 错误类型
    errorCode      ErrorCode // 错误代码
    StatusCode     int       // HTTP 状态码
}
```

**关键方法：**
- `Error()`: 返回错误信息字符串
- `MaskSensitiveError()`: 屏蔽敏感信息的错误信息
- `ToOpenAIError()`: 转换为 OpenAI 格式错误
- `ToClaudeError()`: 转换为 Claude 格式错误

### 2. 文件数据管理 (file_data.go & request_meta.go)

#### LocalFileData 本地文件数据结构
```go
type LocalFileData struct {
    MimeType   string  // MIME 类型
    Base64Data string  // Base64 编码的文件数据
    Url        string  // 文件 URL
    Size       int64   // 文件大小
}
```

#### FileType 文件类型常量
```go
type FileType string

const (
    FileTypeImage FileType = "image" // 图像文件
    FileTypeAudio FileType = "audio" // 音频文件
    FileTypeVideo FileType = "video" // 视频文件
    FileTypeFile  FileType = "file"  // 通用文件
)
```

#### TokenType Token 类型定义
```go
type TokenType string

const (
    TokenTypeTextNumber TokenType = "text_number" // 文本或数字 token
    TokenTypeTokenizer  TokenType = "tokenizer"   // 分词器 token
    TokenTypeImage      TokenType = "image"       // 图像 token
)
```

#### TokenCountMeta Token 计数元数据
```go
type TokenCountMeta struct {
    TokenType     TokenType   `json:"token_type,omitempty"`     // Token 类型
    CombineText   string      `json:"combine_text,omitempty"`   // 合并的文本内容
    ToolsCount    int         `json:"tools_count,omitempty"`    // 工具数量
    NameCount     int         `json:"name_count,omitempty"`     // 名称数量
    MessagesCount int         `json:"messages_count,omitempty"` // 消息数量
    Files         []*FileMeta `json:"files,omitempty"`          // 文件列表
    MaxTokens     int         `json:"max_tokens,omitempty"`     // 最大 token 数
    ImagePriceRatio float64   `json:"image_ratio,omitempty"`    // 图像价格比率
}
```

#### FileMeta 文件元数据
```go
type FileMeta struct {
    FileType                    // 继承文件类型
    MimeType   string          // MIME 类型
    OriginData string          // 原始数据（URL 或 base64）
    Detail     string          // 详细信息
    ParsedData *LocalFileData  // 解析后的数据
}
```

#### RequestMeta 请求元数据
```go
type RequestMeta struct {
    OriginalModelName string `json:"original_model_name"` // 原始模型名称
    UserUsingGroup    string `json:"user_using_group"`    // 用户使用的组
    PromptTokens      int    `json:"prompt_tokens"`       // 提示 token 数
    PreConsumedQuota  int    `json:"pre_consumed_quota"`  // 预消费配额
}
```

### 3. 中继格式定义 (relay_format.go)

定义了系统支持的各种 API 格式类型：

```go
type RelayFormat string

const (
    RelayFormatOpenAI          RelayFormat = "openai"           // OpenAI 格式
    RelayFormatClaude                     = "claude"           // Claude 格式
    RelayFormatGemini                     = "gemini"           // Gemini 格式
    RelayFormatOpenAIResponses            = "openai_responses" // OpenAI Responses API
    RelayFormatOpenAIAudio                = "openai_audio"     // OpenAI 音频 API
    RelayFormatOpenAIImage                = "openai_image"     // OpenAI 图像 API
    RelayFormatOpenAIRealtime             = "openai_realtime"  // OpenAI 实时 API
    RelayFormatRerank                     = "rerank"           // 重排序 API
    RelayFormatEmbedding                  = "embedding"        // 嵌入向量 API
    RelayFormatTask                       = "task"             // 任务 API
    RelayFormatMjProxy                    = "mj_proxy"         // Midjourney 代理
)
```

### 4. 定价数据类型 (price_data.go)

#### GroupRatioInfo 组比率信息
```go
type GroupRatioInfo struct {
    GroupRatio        float64 // 组比率
    GroupSpecialRatio float64 // 组特殊比率
    HasSpecialRatio   bool    // 是否有特殊比率
}
```

#### PriceData 定价数据
```go
type PriceData struct {
    ModelPrice             float64        // 模型价格
    ModelRatio             float64        // 模型比率
    CompletionRatio        float64        // 完成比率
    CacheRatio             float64        // 缓存比率
    CacheCreationRatio     float64        // 缓存创建比率
    ImageRatio             float64        // 图像比率
    AudioRatio             float64        // 音频比率
    AudioCompletionRatio   float64        // 音频完成比率
    UsePrice               bool           // 是否使用价格
    ShouldPreConsumedQuota int            // 应预消费配额
    GroupRatioInfo         GroupRatioInfo // 组比率信息
}
```

#### PerCallPriceData 按调用计价数据
```go
type PerCallPriceData struct {
    ModelPrice     float64        // 模型价格
    Quota          int            // 配额
    GroupRatioInfo GroupRatioInfo // 组比率信息
}
```

### 5. 渠道错误类型 (channel_error.go)

```go
type ChannelError struct {
    ChannelId   int    `json:"channel_id"`   // 渠道 ID
    ChannelType int    `json:"channel_type"` // 渠道类型
    ChannelName string `json:"channel_name"` // 渠道名称
    IsMultiKey  bool   `json:"is_multi_key"` // 是否多密钥
    AutoBan     bool   `json:"auto_ban"`     // 是否自动禁用
    UsingKey    string `json:"using_key"`    // 使用的密钥
}
```

### 6. 泛型集合类型 (set.go)

提供了类型安全的集合数据结构：

```go
type Set[T comparable] struct {
    items map[T]struct{}
}

// 主要方法
func NewSet[T comparable]() *Set[T]           // 创建新集合
func (s *Set[T]) Add(item T)                  // 添加元素
func (s *Set[T]) Remove(item T)               // 移除元素
func (s *Set[T]) Contains(item T) bool        // 检查是否包含
func (s *Set[T]) Len() int                    // 获取长度
func (s *Set[T]) Items() []T                  // 获取所有元素
```

## 关键特性

### 1. 统一错误处理
- **多格式支持**: 支持 OpenAI、Claude 等多种错误格式
- **错误转换**: 自动在不同错误格式间转换
- **敏感信息屏蔽**: 自动屏蔽敏感信息
- **错误分类**: 详细的错误类型和代码分类

### 2. 文件数据管理
- **多格式支持**: 支持图像、音频、视频、通用文件
- **数据解析**: 自动解析 URL 和 base64 数据
- **元数据管理**: 完整的文件元数据跟踪
- **大小计算**: 准确的文件大小计算

### 3. Token 计算支持
- **多类型 Token**: 支持文本、图像等多种 token 类型
- **精确计算**: 准确计算各种内容的 token 消耗
- **元数据跟踪**: 详细的 token 使用元数据

### 4. 类型安全
- **强类型定义**: 所有类型都有明确的定义
- **泛型支持**: 使用 Go 泛型提供类型安全的集合
- **常量定义**: 使用常量避免魔法字符串

## 使用示例

### 错误处理示例
```go
// 创建 OpenAI 错误
err := types.NewOpenAIError(
    errors.New("invalid model"), 
    types.ErrorCodeModelNotFound, 
    404,
)

// 转换错误格式
openaiErr := err.ToOpenAIError()
claudeErr := err.ToClaudeError()

// 检查错误类型
if types.IsChannelError(err) {
    log.Println("这是渠道相关错误")
}
```

### 文件数据处理示例
```go
// 创建文件元数据
fileMeta := &types.FileMeta{
    FileType:   types.FileTypeImage,
    MimeType:   "image/jpeg",
    OriginData: "data:image/jpeg;base64,/9j/4AAQ...",
    Detail:     "high",
}

// Token 计算元数据
tokenMeta := &types.TokenCountMeta{
    TokenType:     types.TokenTypeImage,
    CombineText:   "请描述这张图片",
    MessagesCount: 1,
    Files:         []*types.FileMeta{fileMeta},
    MaxTokens:     1000,
}
```

### 集合操作示例
```go
// 创建字符串集合
modelSet := types.NewSet[string]()
modelSet.Add("gpt-4")
modelSet.Add("gpt-3.5-turbo")
modelSet.Add("claude-3")

// 检查是否包含
if modelSet.Contains("gpt-4") {
    fmt.Println("支持 GPT-4 模型")
}

// 获取所有模型
models := modelSet.Items()
fmt.Printf("支持的模型: %v", models)
```

### 定价计算示例
```go
// 创建定价数据
priceData := types.PriceData{
    ModelPrice:      0.03,
    ModelRatio:      1.0,
    CompletionRatio: 2.0,
    UsePrice:        true,
    GroupRatioInfo: types.GroupRatioInfo{
        GroupRatio:      1.2,
        HasSpecialRatio: false,
    },
}

// 输出定价设置
fmt.Println(priceData.ToSetting())
```

## 设计原则

### 1. 类型安全
- 使用强类型定义避免运行时错误
- 利用 Go 的类型系统进行编译时检查
- 提供类型安全的泛型集合

### 2. 扩展性
- 使用接口和抽象类型支持扩展
- 常量定义便于添加新的类型
- 结构体设计支持向后兼容

### 3. 一致性
- 统一的错误处理机制
- 一致的命名规范
- 标准化的数据结构

### 4. 性能优化
- 高效的集合实现
- 最小化内存分配
- 优化的数据结构设计

## 依赖关系

### 内部依赖
- `one-api/common`: 通用工具函数（用于敏感信息屏蔽等）

### 外部依赖
- `errors`: 标准错误处理
- `fmt`: 格式化输出
- `net/http`: HTTP 状态码常量

## 最佳实践

### 1. 错误处理
```go
// 好的做法：使用类型化的错误创建
func handleAPIError(err error) *types.NewAPIError {
    return types.NewError(err, types.ErrorCodeBadResponse)
}

// 避免：直接使用字符串错误
func handleAPIError(err error) error {
    return errors.New("API error: " + err.Error())
}
```

### 2. 类型使用
```go
// 好的做法：使用常量
format := types.RelayFormatOpenAI

// 避免：使用魔法字符串
format := "openai"
```

### 3. 集合操作
```go
// 好的做法：使用类型安全的集合
supportedModels := types.NewSet[string]()
supportedModels.Add("gpt-4")

// 避免：使用 map[string]bool
supportedModels := make(map[string]bool)
supportedModels["gpt-4"] = true
```

## 注意事项

1. **错误信息安全**: 错误信息可能包含敏感信息，需要适当屏蔽
2. **类型转换**: 不同错误格式间的转换需要保持信息完整性
3. **内存管理**: 文件数据可能占用大量内存，需要及时释放
4. **并发安全**: 集合类型在并发环境中使用需要额外的同步机制

## 总结

Types 模块是 One-API 系统的类型基础，提供了完整的类型定义体系。通过统一的错误处理、灵活的文件数据管理、精确的 token 计算支持以及类型安全的集合操作，为整个系统提供了稳固的类型基础。模块设计充分考虑了类型安全、扩展性和性能，是系统架构的重要组成部分。
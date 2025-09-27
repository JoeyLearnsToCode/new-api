# Constant 模块 - 系统常量定义

## 模块概述

Constant 模块定义了 New API 系统中所有的常量值，包括 API 类型、通道类型、缓存键、上下文键、任务类型、多键模式等。该模块为整个系统提供了标准化的常量定义，确保系统各组件之间的一致性和可维护性。

## 核心功能

### 1. API 提供商类型定义
- 支持 30+ 主流 AI API 提供商
- 统一的 API 类型枚举
- API 提供商标识管理

### 2. 通道类型管理
- 通道类型常量定义
- 通道基础 URL 配置
- 流式支持模式定义

### 3. 系统键值定义
- 缓存键标准化
- 上下文键管理
- 环境变量键定义

### 4. 业务常量定义
- 任务类型枚举
- 多键模式配置
- 特定服务配置

## 文件结构

```javascript
constant/
├── api_type.go        # API 类型常量定义
├── channel.go         # 通道类型和配置
├── cache_key.go       # 缓存键定义
├── context_key.go     # 上下文键定义
├── env.go            # 环境变量键
├── task.go           # 任务类型定义
├── multi_key_mode.go # 多键模式配置
├── finish_reason.go  # 完成原因枚举
├── setup.go          # 初始化配置
├── azure.go          # Azure 特定配置
└── midjourney.go     # Midjourney 特定配置
```

## 详细功能说明

### API 类型定义 (api_type.go)

系统支持的 AI API 提供商类型：

```go
const (
    APITypeOpenAI = iota      // 0  - OpenAI
    APITypeAnthropic          // 1  - Anthropic (Claude)
    APITypePaLM               // 2  - Google PaLM
    APITypeBaidu              // 3  - 百度文心一言
    APITypeZhipu              // 4  - 智谱 AI
    APITypeAli                // 5  - 阿里云通义千问
    APITypeXunfei             // 6  - 科大讯飞
    APITypeAIProxyLibrary     // 7  - AI Proxy Library
    APITypeTencent            // 8  - 腾讯混元
    APITypeGemini             // 9  - Google Gemini
    APITypeZhipuV4           // 10 - 智谱 AI V4
    APITypeOllama            // 11 - Ollama
    APITypePerplexity        // 12 - Perplexity
    APITypeAws               // 13 - Amazon Bedrock
    APITypeCohere            // 14 - Cohere
    APITypeDify              // 15 - Dify
    APITypeJina              // 16 - Jina AI
    APITypeCloudflare        // 17 - Cloudflare Workers AI
    APITypeSiliconFlow       // 18 - SiliconFlow
    APITypeVertexAi          // 19 - Google Vertex AI
    APITypeMistral           // 20 - Mistral AI
    APITypeDeepSeek          // 21 - DeepSeek
    APITypeMokaAI            // 22 - MokaAI
    APITypeVolcEngine        // 23 - 火山引擎
    APITypeBaiduV2           // 24 - 百度千帆 V2
    APITypeOpenRouter        // 25 - OpenRouter
    APITypeXinference        // 26 - Xinference
    APITypeXai               // 27 - xAI (Grok)
    APITypeCoze              // 28 - 字节跳动 Coze
    APITypeJimeng            // 29 - 即梦 AI
    APITypeMoonshot          // 30 - 月之暗面 Kimi
    APITypeDummy             // 31 - 仅用于计数，不要在此之后添加通道
)
```

### 通道类型和配置 (channel.go)

#### 通道类型定义
```go
const (
    ChannelTypeUnknown        = 0   // 未知通道
    ChannelTypeOpenAI         = 1   // OpenAI 标准接口
    ChannelTypeMidjourney     = 2   // Midjourney 绘图
    ChannelTypeAzure          = 3   // Azure OpenAI
    ChannelTypeOllama         = 4   // Ollama 本地部署
    ChannelTypeMidjourneyPlus = 5   // Midjourney Plus
    ChannelTypeOpenAIMax      = 6   // OpenAI Max
    ChannelTypeOhMyGPT        = 7   // OhMyGPT
    ChannelTypeCustom         = 8   // 自定义通道
    // ... 更多通道类型
    ChannelTypeDummy          // 计数用，不要在此之后添加
)
```

#### 流式支持模式
```go
const (
    StreamSupportBoth      = "BOTH"           // 支持流式和非流式（默认）
    StreamSupportOnly      = "STREAM_ONLY"    // 仅支持流式
    StreamSupportNonStream = "NON_STREAM_ONLY" // 仅支持非流式
)
```

#### 通道基础 URL 配置
```go
var ChannelBaseURLs = []string{
    "",                                          // 0
    "https://api.openai.com",                   // 1  - OpenAI
    "https://oa.api2d.net",                     // 2  - API2D
    "",                                         // 3
    "http://localhost:11434",                   // 4  - Ollama
    "https://api.openai-sb.com",               // 5  - OpenAI-SB
    "https://api.openaimax.com",               // 6  - OpenAIMax
    "https://api.ohmygpt.com",                 // 7  - OhMyGPT
    "",                                        // 8
    "https://api.caipacity.com",               // 9  - Caipacity
    "https://api.aiproxy.io",                  // 10 - AIProxy
    "",                                        // 11
    "https://api.api2gpt.com",                 // 12 - API2GPT
    "https://api.aigc2d.com",                  // 13 - AIGC2D
    "https://api.anthropic.com",               // 14 - Anthropic
    "https://aip.baidubce.com",                // 15 - 百度
    "https://open.bigmodel.cn",                // 16 - 智谱
    "https://dashscope.aliyuncs.com",          // 17 - 阿里云
    "",                                        // 18
    "https://api.360.cn",                      // 19 - 360
    "https://openrouter.ai/api",               // 20 - OpenRouter
    // ... 更多 URL 配置
}
```

### 缓存键定义 (cache_key.go)

标准化的缓存键定义，用于 Redis 缓存管理：

```go
const (
    // 用户相关缓存键
    CacheKeyUserQuota     = "user_quota:"      // 用户配额缓存
    CacheKeyUserInfo      = "user_info:"       // 用户信息缓存
    CacheKeyUserToken     = "user_token:"      // 用户令牌缓存
    
    // 通道相关缓存键
    CacheKeyChannelInfo   = "channel_info:"    // 通道信息缓存
    CacheKeyChannelTest   = "channel_test:"    // 通道测试缓存
    
    // 模型相关缓存键
    CacheKeyModelInfo     = "model_info:"      // 模型信息缓存
    CacheKeyModelPrice    = "model_price:"     // 模型价格缓存
    
    // 系统相关缓存键
    CacheKeySystemConfig  = "system_config:"   // 系统配置缓存
    CacheKeyRateLimit     = "rate_limit:"      // 速率限制缓存
)
```

### 上下文键定义 (context_key.go)

HTTP 请求上下文中使用的键：

```go
const (
    ContextKeyUser      = "user"       // 用户信息
    ContextKeyToken     = "token"      // 令牌信息
    ContextKeyChannel   = "channel"    // 通道信息
    ContextKeyRequestId = "request_id" // 请求 ID
    ContextKeyStartTime = "start_time" // 开始时间
    ContextKeyModel     = "model"      // 模型信息
)
```

### 环境变量键 (env.go)

系统环境变量键定义：

```go
const (
    EnvSessionSecret    = "SESSION_SECRET"     // 会话密钥
    EnvSQLDSN          = "SQL_DSN"            // 数据库连接串
    EnvRedisConnString = "REDIS_CONN_STRING"  // Redis 连接串
    EnvPort            = "PORT"               // 服务端口
    EnvLogDir          = "LOG_DIR"            // 日志目录
    EnvDebug           = "DEBUG"              // 调试模式
    EnvRootUserEmail   = "ROOT_USER_EMAIL"    // 根用户邮箱
    EnvRootUserPassword = "ROOT_USER_PASSWORD" // 根用户密码
)
```

### 任务类型定义 (task.go)

系统支持的任务类型：

```go
const (
    TaskTypeUnknown     = 0  // 未知任务
    TaskTypeTextGen     = 1  // 文本生成
    TaskTypeImageGen    = 2  // 图像生成
    TaskTypeAudioGen    = 3  // 音频生成
    TaskTypeVideoGen    = 4  // 视频生成
    TaskTypeEmbedding   = 5  // 向量嵌入
    TaskTypeModeration  = 6  // 内容审核
    TaskTypeAssistant   = 7  // 助手对话
    TaskTypeFineTuning  = 8  // 模型微调
    TaskTypeBatch       = 9  // 批量处理
    TaskTypeRealtime    = 10 // 实时对话
)
```

### 多键模式配置 (multi_key_mode.go)

API 密钥轮询模式：

```go
const (
    MultiKeyModeRoundRobin = 0  // 轮询模式
    MultiKeyModeRandom     = 1  // 随机模式
    MultiKeyModeWeighted   = 2  // 权重模式
    MultiKeyModeFailover   = 3  // 故障转移模式
)
```

### 完成原因枚举 (finish_reason.go)

AI 响应完成的原因：

```go
const (
    FinishReasonStop        = "stop"         // 正常停止
    FinishReasonLength      = "length"       // 达到长度限制
    FinishReasonToolCalls   = "tool_calls"   // 工具调用
    FinishReasonContentFilter = "content_filter" // 内容过滤
    FinishReasonError       = "error"        // 错误终止
)
```

## 使用示例

### API 类型判断
```go
import "one-api/constant"

func handleRequest(apiType int) {
    switch apiType {
    case constant.APITypeOpenAI:
        // 处理 OpenAI 请求
    case constant.APITypeAnthropic:
        // 处理 Anthropic 请求
    case constant.APITypeGemini:
        // 处理 Gemini 请求
    default:
        // 未知 API 类型
    }
}
```

### 通道配置获取
```go
func getChannelBaseURL(channelType int) string {
    if channelType >= 0 && channelType < len(constant.ChannelBaseURLs) {
        return constant.ChannelBaseURLs[channelType]
    }
    return ""
}
```

### 缓存键使用
```go
import (
    "fmt"
    "one-api/constant"
)

func getCacheKey(userID int) string {
    return fmt.Sprintf("%s%d", constant.CacheKeyUserQuota, userID)
}
```

### 上下文键使用
```go
import (
    "context"
    "one-api/constant"
)

func setUserInContext(ctx context.Context, user *User) context.Context {
    return context.WithValue(ctx, constant.ContextKeyUser, user)
}

func getUserFromContext(ctx context.Context) *User {
    if user := ctx.Value(constant.ContextKeyUser); user != nil {
        return user.(*User)
    }
    return nil
}
```

## 配置管理

### Azure 特定配置 (azure.go)
```go
const (
    AzureAPIVersion2023_05_15    = "2023-05-15"
    AzureAPIVersion2023_06_01    = "2023-06-01-preview"
    AzureAPIVersion2023_07_01    = "2023-07-01-preview"
    AzureAPIVersion2023_10_01    = "2023-10-01-preview"
    AzureAPIVersion2024_02_15    = "2024-02-15-preview"
)

const (
    AzureDeploymentTypeGPT35Turbo = "gpt-35-turbo"
    AzureDeploymentTypeGPT4       = "gpt-4"
    AzureDeploymentTypeGPT4Vision = "gpt-4-vision-preview"
)
```

### Midjourney 特定配置 (midjourney.go)
```go
const (
    MidjourneyActionImagine    = "imagine"
    MidjourneyActionUpscale    = "upscale"
    MidjourneyActionVariation  = "variation"
    MidjourneyActionReroll     = "reroll"
    MidjourneyActionDescribe   = "describe"
    MidjourneyActionBlend      = "blend"
)

const (
    MidjourneyStatusPending    = "pending"
    MidjourneyStatusRunning    = "running"
    MidjourneyStatusSuccess    = "success"
    MidjourneyStatusFailed     = "failed"
)
```

## API 接口

### 常量查询接口
```go
// 获取 API 类型名称
func GetAPITypeName(apiType int) string

// 获取通道类型名称
func GetChannelTypeName(channelType int) string

// 获取通道基础 URL
func GetChannelBaseURL(channelType int) string

// 判断是否支持流式
func IsStreamSupported(channelType int) string
```

### 验证接口
```go
// 验证 API 类型是否有效
func IsValidAPIType(apiType int) bool

// 验证通道类型是否有效
func IsValidChannelType(channelType int) bool

// 验证任务类型是否有效
func IsValidTaskType(taskType int) bool
```

## 最佳实践

### 1. 常量使用
```go
// 好的做法：使用常量
if apiType == constant.APITypeOpenAI {
    // 处理 OpenAI
}

// 避免：使用魔术数字
if apiType == 0 {
    // 不清楚这是什么类型
}
```

### 2. 缓存键管理
```go
// 好的做法：使用预定义的缓存键前缀
key := fmt.Sprintf("%s%d", constant.CacheKeyUserQuota, userID)

// 避免：硬编码缓存键
key := fmt.Sprintf("user_quota:%d", userID)
```

### 3. 上下文键使用
```go
// 好的做法：使用类型安全的上下文键
user := ctx.Value(constant.ContextKeyUser).(*User)

// 避免：使用字符串作为上下文键
user := ctx.Value("user").(*User)
```

### 4. 环境变量访问
```go
// 好的做法：使用预定义的环境变量键
port := os.Getenv(constant.EnvPort)

// 避免：硬编码环境变量名
port := os.Getenv("PORT")
```

## 扩展指南

### 添加新的 API 类型
1. 在 `api_type.go` 中添加新的常量
2. 确保添加在 `APITypeDummy` 之前
3. 更新相关的映射和验证函数

### 添加新的通道类型
1. 在 `channel.go` 中添加通道类型常量
2. 在 `ChannelBaseURLs` 中添加对应的基础 URL
3. 配置流式支持模式

### 添加新的缓存键
1. 在 `cache_key.go` 中定义缓存键前缀
2. 遵循命名规范：`CacheKey + 功能名 + ":"`
3. 添加相应的文档说明

## 注意事项

1. **向后兼容性**: 修改现有常量值会破坏向后兼容性，需要谨慎处理
2. **常量顺序**: API 类型和通道类型的顺序很重要，不能随意调整
3. **URL 配置**: 通道基础 URL 的索引必须与通道类型常量对应
4. **缓存键命名**: 缓存键应该具有描述性且不易冲突
5. **环境变量**: 环境变量键应该遵循大写+下划线的命名规范

## 依赖关系

Constant 模块被以下模块依赖：
- Common 模块：使用 API 类型和通道类型
- Model 模块：使用缓存键和上下文键
- Controller 模块：使用各种常量进行业务逻辑判断
- Middleware 模块：使用上下文键和缓存键
- Provider 模块：使用 API 类型和通道配置

该模块是系统常量的统一定义中心，为整个系统提供标准化的常量值，确保系统各组件之间的一致性。
# Controller 模块文档

## 概述

Controller 模块是 new-api 项目的 HTTP 控制器层，负责处理所有的 API 请求和响应。该模块实现了 RESTful API 接口，提供渠道管理、计费管理、用户认证等核心功能。

## 模块结构

```
controller/
├── billing.go              # 计费相关接口
├── channel-billing.go       # 渠道计费接口
├── channel-test.go          # 渠道测试接口
├── channel.go              # 渠道管理核心接口
├── console_migrate.go       # 控制台迁移接口
├── github.go               # GitHub OAuth 接口
├── group.go                # 用户组管理接口
├── image.go                # 图像处理接口
├── linuxdo.go              # LinuxDo 集成接口
├── log.go                  # 日志管理接口
├── midjourney.go           # Midjourney 集成接口
├── misc.go                 # 杂项功能接口
├── missing_models.go       # 缺失模型管理接口
├── model_meta.go           # 模型元数据接口
└── model_sync.go           # 模型同步接口
```

## 核心功能

### 1. 渠道管理 (channel.go)

#### 主要接口

| 接口 | 方法 | 路径 | 功能描述 |
|------|------|------|----------|
| GetAllChannels | GET | `/api/channel` | 获取所有渠道列表，支持分页和筛选 |
| GetChannel | GET | `/api/channel/:id` | 获取指定渠道详情 |
| AddChannel | POST | `/api/channel` | 添加新渠道，支持单个/批量/多密钥模式 |
| UpdateChannel | PUT | `/api/channel/:id` | 更新渠道配置 |
| DeleteChannel | DELETE | `/api/channel/:id` | 删除指定渠道 |
| SearchChannels | GET | `/api/channel/search` | 搜索渠道 |
| FetchUpstreamModels | GET | `/api/channel/:id/models` | 获取上游模型列表 |

#### 核心数据结构

```go
// OpenAI 模型结构
type OpenAIModel struct {
    ID         string `json:"id"`
    Object     string `json:"object"`
    Created    int64  `json:"created"`
    OwnedBy    string `json:"owned_by"`
    Permission []struct {
        ID                 string `json:"id"`
        Object             string `json:"object"`
        Created            int64  `json:"created"`
        AllowCreateEngine  bool   `json:"allow_create_engine"`
        AllowSampling      bool   `json:"allow_sampling"`
        AllowLogprobs      bool   `json:"allow_logprobs"`
        AllowSearchIndices bool   `json:"allow_search_indices"`
        AllowView          bool   `json:"allow_view"`
        AllowFineTuning    bool   `json:"allow_fine_tuning"`
        Organization       string `json:"organization"`
        Group              string `json:"group"`
        IsBlocking         bool   `json:"is_blocking"`
    } `json:"permission"`
    Root   string `json:"root"`
    Parent string `json:"parent"`
}

// 添加渠道请求结构
type AddChannelRequest struct {
    Mode                      string                `json:"mode"`
    MultiKeyMode              constant.MultiKeyMode `json:"multi_key_mode"`
    BatchAddSetKeyPrefix2Name bool                  `json:"batch_add_set_key_prefix_2_name"`
    Channel                   *model.Channel        `json:"channel"`
}
```

#### 渠道添加模式

1. **单个模式 (single)**: 添加单个渠道
2. **批量模式 (batch)**: 批量添加多个渠道
3. **多密钥转单个模式 (multi_to_single)**: 将多个密钥合并为一个多密钥渠道

#### 多密钥管理功能

- **密钥状态管理**: 支持启用/禁用单个密钥
- **轮询模式**: 支持随机和轮询两种密钥选择策略
- **批量操作**: 支持批量启用/禁用/删除密钥
- **状态查询**: 支持分页查询密钥状态

### 2. 计费管理 (billing.go)

#### 主要接口

| 接口 | 方法 | 功能描述 |
|------|------|----------|
| GetSubscription | GET | 获取用户订阅信息，兼容 OpenAI API 格式 |
| GetUsage | GET | 获取用户使用量统计 |

#### 响应格式

```go
type OpenAISubscriptionResponse struct {
    Object             string  `json:"object"`
    HasPaymentMethod   bool    `json:"has_payment_method"`
    SoftLimitUSD       float64 `json:"soft_limit_usd"`
    HardLimitUSD       float64 `json:"hard_limit_usd"`
    SystemHardLimitUSD float64 `json:"system_hard_limit_usd"`
    AccessUntil        int64   `json:"access_until"`
}

type OpenAIUsageResponse struct {
    Object     string  `json:"object"`
    TotalUsage float64 `json:"total_usage"`
}
```

### 3. 安全功能

#### 2FA 验证
- **GetChannelKey**: 需要 2FA 验证才能查看渠道密钥
- 支持 TOTP 和备用码两种验证方式
- 统一的验证逻辑和错误处理

#### 渠道验证
- **validateChannel**: 统一的渠道配置验证
- **VertexAI 特殊验证**: 部署地区格式验证
- **模型名称长度验证**: 防止过长的模型名称

## API 设计模式

### 1. 统一响应格式

```go
// 成功响应
{
    "success": true,
    "message": "",
    "data": {...}
}

// 错误响应
{
    "success": false,
    "message": "错误信息"
}
```

### 2. 分页查询

```go
// 分页参数
type PageInfo struct {
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
}

// 分页响应
{
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
}
```

### 3. 筛选和搜索

- **状态筛选**: enabled/disabled/all
- **类型筛选**: 按渠道类型筛选
- **关键词搜索**: 支持 ID、名称、密钥、URL 搜索
- **标签模式**: 支持按标签分组查看

## 错误处理

### 1. 统一错误处理

```go
func common.ApiError(c *gin.Context, err error) {
    c.JSON(http.StatusOK, gin.H{
        "success": false,
        "message": err.Error(),
    })
}
```

### 2. 参数验证

- 使用 Gin 的 `ShouldBindJSON` 进行参数绑定
- 自定义验证函数进行业务逻辑验证
- 详细的错误信息返回

## 中间件集成

### 1. 认证中间件
- Token 验证
- 用户权限检查
- 2FA 验证

### 2. 日志中间件
- 请求日志记录
- 操作审计
- 性能监控

## 最佳实践

### 1. 代码组织
- 按功能模块分文件
- 统一的命名规范
- 清晰的接口定义

### 2. 错误处理
- 统一的错误响应格式
- 详细的错误信息
- 适当的 HTTP 状态码

### 3. 性能优化
- 分页查询避免大量数据加载
- 缓存机制减少数据库查询
- 异步处理耗时操作

### 4. 安全考虑
- 敏感信息脱敏
- 输入参数验证
- 权限控制

## 依赖关系

### 内部依赖
- `model`: 数据模型和数据库操作
- `common`: 公共工具函数
- `constant`: 常量定义
- `dto`: 数据传输对象

### 外部依赖
- `gin-gonic/gin`: HTTP 框架
- `gorm`: ORM 框架

## 扩展指南

### 1. 添加新接口
1. 在对应的文件中添加处理函数
2. 定义请求/响应结构体
3. 实现业务逻辑
4. 添加路由注册

### 2. 添加新功能模块
1. 创建新的 Go 文件
2. 实现相关接口
3. 更新路由配置
4. 添加相应的测试

### 3. 集成新的渠道类型
1. 在 `constant` 中定义新类型
2. 实现渠道特定的验证逻辑
3. 添加模型获取逻辑
4. 更新文档

## 注意事项

1. **并发安全**: 多密钥渠道使用锁机制保证线程安全
2. **事务处理**: 批量操作使用数据库事务保证一致性
3. **缓存更新**: 渠道变更后及时更新缓存
4. **向后兼容**: API 变更需要考虑向后兼容性
5. **性能监控**: 关注接口响应时间和数据库查询性能
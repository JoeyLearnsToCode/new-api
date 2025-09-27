# Middleware 模块文档

## 概述

Middleware 模块是 New API 系统的中间件层，负责处理 HTTP 请求的预处理和后处理逻辑。该模块实现了认证、授权、限流、日志记录、跨域处理等核心功能，为整个系统提供统一的请求处理管道。

## 模块架构

### 核心组件

```
middleware/
├── auth.go                           # 认证和授权中间件
├── distributor.go                    # 请求分发和渠道选择
├── rate-limit.go                     # 限流中间件
├── model-rate-limit.go              # 模型级别限流
├── email-verification-rate-limit.go # 邮箱验证限流
├── cors.go                          # 跨域处理
├── logger.go                        # 日志记录
├── stats.go                         # 统计中间件
├── cache.go                         # 缓存中间件
├── gzip.go                          # 压缩中间件
├── recover.go                       # 异常恢复
├── request-id.go                    # 请求ID生成
└── disable-cache.go                 # 缓存禁用
```

## 核心功能

### 1. 认证授权 (auth.go)

#### 主要功能
- **多种认证方式**: 支持 Session、Access Token、API Key 认证
- **角色权限控制**: 支持普通用户、管理员、超级管理员权限
- **IP 白名单**: 支持令牌级别的 IP 访问限制
- **用户状态检查**: 检查用户是否被禁用

#### 认证流程

```go
func authHelper(c *gin.Context, minRole int) {
    // 1. 检查 Session 认证
    session := sessions.Default(c)
    username := session.Get("username")
    
    if username == nil {
        // 2. 检查 Access Token 认证
        accessToken := c.Request.Header.Get("Authorization")
        user := model.ValidateAccessToken(accessToken)
        // 设置用户信息到上下文
    }
    
    // 3. 验证用户权限和状态
    // 4. 设置上下文信息
}
```

#### 权限级别

```go
// 权限中间件
func UserAuth() func(c *gin.Context)    // 普通用户权限
func AdminAuth() func(c *gin.Context)   // 管理员权限  
func RootAuth() func(c *gin.Context)    // 超级管理员权限
func TryUserAuth() func(c *gin.Context) // 尝试认证（可选）
```

#### Token 认证处理

```go
func TokenAuth() func(c *gin.Context) {
    // 1. WebSocket 协议处理
    if c.Request.Header.Get("Sec-WebSocket-Protocol") != "" {
        // 从 WebSocket 协议头提取 API Key
    }
    
    // 2. Anthropic API 兼容
    if strings.Contains(c.Request.URL.Path, "/v1/messages") {
        anthropicKey := c.Request.Header.Get("x-api-key")
        // 设置为标准 Authorization 头
    }
    
    // 3. Gemini API 兼容
    if strings.HasPrefix(c.Request.URL.Path, "/v1beta/models") {
        // 从 query 参数或 x-goog-api-key 头获取密钥
    }
    
    // 4. 验证令牌和设置上下文
}
```

### 2. 请求分发 (distributor.go)

#### 主要功能
- **智能渠道选择**: 根据模型和用户组选择最优渠道
- **模型权限控制**: 检查令牌的模型访问权限
- **负载均衡**: 在多个可用渠道间分发请求
- **渠道状态检查**: 确保选择的渠道处于可用状态

#### 分发流程

```go
func Distribute() func(c *gin.Context) {
    // 1. 解析请求模型信息
    modelRequest, shouldSelectChannel, err := getModelRequest(c)
    
    // 2. 检查令牌模型权限
    if modelLimitEnable {
        // 验证令牌是否有权访问指定模型
    }
    
    // 3. 选择合适的渠道
    if shouldSelectChannel {
        channel, selectGroup, err = model.CacheGetRandomSatisfiedChannel(
            c, userGroup, modelRequest.Model, 0, nil)
    }
    
    // 4. 设置渠道上下文信息
    SetupContextForSelectedChannel(c, channel, modelRequest.Model)
}
```

#### 请求类型识别

```go
func getModelRequest(c *gin.Context) (*ModelRequest, bool, error) {
    // 1. Midjourney 请求处理
    if strings.Contains(c.Request.URL.Path, "/mj/") {
        // 解析 MJ 请求参数
    }
    
    // 2. Suno 音乐生成请求
    if strings.Contains(c.Request.URL.Path, "/suno/") {
        // 解析 Suno 请求参数
    }
    
    // 3. 视频生成请求
    if strings.Contains(c.Request.URL.Path, "/v1/video/generations") {
        // 解析视频生成请求
    }
    
    // 4. Gemini API 请求
    if strings.HasPrefix(c.Request.URL.Path, "/v1beta/models/") {
        // 从路径提取模型名
    }
    
    // 5. 标准 OpenAI 格式请求
    // 解析 JSON 请求体
}
```

#### 渠道上下文设置

```go
func SetupContextForSelectedChannel(c *gin.Context, channel *model.Channel, modelName string) {
    // 设置渠道基本信息
    common.SetContextKey(c, constant.ContextKeyChannelId, channel.Id)
    common.SetContextKey(c, constant.ContextKeyChannelName, channel.Name)
    common.SetContextKey(c, constant.ContextKeyChannelType, channel.Type)
    
    // 设置渠道配置
    common.SetContextKey(c, constant.ContextKeyChannelSetting, channel.GetSetting())
    common.SetContextKey(c, constant.ContextKeyChannelModelMapping, channel.GetModelMapping())
    
    // 获取并设置 API Key
    key, index, newAPIError := channel.GetNextEnabledKey()
    common.SetContextKey(c, constant.ContextKeyChannelKey, key)
    
    // 设置特定渠道参数
    switch channel.Type {
    case constant.ChannelTypeAzure:
        c.Set("api_version", channel.Other)
    case constant.ChannelTypeGemini:
        c.Set("api_version", channel.Other)
    // ... 其他渠道类型
    }
}
```

### 3. 限流控制 (rate-limit.go)

#### 主要功能
- **多级限流**: 支持全局 Web、全局 API、关键操作等不同级别限流
- **存储后端**: 支持 Redis 和内存两种存储方式
- **灵活配置**: 可配置限流数量和时间窗口
- **IP 级别**: 基于客户端 IP 进行限流

#### 限流实现

```go
// Redis 限流实现
func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
    key := "rateLimit:" + mark + c.ClientIP()
    
    // 1. 检查当前请求数量
    listLength, err := rdb.LLen(ctx, key).Result()
    
    if listLength < int64(maxRequestNum) {
        // 2. 未达到限制，记录请求
        rdb.LPush(ctx, key, time.Now().Format(timeFormat))
        rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
    } else {
        // 3. 检查时间窗口
        oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
        oldTime, _ := time.Parse(timeFormat, oldTimeStr)
        
        if int64(nowTime.Sub(oldTime).Seconds()) < duration {
            // 4. 在限流时间窗口内，拒绝请求
            c.Status(http.StatusTooManyRequests)
            c.Abort()
            return
        } else {
            // 5. 超出时间窗口，允许请求
            rdb.LPush(ctx, key, time.Now().Format(timeFormat))
            rdb.LTrim(ctx, key, 0, int64(maxRequestNum-1))
        }
    }
}

// 内存限流实现
func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
    key := mark + c.ClientIP()
    if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
        c.Status(http.StatusTooManyRequests)
        c.Abort()
        return
    }
}
```

#### 限流类型

```go
// 全局 Web 限流
func GlobalWebRateLimit() func(c *gin.Context)

// 全局 API 限流  
func GlobalAPIRateLimit() func(c *gin.Context)

// 关键操作限流
func CriticalRateLimit() func(c *gin.Context)

// 下载限流
func DownloadRateLimit() func(c *gin.Context)

// 上传限流
func UploadRateLimit() func(c *gin.Context)
```

### 4. 其他核心中间件

#### CORS 跨域处理 (cors.go)
```go
func CORS() gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        // 设置跨域响应头
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "authorization,content-type")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        
        c.Next()
    })
}
```

#### 请求日志 (logger.go)
```go
func RequestLogger() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
            param.ClientIP,
            param.TimeStamp.Format(time.RFC1123),
            param.Method,
            param.Path,
            param.Request.Proto,
            param.StatusCode,
            param.Latency,
            param.Request.UserAgent(),
            param.ErrorMessage,
        )
    })
}
```

#### 异常恢复 (recover.go)
```go
func Recover() gin.HandlerFunc {
    return gin.Recovery()
}
```

#### 统计中间件 (stats.go)
```go
func StatsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        
        // 记录请求统计信息
        duration := time.Since(start)
        // 更新统计数据
    }
}
```

## 中间件链配置

### 标准 API 路由中间件链

```go
relayV1Router := router.Group("/v1")
relayV1Router.Use(middleware.TokenAuth())           // Token 认证
relayV1Router.Use(middleware.ModelRequestRateLimit()) // 模型限流
relayV1Router.Use(middleware.Distribute())          // 请求分发
```

### Web 管理界面中间件链

```go
apiRouter := router.Group("/api")
apiRouter.Use(middleware.CORS())                    // 跨域处理
apiRouter.Use(middleware.GlobalWebRateLimit())      // Web 限流
apiRouter.Use(middleware.UserAuth())                // 用户认证
```

### 任务处理中间件链

```go
relaySunoRouter := router.Group("/suno")
relaySunoRouter.Use(middleware.TokenAuth())         // Token 认证
relaySunoRouter.Use(middleware.Distribute())        // 请求分发
```

## 配置参数

### 限流配置
```go
// 全局限流配置
GlobalWebRateLimitEnable    bool   // 是否启用 Web 限流
GlobalWebRateLimitNum       int    // Web 限流数量
GlobalWebRateLimitDuration  int64  // Web 限流时间窗口

GlobalApiRateLimitEnable    bool   // 是否启用 API 限流
GlobalApiRateLimitNum       int    // API 限流数量  
GlobalApiRateLimitDuration  int64  // API 限流时间窗口

CriticalRateLimitNum        int    // 关键操作限流数量
CriticalRateLimitDuration   int64  // 关键操作限流时间窗口
```

### 认证配置
```go
// Session 配置
SessionName     string  // Session 名称
SessionSecret   string  // Session 密钥
SessionMaxAge   int     // Session 过期时间

// Token 配置  
TokenExpiration int64   // Token 过期时间
```

## 错误处理

### 认证错误
```json
{
    "success": false,
    "message": "无权进行此操作，未登录且未提供 access token"
}
```

### 限流错误
```http
HTTP/1.1 429 Too Many Requests
```

### 权限错误
```json
{
    "success": false,
    "message": "无权进行此操作，权限不足"
}
```

### 渠道选择错误
```json
{
    "error": {
        "code": "model_not_found",
        "message": "分组 default 下模型 gpt-4 无可用渠道",
        "type": "invalid_request_error"
    }
}
```

## 性能优化

### 1. 缓存策略
- **渠道信息缓存**: 缓存渠道配置减少数据库查询
- **用户信息缓存**: 缓存用户权限和配额信息
- **模型映射缓存**: 缓存模型到渠道的映射关系

### 2. 连接池
- **Redis 连接池**: 复用 Redis 连接减少连接开销
- **数据库连接池**: 优化数据库访问性能

### 3. 异步处理
- **日志异步写入**: 避免日志写入阻塞请求处理
- **统计数据异步更新**: 后台更新统计信息

## 监控指标

### 关键指标
- **认证成功率**: 各种认证方式的成功率
- **限流触发率**: 各级别限流的触发频率
- **渠道选择成功率**: 渠道分发的成功率
- **中间件响应时间**: 各中间件的处理时间

### 日志记录
- **认证日志**: 记录认证成功/失败信息
- **限流日志**: 记录限流触发情况
- **错误日志**: 记录中间件处理错误
- **性能日志**: 记录中间件执行时间

## 扩展指南

### 添加新的中间件

1. **创建中间件函数**
```go
func NewMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 预处理逻辑
        
        c.Next() // 调用下一个中间件
        
        // 后处理逻辑
    }
}
```

2. **注册中间件**
```go
router.Use(middleware.NewMiddleware())
```

### 自定义认证方式

1. **实现认证逻辑**
```go
func CustomAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 自定义认证逻辑
        token := c.GetHeader("Custom-Token")
        
        if !validateCustomToken(token) {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "Invalid custom token",
            })
            c.Abort()
            return
        }
        
        // 设置用户信息到上下文
        c.Set("user_id", getUserIdFromToken(token))
        c.Next()
    }
}
```

### 自定义限流策略

1. **实现限流算法**
```go
func CustomRateLimit(limit int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.ClientIP()
        
        if !customLimiter.Allow(key, limit, window) {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

## 最佳实践

### 1. 中间件顺序
- 将认证中间件放在业务逻辑之前
- 将限流中间件放在认证之后
- 将日志和统计中间件放在最外层

### 2. 错误处理
- 提供统一的错误响应格式
- 记录详细的错误日志用于调试
- 区分客户端错误和服务器错误

### 3. 性能考虑
- 避免在中间件中进行耗时操作
- 使用缓存减少重复计算
- 合理设置超时时间

### 4. 安全考虑
- 验证所有输入参数
- 使用安全的 Session 配置
- 实现 IP 白名单功能
- 定期轮换 API 密钥

---

*本文档描述了 Middleware 模块的核心功能和使用方法。如需了解具体的实现细节，请参考源代码和相关的技术文档。*
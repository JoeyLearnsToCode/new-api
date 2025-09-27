# Router 模块文档

## 概述

Router 模块是 New API 系统的路由管理组件，负责定义和组织所有 HTTP 路由规则。该模块采用分层路由架构，将不同功能的路由分组管理，并为每个路由配置相应的中间件和处理器。

## 模块架构

### 核心组件

```
router/
├── main.go           # 主路由配置和入口
├── api-router.go     # API 管理路由
├── relay-router.go   # 请求转发路由
├── web-router.go     # Web 界面路由
├── dashboard.go      # 仪表板路由
└── video-router.go   # 视频处理路由
```

## 核心功能

### 1. 主路由配置 (main.go)

#### 路由初始化

```go
func SetRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
    // 1. 设置 API 管理路由
    SetApiRouter(router)
    
    // 2. 设置仪表板路由
    SetDashboardRouter(router)
    
    // 3. 设置请求转发路由
    SetRelayRouter(router)
    
    // 4. 设置视频处理路由
    SetVideoRouter(router)
    
    // 5. 设置前端路由或重定向
    frontendBaseUrl := os.Getenv("FRONTEND_BASE_URL")
    if frontendBaseUrl == "" {
        SetWebRouter(router, buildFS, indexPage)
    } else {
        // 重定向到外部前端
        router.NoRoute(func(c *gin.Context) {
            c.Redirect(http.StatusMovedPermanently, 
                fmt.Sprintf("%s%s", frontendBaseUrl, c.Request.RequestURI))
        })
    }
}
```

#### 架构特点
- **模块化设计**: 不同功能的路由分别管理
- **中间件集成**: 每个路由组配置相应的中间件
- **灵活部署**: 支持内嵌前端和外部前端两种部署方式

### 2. 请求转发路由 (relay-router.go)

#### 主要功能
- **多协议支持**: HTTP 和 WebSocket 协议
- **多格式兼容**: OpenAI、Claude、Gemini 等 API 格式
- **任务处理**: 支持异步任务提交和查询
- **模型管理**: 模型列表和详情查询

#### 路由结构

```go
func SetRelayRouter(router *gin.Engine) {
    // 基础中间件
    router.Use(middleware.CORS())
    router.Use(middleware.DecompressRequestMiddleware())
    router.Use(middleware.StatsMiddleware())
    
    // 1. 模型相关路由
    setupModelsRouter(router)
    
    // 2. 标准 API 路由
    setupRelayV1Router(router)
    
    // 3. 任务处理路由
    setupTaskRouters(router)
    
    // 4. 特殊格式路由
    setupSpecialFormatRouters(router)
}
```

#### 模型路由配置

```go
// OpenAI 兼容模型路由
modelsRouter := router.Group("/v1/models")
modelsRouter.Use(middleware.TokenAuth())
{
    modelsRouter.GET("", func(c *gin.Context) {
        switch {
        case c.GetHeader("x-api-key") != "" && c.GetHeader("anthropic-version") != "":
            controller.ListModels(c, constant.ChannelTypeAnthropic)
        case c.GetHeader("x-goog-api-key") != "" || c.Query("key") != "":
            controller.RetrieveModel(c, constant.ChannelTypeGemini)
        default:
            controller.ListModels(c, constant.ChannelTypeOpenAI)
        }
    })
    
    modelsRouter.GET("/:model", func(c *gin.Context) {
        // 获取特定模型信息
    })
}

// Gemini 原生模型路由
geminiRouter := router.Group("/v1beta/models")
geminiRouter.Use(middleware.TokenAuth())
{
    geminiRouter.GET("", func(c *gin.Context) {
        controller.ListModels(c, constant.ChannelTypeGemini)
    })
}
```

#### 标准 API 路由

```go
relayV1Router := router.Group("/v1")
relayV1Router.Use(middleware.TokenAuth())
relayV1Router.Use(middleware.ModelRequestRateLimit())

// WebSocket 路由
wsRouter := relayV1Router.Group("")
wsRouter.Use(middleware.Distribute())
wsRouter.GET("/realtime", func(c *gin.Context) {
    controller.Relay(c, types.RelayFormatOpenAIRealtime)
})

// HTTP 路由
httpRouter := relayV1Router.Group("")
httpRouter.Use(middleware.Distribute())
{
    // 文本生成
    httpRouter.POST("/chat/completions", func(c *gin.Context) {
        controller.Relay(c, types.RelayFormatOpenAI)
    })
    
    // Claude 格式
    httpRouter.POST("/messages", func(c *gin.Context) {
        controller.Relay(c, types.RelayFormatClaude)
    })
    
    // 嵌入向量
    httpRouter.POST("/embeddings", func(c *gin.Context) {
        controller.Relay(c, types.RelayFormatEmbedding)
    })
    
    // 图像生成
    httpRouter.POST("/images/generations", func(c *gin.Context) {
        controller.Relay(c, types.RelayFormatOpenAIImage)
    })
    
    // 音频处理
    httpRouter.POST("/audio/transcriptions", func(c *gin.Context) {
        controller.Relay(c, types.RelayFormatOpenAIAudio)
    })
    
    // 其他路由...
}
```

#### 任务处理路由

```go
// Midjourney 任务路由
relayMjRouter := router.Group("/mj")
registerMjRouterGroup(relayMjRouter)

func registerMjRouterGroup(relayMjRouter *gin.RouterGroup) {
    // 图像获取（无需认证）
    relayMjRouter.GET("/image/:id", relay.RelayMidjourneyImage)
    
    // 任务操作（需要认证）
    relayMjRouter.Use(middleware.TokenAuth(), middleware.Distribute())
    {
        relayMjRouter.POST("/submit/imagine", controller.RelayMidjourney)
        relayMjRouter.POST("/submit/change", controller.RelayMidjourney)
        relayMjRouter.POST("/submit/describe", controller.RelayMidjourney)
        relayMjRouter.GET("/task/:id/fetch", controller.RelayMidjourney)
        // 更多 MJ 操作...
    }
}

// Suno 音乐生成路由
relaySunoRouter := router.Group("/suno")
relaySunoRouter.Use(middleware.TokenAuth(), middleware.Distribute())
{
    relaySunoRouter.POST("/submit/:action", controller.RelayTask)
    relaySunoRouter.POST("/fetch", controller.RelayTask)
    relaySunoRouter.GET("/fetch/:id", controller.RelayTask)
}

// Gemini 原生路由
relayGeminiRouter := router.Group("/v1beta")
relayGeminiRouter.Use(middleware.TokenAuth())
relayGeminiRouter.Use(middleware.ModelRequestRateLimit())
relayGeminiRouter.Use(middleware.Distribute())
{
    relayGeminiRouter.POST("/models/*path", func(c *gin.Context) {
        controller.Relay(c, types.RelayFormatGemini)
    })
}
```

### 3. API 管理路由 (api-router.go)

#### 主要功能
- **用户管理**: 用户注册、登录、信息管理
- **令牌管理**: API 令牌的创建、更新、删除
- **渠道管理**: AI 服务渠道的配置和管理
- **系统设置**: 系统配置和参数管理
- **统计报表**: 使用统计和数据分析

#### 路由结构示例

```go
func SetApiRouter(router *gin.Engine) {
    apiRouter := router.Group("/api")
    apiRouter.Use(middleware.CORS())
    apiRouter.Use(middleware.GlobalWebRateLimit())
    
    // 公开路由（无需认证）
    setupPublicRoutes(apiRouter)
    
    // 用户路由（需要用户认证）
    setupUserRoutes(apiRouter)
    
    // 管理员路由（需要管理员权限）
    setupAdminRoutes(apiRouter)
    
    // 超级管理员路由（需要超级管理员权限）
    setupRootRoutes(apiRouter)
}

// 用户路由示例
func setupUserRoutes(router *gin.RouterGroup) {
    userRouter := router.Group("/user")
    userRouter.Use(middleware.UserAuth())
    {
        userRouter.GET("/self", controller.GetSelf)
        userRouter.PUT("/self", controller.UpdateSelf)
        userRouter.POST("/token", controller.GenerateToken)
        userRouter.GET("/token", controller.GetTokens)
        userRouter.DELETE("/token/:id", controller.DeleteToken)
    }
}

// 管理员路由示例
func setupAdminRoutes(router *gin.RouterGroup) {
    adminRouter := router.Group("/admin")
    adminRouter.Use(middleware.AdminAuth())
    {
        adminRouter.GET("/user", controller.GetAllUsers)
        adminRouter.POST("/user", controller.CreateUser)
        adminRouter.PUT("/user/:id", controller.UpdateUser)
        adminRouter.DELETE("/user/:id", controller.DeleteUser)
        
        adminRouter.GET("/channel", controller.GetAllChannels)
        adminRouter.POST("/channel", controller.CreateChannel)
        adminRouter.PUT("/channel/:id", controller.UpdateChannel)
        adminRouter.DELETE("/channel/:id", controller.DeleteChannel)
    }
}
```

### 4. Web 界面路由 (web-router.go)

#### 主要功能
- **静态资源服务**: 前端静态文件服务
- **单页应用支持**: SPA 路由回退处理
- **资源压缩**: 静态资源的 Gzip 压缩
- **缓存控制**: 静态资源的缓存策略

#### 实现示例

```go
func SetWebRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
    // 静态资源路由
    router.StaticFS("/static", http.FS(buildFS))
    
    // 根路径和 SPA 回退
    router.NoRoute(func(c *gin.Context) {
        // 检查是否为 API 路径
        if strings.HasPrefix(c.Request.URL.Path, "/api/") ||
           strings.HasPrefix(c.Request.URL.Path, "/v1/") {
            c.JSON(http.StatusNotFound, gin.H{
                "error": "API endpoint not found",
            })
            return
        }
        
        // 返回前端页面
        c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
    })
}
```

### 5. 仪表板路由 (dashboard.go)

#### 主要功能
- **实时监控**: 系统状态和性能监控
- **数据统计**: 使用量和费用统计
- **健康检查**: 系统健康状态检查

### 6. 视频处理路由 (video-router.go)

#### 主要功能
- **视频生成**: 视频生成任务提交
- **任务查询**: 视频生成任务状态查询
- **结果获取**: 生成结果的下载和预览

## 路由组织原则

### 1. 按功能分组
- **API 管理**: `/api/*` - 系统管理功能
- **请求转发**: `/v1/*` - AI API 转发
- **任务处理**: `/mj/*`, `/suno/*` - 异步任务
- **Web 界面**: `/*` - 前端页面

### 2. 中间件分层
```go
// 全局中间件
router.Use(middleware.CORS())
router.Use(middleware.Logger())
router.Use(middleware.Recovery())

// 组级中间件
apiGroup.Use(middleware.UserAuth())
relayGroup.Use(middleware.TokenAuth())

// 路由级中间件
router.GET("/admin/*", middleware.AdminAuth(), handler)
```

### 3. 权限控制
- **公开路由**: 无需认证
- **用户路由**: 需要用户登录
- **管理员路由**: 需要管理员权限
- **超级管理员路由**: 需要超级管理员权限

## 路由配置

### 中间件配置

```go
// 标准 API 路由中间件链
relayRouter.Use(middleware.TokenAuth())           // Token 认证
relayRouter.Use(middleware.ModelRequestRateLimit()) // 模型限流
relayRouter.Use(middleware.Distribute())          // 请求分发

// Web 管理路由中间件链
apiRouter.Use(middleware.CORS())                  // 跨域处理
apiRouter.Use(middleware.GlobalWebRateLimit())    // Web 限流
apiRouter.Use(middleware.UserAuth())              // 用户认证
```

### 路由参数

```go
// 路径参数
router.GET("/user/:id", handler)           // /user/123
router.GET("/task/:id/fetch", handler)     // /task/abc/fetch

// 查询参数
router.GET("/models", handler)             // /models?key=value

// 通配符路径
router.POST("/models/*path", handler)      // /models/gpt-4:generateContent
```

## 错误处理

### 404 处理
```go
router.NoRoute(func(c *gin.Context) {
    if strings.HasPrefix(c.Request.URL.Path, "/api/") {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "message": "API 接口不存在",
        })
    } else {
        // 返回前端页面（SPA 路由）
        c.Data(http.StatusOK, "text/html", indexPage)
    }
})
```

### 方法不允许处理
```go
router.NoMethod(func(c *gin.Context) {
    c.JSON(http.StatusMethodNotAllowed, gin.H{
        "success": false,
        "message": "HTTP 方法不被允许",
    })
})
```

## 性能优化

### 1. 路由优化
- **路由顺序**: 将常用路由放在前面
- **路径匹配**: 使用精确匹配而非通配符
- **中间件优化**: 避免不必要的中间件执行

### 2. 静态资源优化
- **资源压缩**: 启用 Gzip 压缩
- **缓存控制**: 设置合适的缓存头
- **CDN 集成**: 使用 CDN 加速静态资源

### 3. 并发处理
- **连接池**: 合理配置连接池大小
- **超时设置**: 设置合适的请求超时时间
- **限流控制**: 防止系统过载

## 监控和日志

### 关键指标
- **路由响应时间**: 各路由的平均响应时间
- **请求成功率**: 各路由的成功率统计
- **并发连接数**: 当前活跃连接数
- **错误率**: 4xx 和 5xx 错误的比例

### 日志记录
- **访问日志**: 记录所有 HTTP 请求
- **错误日志**: 记录路由处理错误
- **性能日志**: 记录慢请求和性能问题

## 扩展指南

### 添加新的路由组

1. **创建路由文件**
```go
// router/new-router.go
func SetNewRouter(router *gin.Engine) {
    newRouter := router.Group("/new")
    newRouter.Use(middleware.TokenAuth())
    {
        newRouter.GET("/endpoint", controller.NewHandler)
        newRouter.POST("/endpoint", controller.CreateHandler)
    }
}
```

2. **注册路由组**
```go
// router/main.go
func SetRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
    SetApiRouter(router)
    SetRelayRouter(router)
    SetNewRouter(router)  // 添加新路由组
    // ...
}
```

### 添加新的中间件

1. **应用到路由组**
```go
apiRouter.Use(middleware.NewMiddleware())
```

2. **应用到特定路由**
```go
router.GET("/special", middleware.SpecialAuth(), handler)
```

### 自定义错误处理

```go
router.Use(func(c *gin.Context) {
    c.Next()
    
    // 处理响应后的错误
    if len(c.Errors) > 0 {
        err := c.Errors.Last()
        // 自定义错误响应
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
    }
})
```

## 最佳实践

### 1. 路由设计
- 使用 RESTful 风格的 URL 设计
- 保持路由结构的一致性
- 合理使用路径参数和查询参数
- 避免过深的路由嵌套

### 2. 中间件使用
- 将通用中间件应用到全局
- 将特定中间件应用到相应的路由组
- 注意中间件的执行顺序
- 避免重复的中间件逻辑

### 3. 错误处理
- 提供统一的错误响应格式
- 区分不同类型的错误
- 记录详细的错误日志
- 提供友好的错误信息

### 4. 安全考虑
- 验证所有输入参数
- 使用 HTTPS 传输敏感数据
- 实现适当的认证和授权
- 防止常见的 Web 攻击

---

*本文档描述了 Router 模块的核心功能和使用方法。如需了解具体的实现细节，请参考源代码和相关的技术文档。*
# Logger 模块 - 日志记录功能

## 模块概述

Logger 模块是 New API 系统的统一日志管理模块，提供了结构化的日志记录功能，支持多级别日志输出、文件轮转、上下文追踪等特性。该模块基于 Gin 框架的日志系统进行扩展，为整个系统提供了标准化的日志记录服务。

## 核心功能

### 1. 多级别日志记录
- INFO：信息级别日志
- WARN：警告级别日志
- ERROR：错误级别日志
- DEBUG：调试级别日志

### 2. 上下文追踪
- 请求 ID 追踪
- 系统级日志标识
- 时间戳记录

### 3. 文件管理
- 自动日志文件创建
- 日志轮转机制
- 并发安全的文件操作

### 4. 配额格式化
- 货币格式显示
- 点数格式显示
- 灵活的配额展示

## 文件结构

```
logger/
└── logger.go     # 日志记录核心功能实现
```

## 详细功能说明

### 日志级别定义

```go
const (
    loggerINFO  = "INFO"   // 信息级别
    loggerWarn  = "WARN"   // 警告级别
    loggerError = "ERR"    // 错误级别
    loggerDebug = "DEBUG"  // 调试级别
)
```

### 日志轮转配置

```go
const maxLogCount = 1000000  // 最大日志条数，超过后自动轮转

var logCount int             // 当前日志计数
var setupLogLock sync.Mutex  // 日志设置锁
var setupLogWorking bool     // 日志设置工作状态
```

### 核心日志记录函数

#### 1. 信息级别日志
```go
func LogInfo(ctx context.Context, msg string)
```
- **功能**: 记录信息级别的日志
- **参数**: 
  - `ctx`: 上下文对象，用于获取请求 ID
  - `msg`: 日志消息内容
- **输出**: 标准输出和日志文件

#### 2. 警告级别日志
```go
func LogWarn(ctx context.Context, msg string)
```
- **功能**: 记录警告级别的日志
- **参数**: 同 LogInfo
- **输出**: 标准错误输出和日志文件

#### 3. 错误级别日志
```go
func LogError(ctx context.Context, msg string)
```
- **功能**: 记录错误级别的日志
- **参数**: 同 LogInfo
- **输出**: 标准错误输出和日志文件

#### 4. 调试级别日志
```go
func LogDebug(ctx context.Context, msg string)
```
- **功能**: 记录调试级别的日志（仅在调试模式下输出）
- **参数**: 同 LogInfo
- **条件**: 需要 `common.DebugEnabled = true`

### 日志设置和管理

#### 日志系统初始化
```go
func SetupLogger()
```
- **功能**: 初始化日志系统，设置日志文件输出
- **特性**:
  - 支持并发安全的初始化
  - 自动创建带时间戳的日志文件
  - 同时输出到控制台和文件
  - 支持日志目录配置

**日志文件命名规则**:
```
oneapi-{YYYYMMDDHHMMSS}.log
例如: oneapi-20240315142530.log
```

#### 自动日志轮转
系统在日志条数超过 `maxLogCount` (1,000,000) 时自动触发日志轮转：
1. 重置日志计数器
2. 异步创建新的日志文件
3. 切换日志输出到新文件

### 配额格式化功能

#### 配额日志格式化
```go
func LogQuota(quota int) string
```
- **功能**: 将配额数值格式化为可读的字符串
- **参数**: `quota` - 配额数值
- **返回**: 格式化后的配额字符串

**格式化规则**:
```go
// 启用货币显示时
if common.DisplayInCurrencyEnabled {
    return fmt.Sprintf("＄%.6f 额度", float64(quota)/common.QuotaPerUnit)
}
// 使用点数显示时
else {
    return fmt.Sprintf("%d 点额度", quota)
}
```

#### 纯配额格式化
```go
func FormatQuota(quota int) string
```
- **功能**: 格式化配额数值（不带"额度"后缀）
- **参数**: `quota` - 配额数值
- **返回**: 纯配额数值字符串

### 测试和调试功能

#### JSON 对象日志
```go
func LogJson(ctx context.Context, msg string, obj any)
```
- **功能**: 将对象序列化为 JSON 并记录日志（仅供测试使用）
- **参数**:
  - `ctx`: 上下文对象
  - `msg`: 日志消息前缀
  - `obj`: 要序列化的对象
- **输出**: `{msg} | {json_string}`

## 使用示例

### 基本日志记录
```go
import (
    "context"
    "one-api/logger"
    "one-api/common"
)

func ExampleUsage() {
    ctx := context.Background()
    
    // 记录信息日志
    logger.LogInfo(ctx, "用户登录成功")
    
    // 记录警告日志
    logger.LogWarn(ctx, "API 调用频率过高")
    
    // 记录错误日志
    logger.LogError(ctx, "数据库连接失败")
    
    // 记录调试日志（需要开启调试模式）
    common.DebugEnabled = true
    logger.LogDebug(ctx, "调试信息")
}
```

### 带请求 ID 的日志记录
```go
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    // 设置请求 ID 到上下文
    ctx := context.WithValue(r.Context(), common.RequestIdKey, "req-12345")
    
    // 日志将自动包含请求 ID
    logger.LogInfo(ctx, "开始处理请求")
    
    // 处理业务逻辑...
    
    logger.LogInfo(ctx, "请求处理完成")
}
```

### 配额相关日志
```go
func LogQuotaUsage(ctx context.Context, userID int, quotaUsed int) {
    quotaStr := logger.LogQuota(quotaUsed)
    logger.LogInfo(ctx, fmt.Sprintf("用户 %d 消费了 %s", userID, quotaStr))
    
    // 输出示例：
    // [INFO] 2024/03/15 - 14:25:30 | req-12345 | 用户 123 消费了 ＄0.002000 额度
}
```

### JSON 对象日志（测试用）
```go
func DebugAPIResponse(ctx context.Context, response *APIResponse) {
    logger.LogJson(ctx, "API 响应", response)
    
    // 输出示例：
    // [INFO] 2024/03/15 - 14:25:30 | req-12345 | API 响应 | {"code":200,"message":"success","data":{...}}
}
```

### 日志系统初始化
```go
import (
    "flag"
    "one-api/common"
    "one-api/logger"
)

func main() {
    // 设置日志目录
    logDir := flag.String("log-dir", "./logs", "日志文件目录")
    flag.Parse()
    
    common.LogDir = logDir
    
    // 初始化日志系统
    logger.SetupLogger()
    
    // 现在可以使用日志功能了
    ctx := context.Background()
    logger.LogInfo(ctx, "系统启动成功")
}
```

## 日志输出格式

### 标准日志格式
```
[{LEVEL}] {YYYY/MM/DD - HH:MM:SS} | {REQUEST_ID} | {MESSAGE}
```

### 输出示例
```
[INFO] 2024/03/15 - 14:25:30 | req-12345 | 用户登录成功
[WARN] 2024/03/15 - 14:25:31 | req-12346 | API 调用频率过高
[ERR] 2024/03/15 - 14:25:32 | SYSTEM | 数据库连接失败
[DEBUG] 2024/03/15 - 14:25:33 | req-12347 | 调试信息
```

### 字段说明
- **LEVEL**: 日志级别（INFO、WARN、ERR、DEBUG）
- **时间戳**: 精确到秒的时间戳
- **REQUEST_ID**: 请求标识符，系统级日志显示为 "SYSTEM"
- **MESSAGE**: 具体的日志消息内容

## 配置选项

### 环境变量配置
```bash
# 日志目录配置
LOG_DIR=/var/log/oneapi

# 调试模式开关
DEBUG=true
```

### 代码配置
```go
// 启用调试日志
common.DebugEnabled = true

// 设置日志目录
common.LogDir = &logDir

// 配置配额显示模式
common.DisplayInCurrencyEnabled = true
common.QuotaPerUnit = 500 * 1000.0  // $0.002 / 1K tokens
```

## 性能特性

### 1. 异步日志轮转
- 日志轮转操作在后台异步执行
- 不阻塞主业务流程
- 使用协程池管理轮转任务

### 2. 并发安全
- 使用互斥锁保护日志设置操作
- 支持多协程并发写入日志
- 防止重复初始化

### 3. 内存友好
- 日志计数器自动重置
- 及时释放旧日志文件句柄
- 避免内存泄漏

## 最佳实践

### 1. 上下文传递
```go
// 好的做法：始终传递上下文
func BusinessLogic(ctx context.Context) {
    logger.LogInfo(ctx, "业务逻辑开始")
    // ...
}

// 避免：不传递上下文
func BusinessLogic() {
    logger.LogInfo(context.Background(), "业务逻辑开始")
}
```

### 2. 日志级别选择
```go
// 使用合适的日志级别
logger.LogInfo(ctx, "正常业务流程信息")      // 一般信息
logger.LogWarn(ctx, "可能的问题或异常情况")    // 警告
logger.LogError(ctx, "系统错误或异常")       // 错误
logger.LogDebug(ctx, "详细的调试信息")       // 调试（仅开发环境）
```

### 3. 日志消息格式
```go
// 好的做法：清晰描述性的消息
logger.LogInfo(ctx, fmt.Sprintf("用户 %d 登录成功，IP: %s", userID, ip))

// 避免：模糊不清的消息
logger.LogInfo(ctx, "success")
```

### 4. 敏感信息保护
```go
// 好的做法：不记录敏感信息
logger.LogInfo(ctx, fmt.Sprintf("用户认证成功，用户ID: %d", userID))

// 避免：记录密码等敏感信息
logger.LogInfo(ctx, fmt.Sprintf("用户认证：%s/%s", username, password))
```

## 监控和运维

### 1. 日志文件管理
- 定期清理旧日志文件
- 监控日志文件大小和数量
- 配置日志轮转策略

### 2. 日志分析
- 使用日志分析工具（如 ELK Stack）
- 设置关键错误告警
- 监控系统性能指标

### 3. 存储规划
- 根据日志量规划存储空间
- 考虑日志压缩和归档
- 设置合理的保留期限

## 故障排查

### 1. 日志文件无法创建
```go
// 检查日志目录权限
if *common.LogDir != "" {
    if _, err := os.Stat(*common.LogDir); os.IsNotExist(err) {
        os.MkdirAll(*common.LogDir, 0755)
    }
}
```

### 2. 日志轮转失败
- 检查磁盘空间是否充足
- 确认日志目录写权限
- 查看系统错误日志

### 3. 调试日志不输出
```go
// 确保启用调试模式
common.DebugEnabled = true
```

## 扩展开发

### 添加新的日志级别
1. 在常量定义中添加新级别
2. 实现对应的日志函数
3. 更新日志输出逻辑

### 集成外部日志系统
1. 实现日志适配器接口
2. 保持现有 API 兼容性
3. 支持配置切换

### 结构化日志支持
1. 定义日志结构体
2. 实现 JSON 格式输出
3. 支持字段索引和搜索

## 注意事项

1. **性能影响**: 频繁的日志输出可能影响系统性能，合理控制日志级别
2. **磁盘空间**: 日志文件会占用磁盘空间，需要定期清理
3. **并发安全**: 多协程环境下的日志操作已经过并发安全处理
4. **敏感信息**: 避免在日志中记录密码、密钥等敏感信息
5. **日志轮转**: 系统会自动进行日志轮转，无需手动干预

## 依赖关系

Logger 模块依赖以下模块：
- Common 模块：获取配置参数和常量定义
- Gin 框架：使用 Gin 的日志输出功能

被以下模块依赖：
- Controller 模块：记录请求处理日志
- Model 模块：记录数据库操作日志
- Middleware 模块：记录中间件执行日志
- Provider 模块：记录 API 调用日志

该模块为整个系统提供统一的日志记录服务，是系统监控、调试和运维的重要基础设施。
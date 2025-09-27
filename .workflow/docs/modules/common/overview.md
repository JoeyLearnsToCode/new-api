# Common 模块 - 公共工具和配置

## 模块概述

Common 模块是 New API 系统的基础支撑模块，提供了系统运行所需的公共工具、配置管理、加密功能、数据库配置等基础设施。该模块被系统中的其他模块广泛依赖，是整个系统的基石。

## 核心功能

### 1. 系统配置与常量管理
- 全局系统配置变量
- 运行时参数管理
- 功能开关控制
- 用户权限等级定义

### 2. 加密与安全
- 密码哈希处理
- HMAC 签名生成
- 安全密钥管理

### 3. 数据库配置
- 多数据库类型支持
- 数据库连接配置
- SQL 类型管理

### 4. API 类型定义
- API 接口类型枚举
- 请求响应结构定义
- 端点类型配置

## 文件结构

```
common/
├── constants.go           # 系统常量和全局配置
├── crypto.go             # 加密相关工具函数
├── database.go           # 数据库配置
├── api_type.go           # API 类型定义
├── endpoint_type.go      # 端点类型配置
├── endpoint_defaults.go  # 端点默认配置
├── env.go               # 环境变量处理
├── gin.go               # Gin 框架增强
├── email.go             # 邮件发送功能
├── email-outlook-auth.go # Outlook 邮件认证
├── copy.go              # 对象复制工具
├── custom-event.go      # 自定义事件处理
├── embed-file-system.go # 嵌入式文件系统
├── go-channel.go        # Go 通道工具
└── gopool.go            # 协程池管理
```

## 详细功能说明

### 系统配置管理 (constants.go)

#### 核心系统变量
```go
var StartTime = time.Now().Unix()    // 系统启动时间
var Version = "v0.0.0"               // 系统版本号
var SystemName = "New API"           // 系统名称
var Footer = ""                      // 页面底部信息
var Logo = ""                        // 系统 Logo
var TopUpLink = ""                   // 充值链接
```

#### 配额和计费配置
```go
var QuotaPerUnit = 500 * 1000.0           // 每单位配额价格 ($0.002 / 1K tokens)
var DisplayInCurrencyEnabled = true       // 是否以货币形式显示
var DisplayTokenStatEnabled = true        // 是否显示 Token 统计
var QuotaForNewUser = 0                   // 新用户初始配额
var QuotaForInviter = 0                   // 邀请者奖励配额
var QuotaForInvitee = 0                   // 被邀请者奖励配额
var QuotaRemindThreshold = 1000           // 配额提醒阈值
var PreConsumedQuota = 500                // 预消费配额
```

#### 功能开关配置
```go
var DrawingEnabled = true                 // 绘图功能开关
var TaskEnabled = true                    // 任务功能开关
var DataExportEnabled = true              // 数据导出功能开关
var DataExportInterval = 5                // 数据导出间隔（分钟）
var DataExportDefaultTime = "hour"        // 默认导出时间单位
var DefaultCollapseSidebar = false        // 默认折叠侧边栏
```

#### 用户认证配置
```go
var PasswordLoginEnabled = true           // 密码登录开关
var PasswordRegisterEnabled = true        // 密码注册开关
var EmailVerificationEnabled = false      // 邮箱验证开关
var GitHubOAuthEnabled = false           // GitHub OAuth 开关
var LinuxDOOAuthEnabled = false          // LinuxDO OAuth 开关
var WeChatAuthEnabled = false            // 微信认证开关
var TelegramOAuthEnabled = false         // Telegram OAuth 开关
var TurnstileCheckEnabled = false        // Turnstile 验证开关
var RegisterEnabled = true               // 注册功能开关
```

#### 邮箱域名限制配置
```go
var EmailDomainRestrictionEnabled = false  // 邮箱域名限制开关
var EmailAliasRestrictionEnabled = false   // 邮箱别名限制开关
var EmailDomainWhitelist = []string{        // 邮箱域名白名单
    "gmail.com", "163.com", "126.com", "qq.com",
    "outlook.com", "hotmail.com", "icloud.com",
    "yahoo.com", "foxmail.com",
}
```

#### 用户角色定义
```go
const (
    RoleGuestUser  = 0    // 游客用户
    RoleCommonUser = 1    // 普通用户
    RoleAdminUser  = 10   // 管理员用户
    RoleRootUser   = 100  // 超级管理员
)

// 角色验证函数
func IsValidateRole(role int) bool
```

#### 状态常量定义
```go
// 用户状态
const (
    UserStatusEnabled  = 1  // 用户启用
    UserStatusDisabled = 2  // 用户禁用
)

// Token 状态
const (
    TokenStatusEnabled   = 1  // Token 启用
    TokenStatusDisabled  = 2  // Token 禁用
    TokenStatusExpired   = 3  // Token 过期
    TokenStatusExhausted = 4  // Token 耗尽
)

// 兑换码状态
const (
    RedemptionCodeStatusEnabled  = 1  // 兑换码启用
    RedemptionCodeStatusDisabled = 2  // 兑换码禁用
    RedemptionCodeStatusUsed     = 3  // 兑换码已使用
)

// 通道状态
const (
    ChannelStatusUnknown          = 0    // 未知状态
    ChannelStatusEnabled          = 1    // 通道启用
    ChannelStatusManuallyDisabled = 2    // 手动禁用
    ChannelStatusAutoDisabled     = 3    // 自动禁用
    ChannelStatusExpiredDisabled  = 104  // 过期禁用
)
```

#### 速率限制配置
```go
// 全局 API 速率限制
var GlobalApiRateLimitEnable bool
var GlobalApiRateLimitNum int
var GlobalApiRateLimitDuration int64

// 全局 Web 速率限制
var GlobalWebRateLimitEnable bool
var GlobalWebRateLimitNum int
var GlobalWebRateLimitDuration int64

// 上传下载速率限制
var UploadRateLimitNum = 10
var UploadRateLimitDuration int64 = 60
var DownloadRateLimitNum = 10
var DownloadRateLimitDuration int64 = 60

// 关键操作速率限制
var CriticalRateLimitNum = 20
var CriticalRateLimitDuration int64 = 20 * 60
```

### 加密工具 (crypto.go)

#### HMAC 签名生成
```go
// 使用指定密钥生成 HMAC
func GenerateHMACWithKey(key []byte, data string) string

// 使用系统密钥生成 HMAC
func GenerateHMAC(data string) string
```

#### 密码处理
```go
// 密码转哈希
func Password2Hash(password string) (string, error)

// 验证密码和哈希
func ValidatePasswordAndHash(password string, hash string) bool
```

**使用示例：**
```go
// 密码哈希
hashedPassword, err := common.Password2Hash("userpassword")
if err != nil {
    // 处理错误
}

// 密码验证
isValid := common.ValidatePasswordAndHash("userpassword", hashedPassword)

// HMAC 签名
signature := common.GenerateHMAC("data to sign")
```

### 数据库配置 (database.go)

#### 数据库类型常量
```go
const (
    DatabaseTypeMySQL      = "mysql"
    DatabaseTypeSQLite     = "sqlite"
    DatabaseTypePostgreSQL = "postgres"
)
```

#### 数据库状态变量
```go
var UsingSQLite = false        // 是否使用 SQLite
var UsingPostgreSQL = false    // 是否使用 PostgreSQL
var UsingMySQL = false         // 是否使用 MySQL
var UsingClickHouse = false    // 是否使用 ClickHouse
var LogSqlType = DatabaseTypeSQLite  // 日志 SQL 类型
var SQLitePath = "one-api.db?_busy_timeout=30000"  // SQLite 路径
```

## API 接口

### 配置管理接口
- **获取系统配置**: 通过 `OptionMap` 获取系统配置项
- **更新系统配置**: 使用 `OptionMapRWMutex` 保证并发安全
- **角色验证**: `IsValidateRole(role int) bool` 验证用户角色

### 加密接口
- **密码哈希**: `Password2Hash(password string) (string, error)`
- **密码验证**: `ValidatePasswordAndHash(password, hash string) bool`
- **HMAC 签名**: `GenerateHMAC(data string) string`

## 配置示例

### 系统基础配置
```go
// 设置系统名称和版本
common.SystemName = "My API System"
common.Version = "v1.0.0"

// 启用功能
common.DrawingEnabled = true
common.TaskEnabled = true
common.DataExportEnabled = true

// 配置配额
common.QuotaPerUnit = 1000 * 1000.0
common.DisplayInCurrencyEnabled = true
```

### 认证配置
```go
// 启用多种认证方式
common.PasswordLoginEnabled = true
common.GitHubOAuthEnabled = true
common.EmailVerificationEnabled = true

// 配置 OAuth
common.GitHubClientId = "your_github_client_id"
common.GitHubClientSecret = "your_github_client_secret"
```

### 邮件配置
```go
// SMTP 配置
common.SMTPServer = "smtp.gmail.com"
common.SMTPPort = 587
common.SMTPSSLEnabled = true
common.SMTPAccount = "your_email@gmail.com"
common.SMTPToken = "your_app_password"
```

## 最佳实践

### 1. 配置管理
- 使用环境变量覆盖默认配置
- 通过 OptionMap 统一管理配置项
- 使用读写锁保证并发安全

### 2. 安全实践
- 定期更新 SessionSecret 和 CryptoSecret
- 使用强密码策略
- 启用适当的速率限制

### 3. 数据库配置
- 根据部署环境选择合适的数据库类型
- 配置合适的连接池参数
- 启用日志记录用于问题排查

### 4. 监控和日志
- 启用 LogConsumeEnabled 记录消费日志
- 配置合适的数据导出间隔
- 监控系统关键指标

## 注意事项

1. **密钥安全**: SessionSecret 和 CryptoSecret 在系统启动时自动生成，生产环境应使用固定密钥
2. **配额管理**: QuotaPerUnit 直接影响计费，修改时需谨慎
3. **功能开关**: 功能开关的变更需要重启系统才能生效
4. **速率限制**: 速率限制配置过严可能影响正常使用，过松可能导致系统过载
5. **数据库类型**: 数据库类型变更需要数据迁移，不能随意切换

## 依赖关系

Common 模块被以下模块依赖：
- Model 模块：使用配置和常量
- Controller 模块：使用加密和验证功能
- Middleware 模块：使用速率限制配置
- Logger 模块：使用调试配置
- Setting 模块：使用配置管理功能

该模块是整个系统的基础，任何对该模块的修改都需要充分测试以确保不影响依赖模块的正常运行。
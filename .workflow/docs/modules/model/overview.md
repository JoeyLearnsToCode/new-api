# Model 模块文档

## 概述

Model 模块是 new-api 项目的数据访问层，负责定义数据模型、数据库操作和业务逻辑。该模块使用 GORM 作为 ORM 框架，支持 MySQL、PostgreSQL 和 SQLite 三种数据库。

## 模块结构

```
model/
├── ability.go              # 能力模型 - 渠道支持的模型能力
├── channel_cache_test.go    # 渠道缓存测试
├── channel_cache.go         # 渠道缓存管理
├── channel.go              # 渠道模型 - 核心数据模型
├── log.go                  # 日志模型
├── main.go                 # 数据库初始化和配置
├── midjourney.go           # Midjourney 集成模型
├── missing_models.go       # 缺失模型管理
├── model_extra.go          # 模型扩展信息
├── model_meta.go           # 模型元数据
├── option.go               # 系统选项配置
├── prefill_group.go        # 预填充用户组
├── pricing_default.go      # 默认定价策略
├── pricing_refresh.go      # 定价刷新机制
├── pricing.go              # 定价模型
├── quota.go                # 配额管理
├── redemption.go           # 兑换码模型
├── token.go                # 令牌模型
├── two_fa.go               # 双因子认证模型
└── user.go                 # 用户模型
```

## 核心数据模型

### 1. 渠道模型 (Channel)

#### 基础结构

```go
type Channel struct {
    Id                 int     `json:"id"`
    Type               int     `json:"type" gorm:"default:0"`
    Key                string  `json:"key" gorm:"not null"`
    OpenAIOrganization *string `json:"openai_organization"`
    TestModel          *string `json:"test_model"`
    Status             int     `json:"status" gorm:"default:1"`
    Name               string  `json:"name" gorm:"index"`
    Weight             *uint   `json:"weight" gorm:"default:0"`
    CreatedTime        int64   `json:"created_time" gorm:"bigint"`
    TestTime           int64   `json:"test_time" gorm:"bigint"`
    ResponseTime       int     `json:"response_time"`
    BaseURL            *string `json:"base_url" gorm:"column:base_url;default:''"`
    Other              string  `json:"other"`
    Balance            float64 `json:"balance"`
    BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`
    Models             string  `json:"models"`
    Group              string  `json:"group" gorm:"type:varchar(64);default:'default'"`
    UsedQuota          int64   `json:"used_quota" gorm:"bigint;default:0"`
    ModelMapping       *string `json:"model_mapping" gorm:"type:text"`
    StatusCodeMapping  *string `json:"status_code_mapping" gorm:"type:varchar(1024);default:''"`
    Priority           *int64  `json:"priority" gorm:"bigint;default:0"`
    AutoBan            *int    `json:"auto_ban" gorm:"default:1"`
    OtherInfo          string  `json:"other_info"`
    Tag                *string `json:"tag" gorm:"index"`
    Setting            *string `json:"setting" gorm:"type:text"`
    ParamOverride      *string `json:"param_override" gorm:"type:text"`
    HeaderOverride     *string `json:"header_override" gorm:"type:text"`
    Remark             string  `json:"remark,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
    ChannelInfo        ChannelInfo `json:"channel_info" gorm:"type:json"`
    OtherSettings      string  `json:"settings" gorm:"column:settings"`
    Keys               []string `json:"-" gorm:"-"` // 缓存信息
}
```

#### 多密钥信息结构

```go
type ChannelInfo struct {
    IsMultiKey             bool                  `json:"is_multi_key"`
    MultiKeySize           int                   `json:"multi_key_size"`
    MultiKeyStatusList     map[int]int           `json:"multi_key_status_list"`
    MultiKeyDisabledReason map[int]string        `json:"multi_key_disabled_reason,omitempty"`
    MultiKeyDisabledTime   map[int]int64         `json:"multi_key_disabled_time,omitempty"`
    MultiKeyPollingIndex   int                   `json:"multi_key_polling_index"`
    MultiKeyMode           constant.MultiKeyMode `json:"multi_key_mode"`
}
```

#### 核心方法

| 方法 | 功能描述 |
|------|----------|
| `GetKeys()` | 解析并返回渠道的所有密钥 |
| `GetNextEnabledKey()` | 获取下一个可用的密钥（支持轮询和随机模式） |
| `GetModels()` | 获取渠道支持的模型列表 |
| `GetGroups()` | 获取渠道所属的用户组 |
| `GetBaseURL()` | 获取渠道的基础 URL |
| `GetSetting()` | 获取渠道的额外设置 |
| `GetOtherSettings()` | 获取其他配置信息 |
| `ValidateSettings()` | 验证渠道设置的有效性 |
| `Insert()` | 插入新渠道 |
| `Update()` | 更新渠道信息 |
| `Delete()` | 删除渠道 |
| `UpdateBalance()` | 更新渠道余额 |
| `UpdateResponseTime()` | 更新响应时间 |

### 2. 数据库初始化 (main.go)

#### 支持的数据库类型

```go
// 数据库类型常量
const (
    DatabaseTypeMySQL      = "mysql"
    DatabaseTypePostgreSQL = "postgres"
    DatabaseTypeSQLite     = "sqlite"
)
```

#### 初始化流程

1. **环境变量读取**: 从环境变量获取数据库连接信息
2. **驱动选择**: 根据数据库类型选择对应的 GORM 驱动
3. **连接建立**: 建立数据库连接并配置连接池
4. **表结构同步**: 自动迁移数据库表结构
5. **索引创建**: 创建必要的数据库索引

#### 连接配置

```go
// MySQL 配置示例
dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
    username, password, host, database)

// PostgreSQL 配置示例
dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
    host, username, password, database, port)

// SQLite 配置示例
dsn := "./database.db"
```

## 数据库操作模式

### 1. 基础 CRUD 操作

#### 查询操作

```go
// 单条查询
func GetChannelById(id int, selectAll bool) (*Channel, error) {
    channel := &Channel{Id: id}
    var err error = nil
    if selectAll {
        err = DB.First(channel, "id = ?", id).Error
    } else {
        err = DB.First(channel, "id = ?", id).Error
    }
    return channel, err
}

// 分页查询
func GetAllChannels(startIdx int, num int, selectAll bool, idSort bool) ([]*Channel, error) {
    var channels []*Channel
    order := "priority desc"
    if idSort {
        order = "id desc"
    }
    if selectAll {
        err = DB.Order(order).Find(&channels).Error
    } else {
        err = DB.Order(order).Limit(num).Offset(startIdx).Find(&channels).Error
    }
    return channels, err
}
```

#### 批量操作

```go
// 批量插入
func BatchInsertChannels(channels []Channel) error {
    if len(channels) == 0 {
        return nil
    }
    tx := DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    for _, chunk := range lo.Chunk(channels, 50) {
        if err := tx.Create(&chunk).Error; err != nil {
            tx.Rollback()
            return err
        }
    }
    return tx.Commit().Error
}

// 批量删除
func BatchDeleteChannels(ids []int) error {
    tx := DB.Begin()
    for _, chunk := range lo.Chunk(ids, 200) {
        if err := tx.Where("id in (?)", chunk).Delete(&Channel{}).Error; err != nil {
            tx.Rollback()
            return err
        }
    }
    return tx.Commit().Error
}
```

### 2. 高级查询功能

#### 搜索查询

```go
func SearchChannels(keyword string, group string, model string, idSort bool) ([]*Channel, error) {
    var channels []*Channel
    baseQuery := DB.Model(&Channel{})
    
    // 构造复杂的WHERE条件
    var whereClause string
    var args []interface{}
    
    if group != "" && group != "null" {
        whereClause = "(id = ? OR name LIKE ? OR key = ? OR base_url LIKE ?) AND models LIKE ? AND group LIKE ?"
        args = append(args, common.String2Int(keyword), "%"+keyword+"%", keyword, "%"+keyword+"%", "%"+model+"%", "%,"+group+",%")
    } else {
        whereClause = "(id = ? OR name LIKE ? OR key = ? OR base_url LIKE ?) AND models LIKE ?"
        args = append(args, common.String2Int(keyword), "%"+keyword+"%", keyword, "%"+keyword+"%", "%"+model+"%")
    }
    
    err := baseQuery.Where(whereClause, args...).Order(order).Find(&channels).Error
    return channels, err
}
```

#### 标签查询

```go
func GetChannelsByTag(tag string, idSort bool) ([]*Channel, error) {
    var channels []*Channel
    order := "priority desc"
    if idSort {
        order = "id desc"
    }
    err := DB.Where("tag = ?", tag).Order(order).Find(&channels).Error
    return channels, err
}
```

## 多密钥管理机制

### 1. 密钥获取策略

#### 轮询模式 (Polling)

```go
case constant.MultiKeyModePolling:
    // 使用渠道特定的锁确保线程安全轮询
    channelInfo, err := CacheGetChannelInfo(channel.Id)
    if err != nil {
        return "", 0, types.NewError(err, types.ErrorCodeGetChannelFailed)
    }
    
    // 从保存的轮询索引开始查找下一个启用的密钥
    start := channelInfo.MultiKeyPollingIndex
    if start < 0 || start >= len(keys) {
        start = 0
    }
    
    for i := 0; i < len(keys); i++ {
        idx := (start + i) % len(keys)
        if getStatus(idx) == common.ChannelStatusEnabled {
            // 更新轮询索引到下一个位置
            channel.ChannelInfo.MultiKeyPollingIndex = (idx + 1) % len(keys)
            return keys[idx], idx, nil
        }
    }
```

#### 随机模式 (Random)

```go
case constant.MultiKeyModeRandom:
    // 随机选择一个启用的密钥
    selectedIdx := enabledIdx[rand.Intn(len(enabledIdx))]
    return keys[selectedIdx], selectedIdx, nil
```

### 2. 密钥状态管理

#### 状态定义

```go
const (
    ChannelStatusEnabled         = 1  // 启用
    ChannelStatusManuallyDisabled = 2  // 手动禁用
    ChannelStatusAutoDisabled    = 3  // 自动禁用
)
```

#### 状态更新机制

```go
func handlerMultiKeyUpdate(channel *Channel, usingKey string, status int, reason string) {
    keys := channel.GetKeys()
    var keyIndex int
    for i, key := range keys {
        if key == usingKey {
            keyIndex = i
            break
        }
    }
    
    if channel.ChannelInfo.MultiKeyStatusList == nil {
        channel.ChannelInfo.MultiKeyStatusList = make(map[int]int)
    }
    
    if status == common.ChannelStatusEnabled {
        delete(channel.ChannelInfo.MultiKeyStatusList, keyIndex)
    } else {
        channel.ChannelInfo.MultiKeyStatusList[keyIndex] = status
        // 记录禁用原因和时间
        channel.ChannelInfo.MultiKeyDisabledReason[keyIndex] = reason
        channel.ChannelInfo.MultiKeyDisabledTime[keyIndex] = common.GetTimestamp()
    }
}
```

## 缓存机制

### 1. 渠道缓存

#### 缓存结构

```go
var channelCache sync.Map // 渠道缓存
var channelInfoCache sync.Map // 渠道信息缓存
```

#### 缓存操作

```go
// 获取缓存渠道
func CacheGetChannel(channelId int) (*Channel, error) {
    if !common.MemoryCacheEnabled {
        return GetChannelById(channelId, true)
    }
    
    if channel, ok := channelCache.Load(channelId); ok {
        return channel.(*Channel), nil
    }
    
    // 缓存未命中，从数据库加载
    channel, err := GetChannelById(channelId, true)
    if err != nil {
        return nil, err
    }
    
    channelCache.Store(channelId, channel)
    return channel, nil
}

// 更新缓存
func CacheUpdateChannel(channel *Channel) {
    if !common.MemoryCacheEnabled {
        return
    }
    channelCache.Store(channel.Id, channel)
}
```

### 2. 缓存初始化

```go
func InitChannelCache() {
    if !common.MemoryCacheEnabled {
        return
    }
    
    channels, err := GetAllChannels(0, 0, true, false)
    if err != nil {
        common.SysLog("Failed to initialize channel cache: " + err.Error())
        return
    }
    
    for _, channel := range channels {
        channelCache.Store(channel.Id, channel)
    }
}
```

## 并发安全机制

### 1. 渠道轮询锁

```go
// 为每个渠道ID存储锁，确保线程安全的轮询
var channelPollingLocks sync.Map

// 获取或创建渠道轮询锁
func GetChannelPollingLock(channelId int) *sync.Mutex {
    if lock, exists := channelPollingLocks.Load(channelId); exists {
        return lock.(*sync.Mutex)
    }
    
    newLock := &sync.Mutex{}
    actual, _ := channelPollingLocks.LoadOrStore(channelId, newLock)
    return actual.(*sync.Mutex)
}
```

### 2. 状态更新锁

```go
var channelStatusLock sync.Mutex

func UpdateChannelStatus(channelId int, usingKey string, status int, reason string) bool {
    if common.MemoryCacheEnabled {
        channelStatusLock.Lock()
        defer channelStatusLock.Unlock()
        
        // 更新缓存中的状态
        channelCache, _ := CacheGetChannel(channelId)
        if channelCache != nil && channelCache.ChannelInfo.IsMultiKey {
            pollingLock := GetChannelPollingLock(channelId)
            pollingLock.Lock()
            handlerMultiKeyUpdate(channelCache, usingKey, status, reason)
            pollingLock.Unlock()
        }
    }
    
    // 更新数据库
    // ...
}
```

## 数据库兼容性

### 1. 跨数据库查询

```go
// 根据数据库类型使用不同的列名引用
var commonGroupCol string
var commonKeyCol string

func initCol() {
    if common.UsingPostgreSQL {
        commonGroupCol = `"group"`
        commonKeyCol = `"key"`
    } else {
        commonGroupCol = "`group`"
        commonKeyCol = "`key`"
    }
}
```

### 2. 数据类型适配

```go
// JSON 字段的数据库适配
func (c ChannelInfo) Value() (driver.Value, error) {
    return common.Marshal(&c)
}

func (c *ChannelInfo) Scan(value interface{}) error {
    bytesValue, _ := value.([]byte)
    return common.Unmarshal(bytesValue, c)
}
```

## 性能优化

### 1. 批量操作

- 使用 `lo.Chunk` 进行分批处理，避免单次操作数据量过大
- 事务处理确保数据一致性
- 合理的批次大小（50-200条记录）

### 2. 索引优化

```go
// 在重要字段上创建索引
Name     string  `gorm:"index"`
Tag      *string `gorm:"index"`
Group    string  `gorm:"type:varchar(64);default:'default'"`
```

### 3. 查询优化

- 使用预编译语句避免 SQL 注入
- 合理使用 `Select` 和 `Omit` 减少数据传输
- 分页查询避免全表扫描

## 扩展指南

### 1. 添加新字段

1. 在结构体中添加字段定义
2. 添加相应的 getter/setter 方法
3. 更新数据库迁移脚本
4. 更新相关的业务逻辑

### 2. 添加新模型

1. 定义模型结构体
2. 实现必要的接口方法
3. 添加数据库操作函数
4. 更新初始化代码

### 3. 性能调优

1. 监控慢查询
2. 优化索引策略
3. 调整缓存策略
4. 优化批量操作

## 最佳实践

### 1. 数据模型设计
- 合理使用指针字段处理可选值
- 使用 JSON 字段存储复杂数据结构
- 适当的字段长度和类型选择

### 2. 并发处理
- 使用适当的锁机制保证线程安全
- 避免长时间持有锁
- 合理的锁粒度设计

### 3. 错误处理
- 统一的错误返回格式
- 详细的错误日志记录
- 优雅的错误恢复机制

### 4. 缓存策略
- 合理的缓存过期策略
- 及时的缓存更新机制
- 缓存穿透和雪崩防护
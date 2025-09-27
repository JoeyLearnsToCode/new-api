# Service 模块文档

## 概述

Service 模块是 new-api 项目的业务逻辑层，负责处理核心业务逻辑、外部服务集成、错误处理和通知机制。该模块在 Controller 和 Model 之间提供了一个抽象层，封装了复杂的业务规则和外部 API 调用。

## 模块结构

```
service/
├── audio.go                # 音频处理服务
├── channel.go              # 渠道管理服务
├── convert.go              # 数据转换服务
├── download.go             # 文件下载服务
├── epay.go                 # 电子支付集成服务
├── error.go                # 错误处理服务
├── file_decoder.go         # 文件解码服务
├── http_client.go          # HTTP 客户端服务
├── http.go                 # HTTP 工具服务
├── image.go                # 图像处理服务
├── log_info_generate.go    # 日志信息生成服务
├── midjourney.go           # Midjourney 集成服务
├── notify-limit.go         # 通知限制服务
├── pre_consume_quota.go    # 预消费配额服务
├── quota.go                # 配额管理服务
├── relay.go                # 请求转发服务
├── token.go                # 令牌管理服务
└── user.go                 # 用户管理服务
```

## 核心功能

### 1. 渠道管理服务 (channel.go)

#### 渠道状态管理

```go
// 禁用渠道并发送通知
func DisableChannel(channelError types.ChannelError, reason string) {
    common.SysLog(fmt.Sprintf("通道「%s」（#%d）发生错误，准备禁用，原因：%s", 
        channelError.ChannelName, channelError.ChannelId, reason))

    // 检查是否启用自动禁用功能
    if !channelError.AutoBan {
        common.SysLog(fmt.Sprintf("通道「%s」（#%d）未启用自动禁用功能，跳过禁用操作", 
            channelError.ChannelName, channelError.ChannelId))
        return
    }

    success := model.UpdateChannelStatus(channelError.ChannelId, channelError.UsingKey, 
        common.ChannelStatusAutoDisabled, reason)
    if success {
        subject := fmt.Sprintf("通道「%s」（#%d）已被禁用", channelError.ChannelName, channelError.ChannelId)
        content := fmt.Sprintf("通道「%s」（#%d）已被禁用，原因：%s", 
            channelError.ChannelName, channelError.ChannelId, reason)
        NotifyRootUser(formatNotifyType(channelError.ChannelId, common.ChannelStatusAutoDisabled), 
            subject, content)
    }
}

// 启用渠道
func EnableChannel(channelId int, usingKey string, channelName string) {
    success := model.UpdateChannelStatus(channelId, usingKey, common.ChannelStatusEnabled, "")
    if success {
        subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
        content := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
        NotifyRootUser(formatNotifyType(channelId, common.ChannelStatusEnabled), subject, content)
    }
}
```

#### 智能渠道管理

```go
// 判断是否应该禁用渠道
func ShouldDisableChannel(channelType int, err *types.NewAPIError) bool {
    if !common.AutomaticDisableChannelEnabled {
        return false
    }
    if err == nil {
        return false
    }
    
    // 渠道错误检查
    if types.IsChannelError(err) {
        return true
    }
    
    // HTTP 状态码检查
    if err.StatusCode == http.StatusUnauthorized {
        return true
    }
    if err.StatusCode == http.StatusForbidden {
        switch channelType {
        case constant.ChannelTypeGemini:
            return true
        }
    }
    
    // OpenAI 错误码检查
    oaiErr := err.ToOpenAIError()
    switch oaiErr.Code {
    case "invalid_api_key":
        return true
    case "account_deactivated":
        return true
    case "billing_not_active":
        return true
    }
    
    // 错误类型检查
    switch oaiErr.Type {
    case "insufficient_quota":
        return true
    case "authentication_error":
        return true
    case "permission_error":
        return true
    }
    
    // 关键词匹配检查
    lowerMessage := strings.ToLower(err.Error())
    search, _ := AcSearch(lowerMessage, operation_setting.AutomaticDisableKeywords, true)
    return search
}

// 判断是否应该启用渠道
func ShouldEnableChannel(newAPIError *types.NewAPIError, status int) bool {
    if !common.AutomaticEnableChannelEnabled {
        return false
    }
    if newAPIError != nil {
        return false
    }
    if status != common.ChannelStatusAutoDisabled {
        return false
    }
    return true
}
```

#### 渠道过期管理

```go
// 检查渠道是否过期
func IsChannelExpired(channel *model.Channel) bool {
    setting := channel.GetSetting()
    if setting.ExpirationTime == "" {
        return false // 未设置过期时间
    }
    
    // 解析 RFC3339 格式的时间字符串
    expirationTime, err := time.Parse(time.RFC3339, setting.ExpirationTime)
    if err != nil {
        common.SysLog(fmt.Sprintf("Failed to parse expiration time for channel %d: %v", channel.Id, err))
        return false // 解析失败不过期渠道
    }
    
    // 与当前 UTC 时间比较
    return time.Now().UTC().After(expirationTime.UTC())
}

// 扫描并禁用过期渠道
func ScanAndDisableExpiredChannels() {
    channels, err := model.GetAllChannels(0, 0, true, false)
    if err != nil {
        common.SysLog(fmt.Sprintf("Failed to get channels for expiration check: %v", err))
        return
    }
    
    expiredCount := 0
    for _, channel := range channels {
        // 只检查启用状态的渠道
        if channel.Status == common.ChannelStatusEnabled && IsChannelExpired(channel) {
            DisableExpiredChannel(channel)
            expiredCount++
        }
    }
    
    if expiredCount > 0 {
        common.SysLog(fmt.Sprintf("Expired channel scan completed, disabled %d channels", expiredCount))
    }
}
```

### 2. HTTP 服务 (http.go)

#### 响应处理工具

```go
// 优雅地关闭响应体
func CloseResponseBodyGracefully(httpResponse *http.Response) {
    if httpResponse == nil || httpResponse.Body == nil {
        return
    }
    err := httpResponse.Body.Close()
    if err != nil {
        common.SysError("failed to close response body: " + err.Error())
    }
}

// 优雅地复制字节数据到响应
func IOCopyBytesGracefully(c *gin.Context, src *http.Response, data []byte) {
    if c.Writer == nil {
        return
    }

    body := io.NopCloser(bytes.NewBuffer(data))

    // 复制响应头（避免在解析响应体之前设置头部）
    if src != nil {
        for k, v := range src.Header {
            // 避免设置 Content-Length
            if k == "Content-Length" {
                continue
            }
            c.Writer.Header().Set(k, v[0])
        }
    }

    // 手动设置 Content-Length 头部
    c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

    // 写入状态码和头部
    if src != nil {
        c.Writer.WriteHeader(src.StatusCode)
    }
}
```

### 3. 配额管理服务 (quota.go)

#### 配额计算和扣除

```go
// 预消费配额
func PreConsumeQuota(ctx context.Context, userId int, channelId int, promptTokens int, 
    completionTokens int, modelName string, ratio float64, preConsumedQuota int) error {
    
    // 计算实际消费的配额
    quota := 0
    if promptTokens != 0 {
        quota = int(float64(promptTokens) * ratio)
    }
    if completionTokens != 0 {
        quota += int(float64(completionTokens) * ratio)
    }
    
    // 检查用户配额是否足够
    userQuota, err := model.GetUserQuota(userId, false)
    if err != nil {
        return fmt.Errorf("获取用户配额失败: %v", err)
    }
    
    if userQuota < quota {
        return fmt.Errorf("用户配额不足，需要 %d，剩余 %d", quota, userQuota)
    }
    
    // 扣除配额
    err = model.DecreaseUserQuota(userId, quota)
    if err != nil {
        return fmt.Errorf("扣除用户配额失败: %v", err)
    }
    
    // 更新渠道使用量
    model.UpdateChannelUsedQuota(channelId, quota)
    
    return nil
}

// 退还配额
func RefundQuota(ctx context.Context, userId int, quota int, reason string) error {
    err := model.IncreaseUserQuota(userId, quota)
    if err != nil {
        common.SysLog(fmt.Sprintf("退还用户配额失败: userId=%d, quota=%d, reason=%s, error=%v", 
            userId, quota, reason, err))
        return err
    }
    
    // 记录退还日志
    model.RecordLog(userId, model.LogTypeSystem, 
        fmt.Sprintf("配额退还: %d，原因: %s", quota, reason))
    
    return nil
}
```

### 4. 错误处理服务 (error.go)

#### 统一错误处理

```go
// 错误分类和处理
type ErrorHandler struct {
    retryableErrors   map[string]bool
    fatalErrors      map[string]bool
    temporaryErrors  map[string]bool
}

// 判断错误是否可重试
func (eh *ErrorHandler) IsRetryable(err error) bool {
    if err == nil {
        return false
    }
    
    errorMsg := strings.ToLower(err.Error())
    
    // 检查致命错误
    for fatalError := range eh.fatalErrors {
        if strings.Contains(errorMsg, fatalError) {
            return false
        }
    }
    
    // 检查可重试错误
    for retryableError := range eh.retryableErrors {
        if strings.Contains(errorMsg, retryableError) {
            return true
        }
    }
    
    return false
}

// 错误恢复策略
func (eh *ErrorHandler) HandleError(err error, context map[string]interface{}) error {
    if err == nil {
        return nil
    }
    
    // 记录错误日志
    common.SysError(fmt.Sprintf("处理错误: %v, 上下文: %+v", err, context))
    
    // 根据错误类型采取不同的处理策略
    if eh.IsRetryable(err) {
        return eh.handleRetryableError(err, context)
    } else {
        return eh.handleFatalError(err, context)
    }
}
```

### 5. 通知服务 (notify-limit.go)

#### 限流通知机制

```go
// 通知限流器
type NotifyLimiter struct {
    limits map[string]*rate.Limiter
    mutex  sync.RWMutex
}

// 获取或创建限流器
func (nl *NotifyLimiter) getLimiter(key string) *rate.Limiter {
    nl.mutex.RLock()
    limiter, exists := nl.limits[key]
    nl.mutex.RUnlock()
    
    if !exists {
        nl.mutex.Lock()
        limiter = rate.NewLimiter(rate.Every(time.Minute), 1) // 每分钟最多1次
        nl.limits[key] = limiter
        nl.mutex.Unlock()
    }
    
    return limiter
}

// 发送限流通知
func (nl *NotifyLimiter) SendNotification(notifyType string, subject string, content string) bool {
    limiter := nl.getLimiter(notifyType)
    
    if limiter.Allow() {
        // 发送通知
        return sendActualNotification(subject, content)
    }
    
    // 被限流，跳过发送
    common.SysLog(fmt.Sprintf("通知被限流: %s", notifyType))
    return false
}
```

### 6. 文件处理服务

#### 文件解码服务 (file_decoder.go)

```go
// 文件解码器接口
type FileDecoder interface {
    Decode(data []byte) (interface{}, error)
    GetSupportedTypes() []string
}

// JSON 文件解码器
type JSONDecoder struct{}

func (jd *JSONDecoder) Decode(data []byte) (interface{}, error) {
    var result interface{}
    err := json.Unmarshal(data, &result)
    return result, err
}

func (jd *JSONDecoder) GetSupportedTypes() []string {
    return []string{"application/json", "text/json"}
}

// 文件解码管理器
type FileDecoderManager struct {
    decoders map[string]FileDecoder
}

func (fdm *FileDecoderManager) RegisterDecoder(contentType string, decoder FileDecoder) {
    fdm.decoders[contentType] = decoder
}

func (fdm *FileDecoderManager) DecodeFile(contentType string, data []byte) (interface{}, error) {
    decoder, exists := fdm.decoders[contentType]
    if !exists {
        return nil, fmt.Errorf("不支持的文件类型: %s", contentType)
    }
    
    return decoder.Decode(data)
}
```

#### 下载服务 (download.go)

```go
// 文件下载配置
type DownloadConfig struct {
    MaxFileSize   int64         // 最大文件大小
    Timeout       time.Duration // 下载超时时间
    AllowedTypes  []string      // 允许的文件类型
    UserAgent     string        // 用户代理
}

// 下载文件
func DownloadFile(url string, config DownloadConfig) ([]byte, string, error) {
    // 创建 HTTP 客户端
    client := &http.Client{
        Timeout: config.Timeout,
    }
    
    // 创建请求
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, "", fmt.Errorf("创建请求失败: %v", err)
    }
    
    if config.UserAgent != "" {
        req.Header.Set("User-Agent", config.UserAgent)
    }
    
    // 发送请求
    resp, err := client.Do(req)
    if err != nil {
        return nil, "", fmt.Errorf("下载失败: %v", err)
    }
    defer resp.Body.Close()
    
    // 检查响应状态
    if resp.StatusCode != http.StatusOK {
        return nil, "", fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
    }
    
    // 检查文件大小
    if resp.ContentLength > config.MaxFileSize {
        return nil, "", fmt.Errorf("文件过大: %d bytes", resp.ContentLength)
    }
    
    // 检查文件类型
    contentType := resp.Header.Get("Content-Type")
    if !isAllowedType(contentType, config.AllowedTypes) {
        return nil, "", fmt.Errorf("不允许的文件类型: %s", contentType)
    }
    
    // 读取文件内容
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, "", fmt.Errorf("读取文件内容失败: %v", err)
    }
    
    return data, contentType, nil
}
```

## 外部服务集成

### 1. Midjourney 集成 (midjourney.go)

```go
// Midjourney 客户端
type MidjourneyClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

// 创建图像生成任务
func (mc *MidjourneyClient) CreateImageTask(prompt string, options MidjourneyOptions) (*MidjourneyTask, error) {
    requestBody := map[string]interface{}{
        "prompt": prompt,
        "aspect_ratio": options.AspectRatio,
        "quality": options.Quality,
        "style": options.Style,
    }
    
    data, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("序列化请求失败: %v", err)
    }
    
    req, err := http.NewRequest("POST", mc.baseURL+"/v1/imagine", bytes.NewBuffer(data))
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %v", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+mc.apiKey)
    
    resp, err := mc.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("发送请求失败: %v", err)
    }
    defer resp.Body.Close()
    
    var task MidjourneyTask
    err = json.NewDecoder(resp.Body).Decode(&task)
    if err != nil {
        return nil, fmt.Errorf("解析响应失败: %v", err)
    }
    
    return &task, nil
}
```

### 2. 支付服务集成 (epay.go)

```go
// 电子支付客户端
type EPayClient struct {
    merchantId string
    secretKey  string
    baseURL    string
}

// 创建支付订单
func (epc *EPayClient) CreatePaymentOrder(order PaymentOrder) (*PaymentResponse, error) {
    // 构建签名参数
    params := map[string]string{
        "merchant_id": epc.merchantId,
        "order_id":    order.OrderId,
        "amount":      fmt.Sprintf("%.2f", order.Amount),
        "currency":    order.Currency,
        "notify_url":  order.NotifyURL,
        "return_url":  order.ReturnURL,
    }
    
    // 生成签名
    signature := epc.generateSignature(params)
    params["signature"] = signature
    
    // 发送请求
    resp, err := epc.sendRequest("POST", "/api/payment/create", params)
    if err != nil {
        return nil, fmt.Errorf("创建支付订单失败: %v", err)
    }
    
    var paymentResp PaymentResponse
    err = json.Unmarshal(resp, &paymentResp)
    if err != nil {
        return nil, fmt.Errorf("解析支付响应失败: %v", err)
    }
    
    return &paymentResp, nil
}
```

## 业务规则引擎

### 1. 规则定义

```go
// 业务规则接口
type BusinessRule interface {
    Evaluate(context map[string]interface{}) (bool, error)
    GetName() string
    GetDescription() string
}

// 配额限制规则
type QuotaLimitRule struct {
    MaxDailyQuota int
    MaxMonthlyQuota int
}

func (qlr *QuotaLimitRule) Evaluate(context map[string]interface{}) (bool, error) {
    userId, ok := context["user_id"].(int)
    if !ok {
        return false, fmt.Errorf("缺少用户ID")
    }
    
    // 检查日配额
    dailyUsage, err := model.GetUserDailyUsage(userId)
    if err != nil {
        return false, err
    }
    
    if dailyUsage >= qlr.MaxDailyQuota {
        return false, fmt.Errorf("超出日配额限制")
    }
    
    // 检查月配额
    monthlyUsage, err := model.GetUserMonthlyUsage(userId)
    if err != nil {
        return false, err
    }
    
    if monthlyUsage >= qlr.MaxMonthlyQuota {
        return false, fmt.Errorf("超出月配额限制")
    }
    
    return true, nil
}
```

### 2. 规则引擎

```go
// 规则引擎
type RuleEngine struct {
    rules []BusinessRule
}

func (re *RuleEngine) AddRule(rule BusinessRule) {
    re.rules = append(re.rules, rule)
}

func (re *RuleEngine) EvaluateAll(context map[string]interface{}) error {
    for _, rule := range re.rules {
        passed, err := rule.Evaluate(context)
        if err != nil {
            return fmt.Errorf("规则 %s 评估失败: %v", rule.GetName(), err)
        }
        
        if !passed {
            return fmt.Errorf("规则 %s 验证失败", rule.GetName())
        }
    }
    
    return nil
}
```

## 性能监控和指标

### 1. 性能指标收集

```go
// 性能指标收集器
type MetricsCollector struct {
    requestCount    int64
    errorCount      int64
    responseTime    time.Duration
    mutex          sync.RWMutex
}

func (mc *MetricsCollector) RecordRequest(duration time.Duration, hasError bool) {
    mc.mutex.Lock()
    defer mc.mutex.Unlock()
    
    mc.requestCount++
    mc.responseTime += duration
    
    if hasError {
        mc.errorCount++
    }
}

func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
    mc.mutex.RLock()
    defer mc.mutex.RUnlock()
    
    avgResponseTime := time.Duration(0)
    if mc.requestCount > 0 {
        avgResponseTime = mc.responseTime / time.Duration(mc.requestCount)
    }
    
    return map[string]interface{}{
        "request_count":       mc.requestCount,
        "error_count":         mc.errorCount,
        "error_rate":          float64(mc.errorCount) / float64(mc.requestCount),
        "avg_response_time":   avgResponseTime.Milliseconds(),
    }
}
```

## 最佳实践

### 1. 错误处理
- 统一的错误分类和处理策略
- 详细的错误日志记录
- 优雅的错误恢复机制
- 合理的重试策略

### 2. 性能优化
- 异步处理耗时操作
- 合理的缓存策略
- 连接池管理
- 资源及时释放

### 3. 安全考虑
- 输入参数验证
- 敏感信息脱敏
- 访问控制
- 审计日志

### 4. 可维护性
- 清晰的模块划分
- 统一的接口设计
- 完善的文档
- 充分的测试覆盖

## 扩展指南

### 1. 添加新服务
1. 定义服务接口
2. 实现具体功能
3. 添加错误处理
4. 集成到业务流程

### 2. 集成外部服务
1. 封装客户端
2. 处理认证和授权
3. 实现重试机制
4. 添加监控和日志

### 3. 优化性能
1. 识别性能瓶颈
2. 优化算法和数据结构
3. 使用缓存和异步处理
4. 监控和调优
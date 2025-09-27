# New API 统一接口文档

## 概述

本文档详细描述了 New API 系统的所有 REST API 端点。New API 是一个 OpenAI API 兼容的中继系统，支持多种 AI 模型提供商的统一访问。

## 基础信息

- **基础 URL**: `http://localhost:3000` (默认)
- **API 版本**: v1
- **认证方式**: Bearer Token / Session Cookie
- **数据格式**: JSON
- **编码**: UTF-8

## 认证机制

### 1. Bearer Token 认证
```http
Authorization: Bearer YOUR_API_TOKEN
```

### 2. Session 认证
通过登录接口获取 session cookie 进行认证。

### 3. 管理员认证
需要管理员角色的用户才能访问的接口。

## API 端点分类

### 1. 系统管理接口 (`/api`)

#### 1.1 系统状态
- **GET** `/api/status` - 获取系统状态
- **GET** `/api/uptime/status` - 获取运行时间状态
- **GET** `/api/setup` - 获取系统设置状态
- **POST** `/api/setup` - 初始化系统设置

#### 1.2 系统信息
- **GET** `/api/about` - 获取系统信息
- **GET** `/api/notice` - 获取通知信息
- **GET** `/api/home_page_content` - 获取主页内容
- **GET** `/api/pricing` - 获取定价信息

### 2. 用户管理接口 (`/api/user`)

#### 2.1 用户认证
- **POST** `/api/user/register` - 用户注册
- **POST** `/api/user/login` - 用户登录
- **POST** `/api/user/login/2fa` - 两步验证登录
- **GET** `/api/user/logout` - 用户登出

#### 2.2 用户信息管理
- **GET** `/api/user/self` - 获取当前用户信息
- **PUT** `/api/user/self` - 更新当前用户信息
- **DELETE** `/api/user/self` - 删除当前用户
- **GET** `/api/user/models` - 获取用户可用模型
- **GET** `/api/user/groups` - 获取用户组信息

#### 2.3 用户令牌管理
- **GET** `/api/user/token` - 生成访问令牌
- **GET** `/api/user/aff` - 获取推广码

#### 2.4 用户充值
- **GET** `/api/user/topup/info` - 获取充值信息
- **POST** `/api/user/topup` - 充值
- **POST** `/api/user/pay` - 请求支付
- **POST** `/api/user/amount` - 请求金额
- **POST** `/api/user/stripe/pay` - Stripe 支付
- **POST** `/api/user/stripe/amount` - Stripe 金额

#### 2.5 两步验证 (2FA)
- **GET** `/api/user/2fa/status` - 获取2FA状态
- **POST** `/api/user/2fa/setup` - 设置2FA
- **POST** `/api/user/2fa/enable` - 启用2FA
- **POST** `/api/user/2fa/disable` - 禁用2FA
- **POST** `/api/user/2fa/backup_codes` - 重新生成备用码

#### 2.6 用户管理 (管理员)
- **GET** `/api/user/` - 获取所有用户
- **GET** `/api/user/search` - 搜索用户
- **GET** `/api/user/:id` - 获取指定用户
- **POST** `/api/user/` - 创建用户
- **POST** `/api/user/manage` - 管理用户
- **PUT** `/api/user/` - 更新用户
- **DELETE** `/api/user/:id` - 删除用户

### 3. 渠道管理接口 (`/api/channel`)

#### 3.1 渠道基础操作
- **GET** `/api/channel/` - 获取所有渠道
- **GET** `/api/channel/search` - 搜索渠道
- **GET** `/api/channel/:id` - 获取指定渠道
- **POST** `/api/channel/` - 添加渠道
- **PUT** `/api/channel/` - 更新渠道
- **DELETE** `/api/channel/:id` - 删除渠道

#### 3.2 渠道测试与维护
- **GET** `/api/channel/test` - 测试所有渠道
- **GET** `/api/channel/test/:id` - 测试指定渠道
- **GET** `/api/channel/update_balance` - 更新所有渠道余额
- **GET** `/api/channel/update_balance/:id` - 更新指定渠道余额
- **POST** `/api/channel/fix` - 修复渠道能力

#### 3.3 渠道模型管理
- **GET** `/api/channel/models` - 获取渠道模型列表
- **GET** `/api/channel/models_enabled` - 获取启用的模型列表
- **GET** `/api/channel/fetch_models/:id` - 获取上游模型
- **POST** `/api/channel/fetch_models` - 批量获取模型

#### 3.4 渠道标签管理
- **POST** `/api/channel/tag/disabled` - 禁用标签渠道
- **POST** `/api/channel/tag/enabled` - 启用标签渠道
- **PUT** `/api/channel/tag` - 编辑标签渠道
- **POST** `/api/channel/batch/tag` - 批量设置渠道标签
- **GET** `/api/channel/tag/models` - 获取标签模型

#### 3.5 渠道批量操作
- **DELETE** `/api/channel/disabled` - 删除禁用的渠道
- **POST** `/api/channel/batch` - 批量删除渠道
- **POST** `/api/channel/copy/:id` - 复制渠道

#### 3.6 多密钥管理
- **POST** `/api/channel/multi_key/manage` - 管理多密钥

#### 3.7 渠道密钥获取
- **POST** `/api/channel/:id/key` - 获取渠道密钥 (需要2FA验证)

### 4. 令牌管理接口 (`/api/token`)

#### 4.1 令牌基础操作
- **GET** `/api/token/` - 获取所有令牌
- **GET** `/api/token/search` - 搜索令牌
- **GET** `/api/token/:id` - 获取指定令牌
- **POST** `/api/token/` - 添加令牌
- **PUT** `/api/token/` - 更新令牌
- **DELETE** `/api/token/:id` - 删除令牌
- **POST** `/api/token/batch` - 批量删除令牌

### 5. 使用统计接口 (`/api/usage`)

#### 5.1 令牌使用统计
- **GET** `/api/usage/token/` - 获取令牌使用统计

### 6. 兑换码管理接口 (`/api/redemption`)

#### 6.1 兑换码操作 (管理员)
- **GET** `/api/redemption/` - 获取所有兑换码
- **GET** `/api/redemption/search` - 搜索兑换码
- **GET** `/api/redemption/:id` - 获取指定兑换码
- **POST** `/api/redemption/` - 添加兑换码
- **PUT** `/api/redemption/` - 更新兑换码
- **DELETE** `/api/redemption/invalid` - 删除无效兑换码
- **DELETE** `/api/redemption/:id` - 删除兑换码

### 7. 日志管理接口 (`/api/log`)

#### 7.1 日志查看
- **GET** `/api/log/` - 获取所有日志 (管理员)
- **DELETE** `/api/log/` - 删除历史日志 (管理员)
- **GET** `/api/log/stat` - 获取日志统计 (管理员)
- **GET** `/api/log/self/stat` - 获取个人日志统计
- **GET** `/api/log/search` - 搜索所有日志 (管理员)
- **GET** `/api/log/self` - 获取个人日志
- **GET** `/api/log/self/search` - 搜索个人日志
- **GET** `/api/log/token` - 通过密钥获取日志

### 8. 数据统计接口 (`/api/data`)

#### 8.1 配额数据
- **GET** `/api/data/` - 获取所有配额数据 (管理员)
- **GET** `/api/data/self` - 获取个人配额数据

### 9. 组管理接口 (`/api/group`)

#### 9.1 组操作 (管理员)
- **GET** `/api/group/` - 获取所有组

### 10. 预填充组管理接口 (`/api/prefill_group`)

#### 10.1 预填充组操作 (管理员)
- **GET** `/api/prefill_group/` - 获取预填充组
- **POST** `/api/prefill_group/` - 创建预填充组
- **PUT** `/api/prefill_group/` - 更新预填充组
- **DELETE** `/api/prefill_group/:id` - 删除预填充组

### 11. Midjourney 接口 (`/api/mj`)

#### 11.1 Midjourney 操作
- **GET** `/api/mj/self` - 获取个人 Midjourney 信息
- **GET** `/api/mj/` - 获取所有 Midjourney 信息 (管理员)

### 12. 任务管理接口 (`/api/task`)

#### 12.1 任务查看
- **GET** `/api/task/self` - 获取个人任务
- **GET** `/api/task/` - 获取所有任务 (管理员)

### 13. 供应商管理接口 (`/api/vendors`)

#### 13.1 供应商操作 (管理员)
- **GET** `/api/vendors/` - 获取所有供应商
- **GET** `/api/vendors/search` - 搜索供应商
- **GET** `/api/vendors/:id` - 获取指定供应商
- **POST** `/api/vendors/` - 创建供应商
- **PUT** `/api/vendors/` - 更新供应商
- **DELETE** `/api/vendors/:id` - 删除供应商

### 14. 模型管理接口 (`/api/models`)

#### 14.1 模型操作 (管理员)
- **GET** `/api/models/sync_upstream/preview` - 预览上游同步
- **POST** `/api/models/sync_upstream` - 同步上游模型
- **GET** `/api/models/missing` - 获取缺失模型
- **GET** `/api/models/` - 获取所有模型元数据
- **GET** `/api/models/search` - 搜索模型元数据
- **GET** `/api/models/:id` - 获取指定模型元数据
- **POST** `/api/models/` - 创建模型元数据
- **PUT** `/api/models/` - 更新模型元数据
- **DELETE** `/api/models/:id` - 删除模型元数据

### 15. 系统配置接口 (`/api/option`)

#### 15.1 配置管理 (Root)
- **GET** `/api/option/` - 获取系统配置
- **PUT** `/api/option/` - 更新系统配置
- **POST** `/api/option/rest_model_ratio` - 重置模型比例
- **POST** `/api/option/migrate_console_setting` - 迁移控制台设置

### 16. 比例同步接口 (`/api/ratio_sync`)

#### 16.1 比例同步 (Root)
- **GET** `/api/ratio_sync/channels` - 获取可同步的渠道
- **POST** `/api/ratio_sync/fetch` - 获取上游比例

## OpenAI 兼容接口 (`/v1`)

### 1. 模型接口
- **GET** `/v1/models` - 获取可用模型列表
- **GET** `/v1/models/:model` - 获取指定模型信息

### 2. 聊天完成接口
- **POST** `/v1/chat/completions` - 聊天完成
- **POST** `/v1/completions` - 文本完成

### 3. Embedding 接口
- **POST** `/v1/embeddings` - 文本嵌入

### 4. 图像接口
- **POST** `/v1/images/generations` - 图像生成
- **POST** `/v1/images/edits` - 图像编辑
- **POST** `/v1/edits` - 编辑

### 5. 音频接口
- **POST** `/v1/audio/transcriptions` - 音频转录
- **POST** `/v1/audio/translations` - 音频翻译
- **POST** `/v1/audio/speech` - 文本转语音

### 6. 其他接口
- **POST** `/v1/moderations` - 内容审核
- **POST** `/v1/rerank` - 重排序
- **GET** `/v1/realtime` - 实时通信 (WebSocket)

### 7. Claude 兼容接口
- **POST** `/v1/messages` - Claude 消息接口

### 8. Responses 接口
- **POST** `/v1/responses` - OpenAI Responses

## Gemini 兼容接口 (`/v1beta`)

### 1. Gemini 模型接口
- **GET** `/v1beta/models` - 获取 Gemini 模型列表
- **GET** `/v1beta/openai/models` - 获取 OpenAI 兼容的模型列表

### 2. Gemini API 接口
- **POST** `/v1beta/models/*path` - Gemini API 路径

## 特殊功能接口

### 1. Midjourney 接口 (`/mj`)
- **GET** `/mj/image/:id` - 获取 Midjourney 图像
- **POST** `/mj/submit/imagine` - 提交想象任务
- **POST** `/mj/submit/action` - 提交动作任务
- **POST** `/mj/submit/change` - 提交变更任务
- **POST** `/mj/submit/describe` - 提交描述任务
- **POST** `/mj/submit/blend` - 提交混合任务
- **GET** `/mj/task/:id/fetch` - 获取任务结果
- **POST** `/mj/task/list-by-condition` - 按条件列出任务

### 2. Suno 音乐接口 (`/suno`)
- **POST** `/suno/submit/:action` - 提交 Suno 任务
- **POST** `/suno/fetch` - 获取结果
- **GET** `/suno/fetch/:id` - 获取指定结果

### 3. 游乐场接口 (`/pg`)
- **POST** `/pg/chat/completions` - 游乐场聊天完成

## OAuth 认证接口

### 1. OAuth 认证
- **GET** `/api/oauth/github` - GitHub OAuth
- **GET** `/api/oauth/oidc` - OIDC 认证
- **GET** `/api/oauth/linuxdo` - LinuxDo OAuth
- **GET** `/api/oauth/state` - 生成 OAuth 状态码
- **GET** `/api/oauth/wechat` - 微信认证
- **GET** `/api/oauth/wechat/bind` - 微信绑定
- **GET** `/api/oauth/email/bind` - 邮箱绑定
- **GET** `/api/oauth/telegram/login` - Telegram 登录
- **GET** `/api/oauth/telegram/bind` - Telegram 绑定

## 支付相关接口

### 1. Stripe 支付
- **POST** `/api/stripe/webhook` - Stripe webhook

### 2. 易支付
- **GET** `/api/user/epay/notify` - 易支付通知

## 错误码和响应格式

### 标准响应格式
```json
{
  "success": true,
  "message": "操作成功",
  "data": {}
}
```

### 错误响应格式
```json
{
  "success": false,
  "message": "错误信息",
  "error": "详细错误"
}
```

### 分页响应格式
```json
{
  "success": true,
  "message": "",
  "data": {
    "items": [],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

## 常用请求示例

### 1. 用户登录
```http
POST /api/user/login
Content-Type: application/json

{
  "username": "admin",
  "password": "123456"
}
```

### 2. 获取用户信息
```http
GET /api/user/self
Authorization: Bearer YOUR_TOKEN
```

### 3. 聊天完成
```http
POST /v1/chat/completions
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": "Hello, how are you?"
    }
  ]
}
```

### 4. 获取模型列表
```http
GET /v1/models
Authorization: Bearer YOUR_TOKEN
```

### 5. 添加渠道
```http
POST /api/channel/
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "channel": {
    "name": "测试渠道",
    "type": 1,
    "key": "sk-xxx",
    "models": "gpt-3.5-turbo,gpt-4",
    "status": 1
  },
  "mode": "single"
}
```

## 速率限制

- **全局 API 速率限制**: 应用于所有 `/api/*` 路径
- **关键操作速率限制**: 应用于登录、注册、支付等关键操作
- **邮箱验证速率限制**: 应用于邮箱验证相关操作
- **模型请求速率限制**: 应用于模型调用相关操作

## 中间件说明

- **CORS**: 跨域资源共享支持
- **压缩**: Gzip 压缩支持
- **统计**: 请求统计中间件
- **解压**: 请求解压中间件
- **缓存**: 缓存控制中间件
- **Turnstile**: Cloudflare Turnstile 验证

## 注意事项

1. 所有需要认证的接口都需要在请求头中包含有效的认证信息
2. 管理员接口需要管理员权限才能访问
3. Root 接口需要超级管理员权限才能访问
4. 部分接口有速率限制，请合理控制请求频率
5. 所有时间戳均为 Unix 时间戳格式
6. 文件上传接口支持多种格式，具体限制请参考系统配置
7. WebSocket 接口需要升级协议连接
8. 2FA 相关操作需要额外的验证步骤

## API 版本说明

- **v1**: 当前稳定版本，兼容 OpenAI API
- **v1beta**: Gemini 兼容版本，支持 Google AI 模型

本文档将随着系统更新持续维护，请以最新版本为准。
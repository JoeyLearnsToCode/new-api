# New API 系统概览文档

## 项目简介

New API 是一个基于 Go 语言开发的新一代大模型网关与AI资产管理系统。本项目在 [One API](https://github.com/songquanpeng/one-api) 的基础上进行二次开发，提供了更强大的功能和更友好的用户体验。

## 核心特性

### 🎯 系统定位
- **大模型网关**：统一管理和转发多种AI模型的API请求
- **AI资产管理**：提供完整的用户、渠道、令牌管理功能
- **多模型兼容**：支持50+种不同的AI服务提供商和模型

### 🚀 主要功能

#### 1. 多模型支持
- **文本生成**：OpenAI GPT系列、Claude、Gemini、国产大模型等
- **图像生成**：DALL-E、Midjourney、Stable Diffusion等
- **语音处理**：语音转文字、文字转语音
- **向量嵌入**：支持多种embedding模型
- **重排序**：Cohere、Jina等rerank模型

#### 2. 渠道管理
- **多渠道支持**：支持50+种AI服务提供商
- **负载均衡**：渠道加权随机分配
- **故障转移**：自动重试和故障切换
- **实时监控**：渠道状态监控和自动测试

#### 3. 用户管理
- **用户系统**：完整的用户注册、登录、权限管理
- **令牌管理**：API密钥生成、分组、权限控制
- **额度管理**：用户余额、消费记录、充值功能
- **多种登录**：支持GitHub、LinuxDO、Telegram、OIDC等

#### 4. 计费系统
- **灵活计费**：支持按token、按次数计费
- **实时统计**：详细的使用统计和数据分析
- **在线充值**：集成易支付等支付方式
- **缓存计费**：支持提示缓存的差异化计费

#### 5. 高级功能
- **实时对话**：支持OpenAI Realtime API
- **格式转换**：支持不同API格式间的转换
- **思考模式**：支持o系列和Claude思考模型
- **批量更新**：支持Midjourney、Suno等异步任务

## 技术架构

### 核心技术栈

#### 后端框架
- **Go 1.23.4**：主要开发语言
- **Gin**：HTTP Web框架，提供高性能的API服务
- **GORM**：ORM框架，支持多种数据库

#### 数据存储
- **SQLite**：默认数据库，轻量级部署
- **MySQL**：生产环境推荐，版本 >= 5.7.8
- **PostgreSQL**：企业级数据库，版本 >= 9.6
- **Redis**：缓存和会话存储

#### 前端技术
- **React**：现代化Web界面
- **嵌入式部署**：前端资源嵌入到Go二进制文件

#### 关键依赖
- **JWT**：用户身份认证
- **WebSocket**：实时通信支持
- **AWS SDK**：云服务集成
- **Stripe**：支付处理
- **多种AI SDK**：各大模型提供商的官方SDK

### 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend  │    │   Mobile App    │    │  Third Party    │
│    (React)      │    │                 │    │   Integration   │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────┴─────────────┐
                    │      New API Gateway      │
                    │       (Gin Router)        │
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │     Business Logic        │
                    │  ┌─────────┬─────────┐    │
                    │  │ Channel │ Billing │    │
                    │  │ Manager │ System  │    │
                    │  └─────────┴─────────┘    │
                    └─────────────┬─────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
    ┌─────┴─────┐         ┌───────┴───────┐       ┌───────┴───────┐
    │ Database  │         │     Cache     │       │   AI Providers │
    │(SQLite/   │         │   (Redis)     │       │   (50+ APIs)   │
    │MySQL/PG)  │         │               │       │                │
    └───────────┘         └───────────────┘       └───────────────┘
```

## 项目结构

### 目录结构
```
new-api/
├── common/          # 公共工具和配置
├── constant/        # 常量定义
├── controller/      # 控制器层
├── dto/            # 数据传输对象
├── logger/         # 日志系统
├── middleware/     # 中间件
├── model/          # 数据模型
├── relay/          # 转发核心逻辑
├── router/         # 路由配置
├── service/        # 业务服务层
├── setting/        # 配置管理
├── types/          # 类型定义
├── web/            # 前端资源
├── main.go         # 程序入口
└── go.mod          # 依赖管理
```

### 核心模块

#### 1. 转发模块 (relay/)
- **channel/**：各AI提供商的适配器
- **common/**：转发通用逻辑
- **helper/**：转发辅助工具
- **各种handler**：不同类型请求的处理器

#### 2. 控制器模块 (controller/)
- **用户管理**：user.go, token.go, group.go
- **渠道管理**：channel.go, channel-test.go
- **计费系统**：billing.go, pricing.go
- **第三方集成**：midjourney.go, task.go

#### 3. 数据模型 (model/)
- **核心实体**：用户、渠道、令牌、日志
- **缓存管理**：channel_cache.go, user_cache.go
- **数据库操作**：CRUD和复杂查询

#### 4. 业务服务 (service/)
- **HTTP客户端管理**
- **令牌编码器**
- **渠道扫描和管理**

## 部署方式

### 1. Docker部署（推荐）
```bash
docker run --name new-api -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v /home/ubuntu/data/new-api:/data \
  calciumion/new-api:latest
```

### 2. Docker Compose
```bash
git clone https://github.com/Calcium-Ion/new-api.git
cd new-api
docker-compose up -d
```

### 3. 二进制部署
- 下载对应平台的二进制文件
- 配置环境变量
- 直接运行

## 环境配置

### 核心环境变量
- `SQL_DSN`：数据库连接字符串
- `REDIS_CONN_STRING`：Redis连接字符串
- `SESSION_SECRET`：会话密钥
- `CRYPTO_SECRET`：加密密钥
- `PORT`：服务端口（默认3000）

### 功能开关
- `MEMORY_CACHE_ENABLED`：启用内存缓存
- `BATCH_UPDATE_ENABLED`：启用批量更新
- `GENERATE_DEFAULT_TOKEN`：为新用户生成默认令牌
- `GET_MEDIA_TOKEN`：统计图片token

## 监控与运维

### 性能监控
- **pprof**：Go程序性能分析
- **系统监控**：CPU、内存使用情况
- **业务监控**：请求量、错误率、响应时间

### 日志系统
- **结构化日志**：JSON格式日志输出
- **日志轮转**：自动日志文件管理
- **错误追踪**：详细的错误堆栈信息

### 数据备份
- **数据库备份**：定期备份用户数据
- **配置备份**：系统配置和渠道信息
- **日志归档**：历史日志数据管理

## 扩展开发

### 添加新的AI提供商
1. 在 `constant/` 中定义新的渠道类型
2. 在 `relay/channel/` 中实现适配器
3. 在相应的handler中添加处理逻辑
4. 更新前端界面支持

### 自定义功能
- **中间件扩展**：添加自定义的请求处理逻辑
- **计费规则**：实现自定义的计费算法
- **用户界面**：定制化前端界面
- **API扩展**：添加新的API端点

## 社区与支持

### 官方资源
- **项目地址**：https://github.com/Calcium-Ion/new-api
- **官方文档**：https://docs.newapi.pro/
- **Docker镜像**：calciumion/new-api:latest

### 相关项目
- **One API**：原版项目基础
- **Midjourney-Proxy**：Midjourney接口支持
- **neko-api-key-tool**：密钥查询工具
- **new-api-horizon**：高性能优化版

---

*本文档由系统自动生成，最后更新时间：2025-09-27*
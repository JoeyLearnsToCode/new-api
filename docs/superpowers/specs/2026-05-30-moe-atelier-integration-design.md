# moe-atelier 前端集成设计

## 概述

在 new-api 的同一个端口（默认 3000）上，根据 `Host` 请求头做路由分发：
- Host 以 `imagen.` 开头 → 服务 moe-atelier 前端 SPA（PWA）
- 其他 Host → 现有 new-api 逻辑不变

## 架构

```
Client (imagen.example.com) ──→ new-api (端口 3000)
                                    │
                              ┌─────┴─────┐
                              │ Host Check  │（MoeAtelierMiddleware）
                              └─────┬─────┘
                                  │              │
                        imagen.*  Host          其他 Host
                                  │              │
                                  ▼              ▼
                        moe-atelier SPA     现有 new-api 路由
                        (go:embed 嵌入)     (/api/* /v1/* /mj/* ...)
```

实现方式：一个**前置 Gin 中间件**，在所有路由之前注册。匹配到 `imagen.` 前缀的 Host 就 serve moe-atelier SPA 并 `c.Abort()`，否则 `c.Next()` 走正常流程。

## 改动详情

### 1. moe-atelier 侧修改

moe-atelier 作为独立仓库在 CI 中检出，通过 patch 方式修改后构建。

#### 1a. PWA 支持

在 `vite.config.ts` 添加 `vite-plugin-pwa`：

```ts
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,webp}'],
      },
      manifest: {
        name: '萌图工坊',
        short_name: '萌图工坊',
        description: 'AI 图片生成工具',
        theme_color: '#fff0f3',
        icons: [
          { src: 'pwa-192x192.png', sizes: '192x192', type: 'image/png' },
          { src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png' },
        ],
      },
    }),
  ],
})
```

需要添加 PWA 图标文件（`public/pwa-192x192.png`、`public/pwa-512x512.png`）。

#### 1b. 移除后端模式

后端模式依赖 Express 服务器，嵌入后不再可用。移除相关代码：

| 文件 | 改动 |
|------|------|
| `src/components/ConfigDrawer.tsx` | 删除"后端模式"开关 UI（约 lines 472-536） |
| `src/App.tsx` | 删除 `backendMode` 状态、SSE 连接、auth 流程相关代码 |
| `src/utils/backendApi.ts` | 删除整个文件（不再被引用） |
| `src/utils/imageDb.ts` | 保留（本地图片缓存仍需使用） |
| `src/components/ImageTask.tsx` | 删除 `if (backendMode)` 分支，删除 `/api/save-image` 调用 |
| `src/components/imageTaskState.ts` | 保留（本地任务状态管理） |

#### 1c. 移除 `/api/save-image` 调用

在 `ImageTask.tsx` 中注释或删除 `fetch('/api/save-image', ...)` 调用。图片已通过 IndexedDB 在浏览器端缓存，保存到服务器磁盘的功能不需要。

### 2. new-api 侧修改

#### 2a. `main.go`

新增 `go:embed` 指令嵌入 moe-atelier 构建产物：

```go
//go:embed web/moe-atelier-dist
var moeAtelierFS embed.FS

//go:embed web/moe-atelier-dist/index.html
var moeAtelierIndexPage []byte
```

在 `router.SetRouter()` 调用前先注册 Host 路由中间件。

#### 2b. `middleware/host_router.go`（新文件）

Gin 中间件，检查 `c.Request.Host`：

```go
func MoeAtelierMiddleware(moeFS embed.FS, moeIndexHTML []byte) gin.HandlerFunc {
    subFS, _ := fs.Sub(moeFS, "web/moe-atelier-dist")
    fileServer := http.FileServer(http.FS(subFS))

    return func(c *gin.Context) {
        if !strings.HasPrefix(strings.ToLower(c.Request.Host), "imagen.") {
            c.Next()
            return
        }

        path := c.Request.URL.Path
        if f, err := subFS.Open(strings.TrimPrefix(path, "/")); err == nil {
            f.Close()
            fileServer.ServeHTTP(c.Writer, c.Request)
            c.Abort()
            return
        }

        c.Data(http.StatusOK, "text/html; charset=utf-8", moeIndexHTML)
        c.Abort()
    }
}
```

#### 2c. `router/main.go`

```go
func SetRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte, moeFS embed.FS, moeIndexHTML []byte) {
    // MUST be first - intercepts imagen.* requests
    router.Use(middleware.MoeAtelierMiddleware(moeFS, moeIndexHTML))

    SetApiRouter(router)
    SetDashboardRouter(router)
    SetRelayRouter(router)
    SetVideoRouter(router)
    SetWebRouter(router, buildFS, indexPage)
}
```

#### 2d. 构建流程

新增构建脚本 `scripts/build-moe-atelier.sh` / `scripts/build-moe-atelier.ps1`：

```
1. 克隆 moe-atelier 仓库 (如果本地没有)
2. 应用 patches（移除后端模式，添加 PWA）
3. npm install
4. npm run build
5. 复制 dist/ → new-api/web/moe-atelier-dist/
```

#### 2e. Makefile 变更

新增 target：

```makefile
build-moe-frontend:
    @echo "Building moe-atelier frontend..."
    @pwsh scripts/build-moe-atelier.ps1

build-all: build-frontend build-moe-frontend
    @echo "Building Go backend..."
    @cd $(BACKEND_DIR) && go build -o new-api
```

#### 2f. GitHub Actions 变更

所有 release workflow（linux/windows/macos/freebsd/docker）需添加步骤：

```yaml
- name: Checkout moe-atelier
  uses: actions/checkout@v4
  with:
    repository: JoeyLearnsToCode/moe-atelier
    path: moe-atelier-tmp

- name: Build moe-atelier frontend
  run: |
    cd moe-atelier-tmp
    npm install
    npm run build
    mkdir -p ../web/moe-atelier-dist
    cp -r dist/* ../web/moe-atelier-dist/
```

### 3. Patch 文件管理

moe-atelier 的修改（移除后端模式 + PWA 支持）以 patch 文件形式存放在 `patches/moe-atelier/` 目录下。

`build-moe-atelier.sh` 流程：
```
git clone <moe-atelier-url>
cd moe-atelier
git am ../patches/moe-atelier/*.patch   # 或 git apply 逐个应用
npm install
npm run build
```

### 4. PWA 配置细节

| 配置项 | 值 |
|--------|-----|
| registerType | `autoUpdate` |
| workbox strategy | `CacheFirst` for static assets (JS/CSS) |
| runtimeCaching | 不需要（无外部 API 代理） |
| manifest.name | 萌图工坊 |
| manifest.theme_color | `#fff0f3` |
| icons | 192x192 + 512x512 PNG |

## 边界情况

1. **imagen. 紧跟在端口号后**：Go 的 `c.Request.Host` 包含端口（如 `imagen.example.com:3000`），`strings.HasPrefix` 仍然匹配。
2. **子域名层级**：`imagen.foo.bar.com` 也会匹配。
3. **index.html 缓存**：PWA service worker 会缓存 app shell，版本更新通过 `autoUpdate` 自动检测。
4. **开发环境**：本地测试时可用 `curl -H "Host: imagen.test.local" http://localhost:3000`。
5. **moe-atelier 与 new-api 前端路径冲突**：moe-atelier 的 `/assets/*` 与 new-api 的 `/assets` 路径无冲突（不同目录），因为中间件会先拦截 `imagen.` Host。

## 不做的事

- 不嵌入或代理 moe-atelier 的 Express 后端（`server.mjs`）
- 不修改 new-api 现有的 AI relay 逻辑
- 不修改 new-api 前端（`web/` 目录）
- 不添加新的 Go 依赖

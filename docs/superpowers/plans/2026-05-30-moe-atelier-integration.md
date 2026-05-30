# moe-atelier 前端集成实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 new-api 同一端口上，Host 以 `imagen.` 开头的请求转发到嵌入的 moe-atelier PWA 前端

**Architecture:** 前置 Gin 中间件检查 `c.Request.Host` → 匹配 `imagen.` 前缀则 serve 嵌入的 moe-atelier SPA 并 `c.Abort()`，否则走现有路由。moe-atelier 作为独立仓库在 CI 中检出构建，通过 Node.js 转换脚本移除后端模式代码并添加 PWA 支持。

**Tech Stack:** Go 1.23+ (gin), moe-atelier (React + Vite), vite-plugin-pwa

---

### Task 1: moe-atelier — PWA 支持

**Files:**
- Modify: `moe-atelier/package.json` (dependencies)
- Modify: `moe-atelier/vite.config.ts`
- Create: `moe-atelier/public/pwa-192x192.png`
- Create: `moe-atelier/public/pwa-512x512.png`

- [ ] **Step 1: 添加 vite-plugin-pwa 依赖**

编辑 `package.json`，在 `devDependencies` 中添加：

```json
"vite-plugin-pwa": "^0.20.0"
```

- [ ] **Step 2: 配置 vite.config.ts**

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
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
          { src: 'vite.svg', sizes: '192x192', type: 'image/svg+xml' },
          { src: 'vite.svg', sizes: '512x512', type: 'image/svg+xml' },
        ],
      },
    }),
  ],
  server: {
    host: process.env.VITE_HOST || '127.0.0.1',
  },
})
```

- [ ] **Step 3: 验证 PWA 构建**

PWA manifest 使用 `vite.svg` 作为图标，无需额外创建 PNG 文件。

```bash
cd moe-atelier
npm install
npm run build
# 验证 dist/ 中生成了 sw.js 和 workbox-*.js
ls dist/sw.js dist/manifest.webmanifest
```

Expected: `dist/sw.js`、`dist/manifest.webmanifest` 文件存在。

---

### Task 2: moe-atelier — 移除后端模式 + 保存图片到服务器功能

**Files:**
- Modify: `moe-atelier/src/App.tsx`
- Modify: `moe-atelier/src/components/ConfigDrawer.tsx`
- Modify: `moe-atelier/src/components/ImageTask.tsx`
- Delete: `moe-atelier/src/utils/backendApi.ts`

- [ ] **Step 1: 删除 backendApi.ts**

删除 `src/utils/backendApi.ts` 整个文件（约 242 行）。所有后端模式 API 调用不再存在。

- [ ] **Step 2: 修改 App.tsx — 删除后端模式相关代码**

替换 App.tsx 的 import 部分（移除 backendApi 导入，保留 storage、inputSync、apiUrl）：

```tsx
import * as React from 'react';
import { useState, useCallback, useRef } from 'react';
import { Layout, Button, Form, Row, Col, Typography, Space, ConfigProvider, message, Tooltip } from 'antd';
import {
  PlusOutlined,
  SettingFilled,
  ThunderboltFilled,
  CheckCircleFilled,
  HeartFilled,
  AppstoreFilled,
  DeleteFilled,
  RocketFilled,
  HourglassFilled,
  DashboardFilled,
  TrophyFilled,
} from '@ant-design/icons';
import { v4 as uuidv4 } from 'uuid';
import PromptDrawer from './components/PromptDrawer';
import CollectionBox from './components/CollectionBox';
import TaskGrid from './components/TaskGrid';
import ConfigDrawer from './components/ConfigDrawer';
import type { AppConfig, TaskConfig } from './types/app';
import type { CollectionItem } from './types/collection';
import type { GlobalStats } from './types/stats';
import type { PersistedUploadImage } from './types/imageTask';
import {
  cleanupTaskCache,
  cleanupUnusedImageCache,
  collectTaskImageKeys,
  deleteImageCache,
  loadCollectionItems,
  loadConfig,
  loadGlobalStats,
  loadTasks,
  saveConfig,
  saveCollectionItems,
  STORAGE_KEYS,
} from './app/storage';
import { useDebouncedSync, useInputGuard } from './utils/inputSync';
import {
  type ApiFormat,
  extractVertexProjectId,
  inferApiVersionFromUrl,
  normalizeApiBase,
  resolveApiUrl,
  resolveApiVersion,
} from './utils/apiUrl';
import { safeStorageSet } from './utils/storage';
import { calculateSuccessRate, formatDuration } from './utils/stats';
import { TASK_STATE_VERSION, saveTaskState, DEFAULT_TASK_STATS } from './components/imageTaskState';
```

删掉所有 `backendApi` 相关的导入行（lines 56-71）。

- [ ] **Step 3: 修改 App.tsx — 移除 backendMode 状态和所有相关逻辑**

在 `function App()` 内部：

1. 删除 `initialBackendMode` 的计算和 backendMode 状态声明
2. 删除所有 `useEffect` 中 `if (backendMode)` 分支
3. 删除 `bootstrapBackendState`、`applyBackendState`、SSE 连接、auth 流程
4. 简化 `fetchModels` — 移除 model list 缓存相关逻辑（keep it simple）
5. 简化 `handleAddTask`、`handleRemoveTask` — 移除 backendMode 分支
6. 简化 `handleConfigChange` — 移除 `markConfigDirty()`、`backendFormatConfigsRef` 相关
7. 简化 `handleCollect` — 移除 `backendMode` 参数
8. 简化所有收藏相关方法 — 移除 `backendMode` 分支

`App.tsx` 的 state 声明简化为：

```tsx
function App() {
  const [config, setConfig] = useState<AppConfig>(() => loadConfig());
  const [tasks, setTasks] = useState<TaskConfig[]>(() => loadTasks());
  const [globalStats, setGlobalStats] = useState<GlobalStats>(() => loadGlobalStats());
  const [configVisible, setConfigVisible] = useState(false);
  const [collectionVisible, setCollectionVisible] = useState(false);
  const [collectedItems, setCollectedItems] = useState<CollectionItem[]>(() => loadCollectionItems());
  const [collectionRevision, setCollectionRevision] = useState(0);
  const [promptDrawerVisible, setPromptDrawerVisible] = useState(false);
  const [models, setModels] = useState<{ label: string; value: string }[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [form] = Form.useForm();
```

删除 `ConfigDrawer` 中传递的所有 backendMode 相关 props，改为：

```tsx
<ConfigDrawer
  visible={configVisible}
  config={config}
  form={form}
  onClose={() => setConfigVisible(false)}
  onConfigChange={handleConfigChange}
  models={models}
  loadingModels={loadingModels}
  fetchModels={fetchModels}
/>
```

以及 `TaskGrid` prop 中去掉 `backendMode`。

以及 `CollectionBox` prop 中去掉 `backendMode`。

- [ ] **Step 4: 修改 ConfigDrawer.tsx — 移除后端模式 UI**

删除 `ConfigDrawerProps` 接口中 backend 相关属性：

```tsx
interface ConfigDrawerProps {
  visible: boolean;
  config: AppConfig;
  form: FormInstance<AppConfig>;
  onClose: () => void;
  onConfigChange: (changedValues: Partial<AppConfig>, values: AppConfig) => void;
  models: { label: string; value: string }[];
  loadingModels: boolean;
  fetchModels: () => void;
}
```

删除 JSX 中"后端模式"开关部分（约 lines 472-536），即包含 `后端模式` label 和 `.env` 说明的整个 div。

删除函数体中解构的 backend prop，只保留使用到的。

- [ ] **Step 5: 修改 ImageTask.tsx — 移除 backendMode 分支和 /api/save-image 调用**

在 `ImageTask.tsx` 中：

1. 删除 import 中所有 `backendApi` 的引用（`generateBackendTask`、`patchBackendTask` 等）
2. 在 `handleGenerate` 函数中，删除 `if (backendMode)` 分支，只保留 `else` 分支（local mode 逻辑）
3. 删除所有 `if (backendMode)` / `if (!backendMode)` 检查
4. 删除或注释掉调用 `fetch('/api/save-image', ...)` 的代码行（约 line 1547）
5. 删除 `performRequest()` 开头的 `if (backendMode) return;`
6. 将 `backendMode` prop 从所有内部组件调用中移除

- [ ] **Step 6: 验证构建**

```bash
cd moe-atelier
npm install
npm run build
# 构建应该成功，无 TS 错误
```

Expected: `npm run build` 通过，`dist/` 产出包含 SPA 文件 + PWA service worker。

---

### Task 3: new-api — Go 中间件 + go:embed

**Files:**
- Create: `middleware/host_router.go`
- Modify: `main.go`
- Modify: `router/main.go`
- Modify: `router/web-router.go`

- [ ] **Step 1: 创建 middleware/host_router.go**

```go
package middleware

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func MoeAtelierMiddleware(moeFS embed.FS) gin.HandlerFunc {
	subFS, err := fs.Sub(moeFS, "web/moe-atelier-dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(subFS))
	indexBytes, err := fs.ReadFile(subFS, "index.html")
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		if !strings.HasPrefix(strings.ToLower(c.Request.Host), "imagen.") {
			c.Next()
			return
		}

		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path != "" {
			f, err := subFS.Open(path)
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				c.Abort()
				return
			}
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
		c.Abort()
	}
}
```

- [ ] **Step 2: 修改 main.go — 添加 go:embed**

在 `main.go` 中添加新的 embed 指令（放在现有 embed 指令旁）：

```go
//go:embed web/moe-atelier-dist
var moeAtelierFS embed.FS
```

在 `router.SetRouter` 调用前注册中间件：

```go
server.Use(middleware.MoeAtelierMiddleware(moeAtelierFS))
router.SetRouter(server, buildFS, indexPage)
```

最终修改后的 `main.go` 相关部分：

```go
//go:embed web/dist
var buildFS embed.FS

//go:embed web/dist/index.html
var indexPage []byte

//go:embed web/moe-atelier-dist
var moeAtelierFS embed.FS

func main() {
    // ... (init code unchanged) ...

    server := gin.New()
    // ... (recovery, request-id, logger, session unchanged) ...

    // Host-based routing for moe-atelier — MUST be before other routes
    server.Use(middleware.MoeAtelierMiddleware(moeAtelierFS))
    router.SetRouter(server, buildFS, indexPage)

    // ... (port and run unchanged) ...
}
```

注意确保 import 中包含了 `"one-api/middleware"`（已存在）。

- [ ] **Step 3: 验证 Go 编译**

```bash
cd D:\Code\open-source\new-api
# 先创建空的 web/moe-atelier-dist 目录（否则 go:embed 报错）
mkdir web/moe-atelier-dist
# 创建占位 index.html
echo "<html><body>placeholder</body></html>" > web/moe-atelier-dist/index.html
# 编译
go build -o new-api.exe
```

Expected: 编译成功，生成 `new-api.exe`。

- [ ] **Step 4: 验证路由工作**

```bash
# 启动服务器（后台）
Start-Process -NoNewWindow pwsh -ArgumentList "-C", "cd D:\Code\open-source\new-api && .\new-api.exe"

# 测试默认路由（Host 不匹配 imagen.）
curl -s http://localhost:3000/api/

# 测试 imagen. 路由
curl -s -H "Host: imagen.example.com" http://localhost:3000/
# 应该返回 moe-atelier 的 index.html

# 测试静态文件
curl -s -H "Host: imagen.example.com" http://localhost:3000/index.html
```

---

### Task 4: new-api — 构建脚本

**Files:**
- Create: `scripts/build-moe-atelier.ps1`
- Modify: `Makefile`

- [ ] **Step 1: 创建 scripts/build-moe-atelier.ps1**

```powershell
param(
    [string]$MoeAtelierRepo = "https://github.com/JoeyLearnsToCode/moe-atelier.git",
    [string]$MoeAtelierDir = "moe-atelier-tmp",
    [string]$OutputDir = "web/moe-atelier-dist"
)

# Clean up previous build
if (Test-Path $MoeAtelierDir) {
    Remove-Item -Recurse -Force $MoeAtelierDir
}
if (Test-Path $OutputDir) {
    Remove-Item -Recurse -Force $OutputDir
}

# Clone moe-atelier
Write-Host "Cloning moe-atelier..." -ForegroundColor Green
git clone --depth 1 $MoeAtelierRepo $MoeAtelierDir

# Install dependencies
Write-Host "Installing dependencies..." -ForegroundColor Green
Push-Location $MoeAtelierDir
npm install

# Apply transformations: add vite-plugin-pwa
npm install --save-dev vite-plugin-pwa@^0.20.0

Write-Host "Applying PWA config to vite.config.ts..." -ForegroundColor Green
$viteConfig = @"
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
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
          { src: 'vite.svg', sizes: '192x192', type: 'image/svg+xml' },
          { src: 'vite.svg', sizes: '512x512', type: 'image/svg+xml' },
        ],
      },
    }),
  ],
  server: {
    host: process.env.VITE_HOST || '127.0.0.1',
  },
})
"@
Set-Content -Path "vite.config.ts" -Value $viteConfig -NoNewline

# Remove backend mode: delete backendApi.ts and fix imports
Write-Host "Removing backend API module..." -ForegroundColor Green
Write-Host "Patching ImageTask.tsx..." -ForegroundColor Green
$imageTaskPath = "src/components/ImageTask.tsx"
$content = Get-Content $imageTaskPath -Raw
$content = $content -replace "import.*backendApi.*;`n", ""
$content = $content -replace "import.*generateBackendTask.*from.*;`n", ""
Set-Content -Path $imageTaskPath -Value $content

# Remove backendApi.ts if it exists
Remove-Item -Force "src/utils/backendApi.ts" -ErrorAction SilentlyContinue

# Build
Write-Host "Building moe-atelier frontend..." -ForegroundColor Green
npm run build
Pop-Location

# Copy dist to output
Write-Host "Copying dist to $OutputDir..." -ForegroundColor Green
Copy-Item -Recurse "$MoeAtelierDir/dist" -Destination $OutputDir

# Cleanup
Remove-Item -Recurse -Force $MoeAtelierDir

Write-Host "Done! moe-atelier frontend built at $OutputDir" -ForegroundColor Green
```

- [ ] **Step 2: 更新 Makefile**

```makefile
FRONTEND_DIR = ./web
BACKEND_DIR = .
MOE_ATELIER_REPO = https://github.com/JoeyLearnsToCode/moe-atelier.git

.PHONY: all build-frontend build-moe-frontend start-backend build-all

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

build-moe-frontend:
	@echo "Building moe-atelier frontend..."
	@pwsh scripts/build-moe-atelier.ps1

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

build-all: build-frontend build-moe-frontend
	@echo "Building Go backend..."
	@cd $(BACKEND_DIR) && go build -ldflags "-s -w -X 'one-api/common.Version=$(cat VERSION)'" -o new-api
```

- [ ] **Step 3: 验证构建脚本**

```bash
cd D:\Code\open-source\new-api
pwsh scripts/build-moe-atelier.ps1
ls web/moe-atelier-dist/
```

Expected: `web/moe-atelier-dist/` 目录包含 index.html、assets/ 等文件。

---

### Task 5: GitHub Actions CI 变更

**Files:**
- Modify: `.github/workflows/linux-release.yml`
- Modify: `.github/workflows/windows-release.yml`
- Modify: `.github/workflows/macos-release.yml`
- Modify: `.github/workflows/freebsd-release.yml`
- Modify: `.github/workflows/docker-image-arm64.yml`
- Modify: `.github/workflows/docker-image-alpha.yml`

- [ ] **Step 1: 修改 linux-release.yml**

在所有构建步骤之前，添加 moe-atelier 检出和构建步骤：

```yaml
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Checkout and build moe-atelier frontend
        run: |
          git clone --depth 1 https://github.com/JoeyLearnsToCode/moe-atelier.git moe-atelier-tmp
          cd moe-atelier-tmp
          npm install
          npm install --save-dev vite-plugin-pwa@^0.20.0
          # Write PWA vite.config.ts
          cat > vite.config.ts << 'VITEEOF'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: { globPatterns: ['**/*.{js,css,html,ico,png,svg,webp}'] },
      manifest: {
        name: '萌图工坊',
        short_name: '萌图工坊',
        description: 'AI 图片生成工具',
        theme_color: '#fff0f3',
        icons: [
          { src: 'vite.svg', sizes: '192x192', type: 'image/svg+xml' },
          { src: 'vite.svg', sizes: '512x512', type: 'image/svg+xml' },
        ],
      },
    }),
  ],
  server: { host: process.env.VITE_HOST || '127.0.0.1' },
})
VITEEOF
          # Apply patches: remove backend mode
          rm -f src/utils/backendApi.ts
          npm run build
          mkdir -p ../web/moe-atelier-dist
          cp -r dist/* ../web/moe-atelier-dist/
          cd ..
          rm -rf moe-atelier-tmp
```

在 "Build Frontend" 步骤之前插入以上步骤。

- [ ] **Step 2: 修改 windows-release.yml**

同样在 "Build Frontend" 之前添加相同的 moe-atelier 构建步骤（Node.js setup + moe-atelier checkout & build）。注意 Windows 环境下使用 `mkdir` 和 `cp` 兼容命令。

- [ ] **Step 3: 修改 macos-release.yml**

与 linux-release.yml 相同的修改。

- [ ] **Step 4: 修改 freebsd-release.yml**

相同的修改（注意 FreeBSD 的 shell 兼容性）。

- [ ] **Step 5: 修改 docker-image-arm64.yml**

与 linux-release.yml 相同的修改。

- [ ] **Step 6: 修改 docker-image-alpha.yml**

Docker 构建不直接修改 workflow，而是修改 `Dockerfile`。在 Dockerfile 的构建阶段添加 moe-atelier 构建步骤：

```dockerfile
# 在现有的前端构建阶段之后，添加 moe-atelier 构建
FROM node:20-alpine AS moe-atelier-builder
WORKDIR /app
RUN apk add --no-cache git
RUN git clone --depth 1 https://github.com/JoeyLearnsToCode/moe-atelier.git .
RUN npm install && \
    npm install --save-dev vite-plugin-pwa@^0.20.0 && \
    # Write PWA config
    printf 'import { defineConfig } from "vite"\nimport react from "@vitejs/plugin-react"\nimport { VitePWA } from "vite-plugin-pwa"\nexport default defineConfig({ plugins: [react(), VitePWA({ registerType: "autoUpdate", workbox: { globPatterns: ["**/*.{js,css,html,ico,png,svg,webp}"] }, manifest: { name: "萌图工坊", short_name: "萌图工坊", description: "AI 图片生成工具", theme_color: "#fff0f3", icons: [{ src: "pwa-192x192.png", sizes: "192x192", type: "image/png" }, { src: "pwa-512x512.png", sizes: "512x512", type: "image/png" }] } })], server: { host: process.env.VITE_HOST || "127.0.0.1" } })\n' > vite.config.ts && \
    rm -f src/utils/backendApi.ts && \
    npm run build

# 在 Go 构建阶段，复制 moe-atelier dist
COPY --from=moe-atelier-builder /app/dist ./web/moe-atelier-dist
```

- [ ] **Step 7: 验证 CI 流程（可选）**

Push 到一个测试分支，触发 workflow_dispatch，检查构建产物中是否包含 `web/moe-atelier-dist/` 目录。

---

### 验证清单

- [ ] `npm run build` 在 moe-atelier 中成功，产出 `dist/sw.js`
- [ ] `go build` 在 new-api 中成功
- [ ] `curl -H "Host: imagen.example.com" http://localhost:3000/` 返回 moe-atelier 的 index.html
- [ ] `curl -H "Host: api.example.com" http://localhost:3000/api/` 正常返回 new-api 响应
- [ ] moe-atelier 前端在浏览器中可注册 service worker
- [ ] CI workflow 在 GitHub Actions 中通过

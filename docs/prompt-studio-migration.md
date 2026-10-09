# Prompt Studio 嵌入 new-api 迁移改造记录

## 追踪的上游版本

| 项 | 值 |
| --- | --- |
| 上游仓库 | `D:\nodejs-dev\prompt-studio` |
| **追踪 commit** | `63b63b4af997ff58fd959514cadcf18ec40a16e8` |
| commit 时间 | 2026-05-08 17:15:07 +0800 |
| commit 标题 | `perf: better fuzzy search` |
| 上游 `src/` 文件数 | 77（约 10669 行） |
| 迁入后文件数 | 68（约 10100 行，见下表"未迁移文件"） |
| 迁移完成日期 | 2026-10-08 |

> 上游当时存在未提交改动：`AGENTS.md`（新增）、`pnpm-workspace.yaml`（未跟踪）。二者与运行时无关，未纳入。
>
> **后续增量迁移时，以本文件头部记录的 commit 为基线做 `git diff <commit>..HEAD -- src`，只处理 diff 涉及的文件。**

---

## 1. 目标与约束

- 将上游 React 应用作为 new-api 控制台内的一个页面原样嵌入，UI 与交互保持不变
- **数据不再只存浏览器**：全部读写 new-api 的 Go/GORM 存储，按 `user_id` 隔离
- 浏览器存储只用于两件事：UI 偏好（布局比例、编辑器字号、语言）与首屏列表快照缓存
- 必须有 new-api 登录态：路由用 `PrivateRoute` 包裹，接口全部挂在 `middleware.UserAuth()` 下
- 遵循仓库既有约定：三库兼容（SQLite / MySQL / PostgreSQL）、JSON 走 `common.Marshal`、响应体 `{success, message, data}`

## 2. 代码落地位置

### 后端

| 文件 | 内容 |
| --- | --- |
| `model/prompt_studio.go` | 5 个模型结构体 + `TableName()` + `normalize` / `sha256` 去重哈希 + ID 辅助方法 |
| `model/prompt_studio_folder.go` | 文件夹 CRUD |
| `model/prompt_studio_project.go` | 项目 CRUD（含创建时同事务建根版本、级联删除） |
| `model/prompt_studio_version.go` | 版本 CRUD（根版本唯一性、重复检测、"接骨"删除） |
| `model/prompt_studio_attachment.go` | 附件 CRUD（base64 内容、大小上限） |
| `model/prompt_studio_bulk.go` | 整包导出 / 导入 / 示例数据 |
| `controller/prompt_studio.go` | 公共响应与错误处理 |
| `controller/prompt_studio_{folder,project,version,attachment,bulk}.go` | 各资源接口 |
| `router/api-router.go` | `/api/prompt-studio` 路由组 |
| `model/main.go` | `migrateDB()` 的 `AutoMigrate` 列表新增 5 个模型 |

### 前端

| 路径 | 内容 |
| --- | --- |
| `web/src/prompt-studio/` | 上游 `src/` 整体搬入（组件、store、服务、i18n、模型、样式） |
| `web/src/prompt-studio/index.tsx` | 页面入口：hash 子路由 + `I18nProvider` + `AppInitializer` |
| `web/src/prompt-studio/api/client.ts` | **新增**：后端接口客户端，复用 new-api 的 `API` 实例 |
| `web/src/prompt-studio/utils/snapshot.ts` | **新增**：首屏列表快照缓存 |

接入点（宿主侧改动）：`web/src/App.jsx`、`web/src/components/layout/SiderBar.jsx`、
`web/src/components/layout/PageLayout.jsx`、`web/src/helpers/render.jsx`、
`web/src/hooks/common/useSidebar.js`、`web/tailwind.config.js`、`web/package.json`、i18n 两份 locale。

## 3. 存储层改造

### 3.1 表结构

5 张新表，均带 `user_id` 与索引；主键沿用前端生成的 UUID 字符串（保证导入导出时 ID 稳定）。

```
prompt_studio_folders     id, user_id, name, parent_id, created_at
prompt_studio_projects    id, user_id, folder_id, name, tags(TEXT), created_at, updated_at
prompt_studio_versions    id, user_id, project_id, parent_id, name,
                          content(TEXT), content_hash, score, notes, created_at, updated_at
prompt_studio_attachments id, user_id, version_id, file_name, file_type,
                          size, content(TEXT/base64), created_at
prompt_studio_snippets    id, user_id, name, content, created_at
```

已在 SQLite 上实测建表结果，列类型与设计一致（`tags` 为单个 TEXT 列，未被 GORM 拆成嵌入列）。

### 3.2 字段与语义约定

| 约定 | 说明 |
| --- | --- |
| 主键 | 字符串 UUID；新建时服务端 `common.GetUUID()` 生成，导入时沿用文件里的 ID |
| 时间 | 毫秒级 Unix 时间戳，与前端 `Date.now()` 对齐。**必须保留毫秒精度**，否则 `updatedAt` 并列会导致"取最新版本"取错 |
| 根节点 | 前端用 `null`，后端用空字符串 `''`。转换集中在 `api/client.ts` 的 `toRef`；**注意 `folderId: null` 在更新接口里表示"不更新"，移回根目录要显式发 `''`** |
| 主键生成 | 字符串主键没有自增能力，**必须显式生成**。已在 5 个模型上各加 `BeforeCreate` 兜底（`ID == ""` 时 `common.GetUUID()`），防止任何新建路径漏掉 |
| `content_hash` | 标准化内容的 SHA-256，见 3.3 |
| `normalizedContent` | 上游是运行时计算字段，明确不落库 |
| 项目 `tags` | 自定义 `Scanner`/`Valuer` 类型，序列化为 TEXT（跨三库安全） |
| 附件内容 | base64 存 TEXT。不写死 `type:text`（MySQL 只有 64KB），交给 GORM 默认映射：MySQL `longtext` / PG `text` / SQLite `text`。单文件上限 10MB（`model.PromptStudioMaxAttachmentSize`，前端 `MainView` 校验阈值需同步） |
| 根版本 | 新建项目时同事务创建一个 `content=''`、`content_hash=''` 的根版本，与上游行为一致 |

### 3.3 normalize + content_hash 移植（关键）

上游 `src/utils/normalize.ts` + `src/utils/hash.ts` 的逻辑必须逐字节对齐，否则重复检测失效。
Go 实现在 `model/prompt_studio.go`：`NormalizePromptContent()` / `ComputePromptContentHash()`。

规则：转小写 → 剔除空白与不可见字符 → 剔除指定标点 → `trim` → `sha256` 十六进制。

> **已踩坑**：JS 的 `\s` 包含普通空格 `U+0020`，而上游正则里显式列出的只有 `\u0000-\u001F`，
> 第一版 Go 实现漏掉了 `U+0020`，导致 `"Hello, World!"` 哈希不一致。已修正并用例比对通过。
>
> 校验方式：用上游真实 `normalize.ts` / `hash.ts` 源码在 Node 里算参考值，与 Go 结果逐一比对
> （含中文、`Hello, World!`、空串、混合标点与制表符共 5 个用例）。

## 4. 接口清单

路由组 `/api/prompt-studio`，统一 `middleware.UserAuth()`；所有查询带 `user_id` 条件，
`user_id` 由 `c.GetInt("id")` 注入，不接受请求体传入。

```
GET    /folders                        POST   /folders
PUT    /folders/:id                    DELETE /folders/:id          (子项上提一级)
GET    /projects?folderId=             POST   /projects             (同事务建根版本)
PUT    /projects/:id                   DELETE /projects/:id         (级联删版本+附件)
GET    /versions?projectId=            POST   /versions             (重复时 message = DUPLICATE_DETECTED:<id>)
PUT    /versions/:id                   DELETE /versions/:id         (子版本接骨 + 删附件)
GET    /attachments?versionId=         DELETE /attachments/:id
POST   /versions/:id/attachments       (multipart，一次可多文件)
GET    /attachments/:id/content        (?download=1 触发下载)
GET    /export                         POST   /import               (mode: merge | overwrite)
POST   /sample                         (文案由前端传，缺省回落中文)
```

- 业务错误沿用项目约定：HTTP 200 + `success:false`，不做 4xx 语义化
- `projectId` 为空时 `GET /versions` 返回当前用户全部版本，供全局搜索使用
- 导入导出是**整包一次性请求**，前端不做逐条循环（附件也一次多文件提交）

## 5. 前端改造点

| 文件 | 改造 |
| --- | --- |
| `db/schema.ts` 等 Dexie 相关 | **整体删除**，新增 `api/client.ts` 复用 new-api 的 `API` 实例（自带 `New-Api-User` 头与会话 cookie） |
| `store/projectStore.ts` | Dexie → HTTP；删除未被引用的 `getRecentProjects`；`expandFolderPathToProject` 改为在内存文件夹树上回溯 |
| `store/versionStore.ts` | Dexie → HTTP；新增 `loadAllVersions()` 供全局搜索；删除未被引用的 `getVersion/getChildren/isLeafNode/getLatestVersion/checkDuplicate` |
| `services/attachmentManager.ts` | 上传改为后端接口；因附件内容需鉴权、不能直接用 `<img src>`，新增 `preloadPreviewUrls()` 预取 Object URL 并缓存，`getPreviewUrl()` 保持同步 |
| `services/exportService.ts` | 5 次 `db.X.toArray()` → 1 次 `GET /export`；附件由 base64 还原为二进制写回 ZIP 的 `attachments/` 目录，包结构与上游保持一致 |
| `services/importService.ts` | 逐表写入 → 1 次 `POST /import`；附件优先取随包 base64，缺失时回退读 `attachments/<id><ext>` |
| `services/initializeSampleData.ts` | 改为调用 `POST /sample`（服务端单事务创建），文案按当前语言传入 |
| `components/AppInitializer.tsx` | 先加载再判断是否建示例数据；主题改为跟随 new-api 全局主题（不再自行操作 `documentElement` 的 dark class） |
| `hooks/useGlobalSearch.ts` | `db.versions.toArray()` → `loadAllVersions()` |
| `components/common/ThemeToggle.tsx` | 从改本地 store 改为调用 new-api 的 `useSetTheme()`，切换全局主题 |
| `pages/Settings.tsx` | 移除 WebDAV 备份/恢复卡片（连带不再需要 `mt-8` / `sm:flex-none` / `text-[18px]` / `border-primary/30` 等类）；`navigate('/')` 改为设置 hash |
| `pages/MainView.tsx` | `navigate('/settings')` 改为设置 hash；附件上传改为一次批量请求；根容器 `h-dynamic-screen` → 由 `.prompt-studio-root` 提供高度；logo 改为 `import` 随包资源 |
| `components/version/CompareModal.tsx` | `import { editor }` → `import type`（否则 monaco 会被打进产物，约 7MB） |
| `components/version/AttachmentGallery.tsx` | 增加预览 URL 预取与重渲染；下载改为传附件对象以保留原始文件名 |
| `services/webdavService.ts` 等 | 见下方"未迁移文件" |

## 6. 路由与菜单接入

- 宿主路由 `/prompt-studio`，`PrivateRoute` 包裹，`React.lazy` 按需加载（Monaco 体积大）
- **整页独立渲染，不套控制台外壳**：`PageLayout.jsx` 在命中 `/prompt-studio` 时直接 `return <App />`，
  跳过 `Header` / `Sider` / `Footer`。因此不需要（也不要）把该路径加进 `isConsoleRoute`
- 页面高度由 `.prompt-studio-root` 自带（`height: 100vh / 100dvh`），不依赖宿主 Layout 的高度
- 页面内部沿用上游 hash 路由：`#/`、`#/project/:id`、`#/settings`，由 `index.tsx` 的 `useHashRoute` 驱动
- 侧边栏新菜单项生效需要 4 处配合：`SiderBar.jsx` 的 `routerMap`、`workspaceItems`、
  `useSidebar.js` 的 `DEFAULT_ADMIN_CONFIG.console.prompt_studio = true`（**缺了会被静默过滤**）、`render.jsx` 的图标
- 上游 logo 是 `/icon-192.svg`（在 `public/` 下），迁移时改为随包构建：
  放到 `web/src/prompt-studio/assets/icon-192.svg`，由 `MainView.tsx` `import logoUrl from '...'` 引用

## 7. Tailwind 色板（对宿主有全局影响，改动前请确认）

new-api 原 `tailwind.config.js` 用 `theme.colors` **整体替换**了默认色板，只保留 `semi-color-*`，
导致 `text-gray-500` / `bg-white` 一类默认色工具类根本不会生成（`web/dist` 产物中查不到）。
而 prompt-studio 大量依赖默认色板。

因此本次改为在默认色板之上合并 semi 与工坊令牌：

默认色板取自 `tailwindcss/colors`（不是 `tailwindcss/defaultTheme`，后者的 `colors` 是空对象）；
旧色名（`lightBlue` / `warmGray` / `trueGray` / `coolGray` / `blueGray`）是带弃用告警的 getter，
需用 `Object.getOwnPropertyNames()` 过滤后再取值。详见 `web/tailwind.config.js` 里的注释。

```js
colors: { ...defaultColors, ...semiColors }       // 恢复默认色板
extend: { colors: promptStudioColors }            // 合并工坊设计令牌
```

**副作用**：new-api 现存约 240 处原本失效的 `text-gray-*` / `text-white` 类会开始生效，
属于全局视觉变化。若后续不希望保留，可改为给 prompt-studio 的 class 加统一前缀隔离。

样式隔离：上游 `globals.css` 里的 `*` / `body` / `#root` 全局规则改为限定在 `.prompt-studio-root` 下，
并去掉重复的 `@tailwind base/components/utilities`；滚动条与 Monaco 样式同样加作用域前缀。
`Material Symbols` / `Noto Sans SC` 字体通过 `globals.css` 的 `@import` 引入（与上游一样走 CDN，离线环境图标会缺失）。

## 8. 有意偏离上游的地方

1. **移除 WebDAV 备份/恢复**：数据已在服务端持久化，且原实现把密码明文存 localStorage
2. **主题不再独立**：跟随 new-api 全局主题，页内开关切换的是全局主题
3. **附件单文件上限 10MB**：上游 IndexedDB 允许到 50MB，服务端存储改为 10MB（前端校验阈值已同步）
4. **未迁移的死代码**（上游本身未被引用）：`services/folderManager.ts`、`services/snippetManager.ts`、
   `utils/validation.ts`、`db/migrations.ts`、`pages/SnippetLibrary.tsx`、`src/test/`、`App.tsx`、`router.tsx`、`main.tsx`。
   `snippets` 表保留，仅为兼容导入导出
5. **duplicate 检测**：上游 `handleSave` 实际总是传 `skipDuplicateCheck=true`，该分支在 UI 上不会触发；
   服务端仍完整实现（返回 `DUPLICATE_DETECTED:<id>`），行为保持不变

## 9. 后续增量迁移指引

1. 在 `D:\nodejs-dev\prompt-studio` 执行 `git -P diff 63b63b4a..HEAD --stat -- src`，对比本文件记录的基线
2. 只处理 diff 涉及的文件；若 diff 触碰下列文件，需要走后端/接入逻辑，而不是直接覆盖：
   - `src/db/schema.ts`（已删除，任何表结构变化要同步 `model/prompt_studio*.go` + `migrateDB`）
   - `src/store/projectStore.ts`、`src/store/versionStore.ts`（已改为 HTTP）
   - `src/services/attachmentManager.ts`、`exportService.ts`、`importService.ts`、`initializeSampleData.ts`（已改为 HTTP）
   - `src/utils/normalize.ts`、`src/utils/hash.ts`（**变化必须同步 Go 版并重跑哈希比对**）
   - `src/components/AppInitializer.tsx`、`components/common/ThemeToggle.tsx`（已改为跟随宿主主题）
   - `src/pages/Settings.tsx`（已移除 WebDAV）
   - `src/styles/tokens.js`、`src/styles/globals.css`（需同步 `web/tailwind.config.js` 与作用域处理）
3. 其余纯 UI 文件（组件、hooks、canvas、diff、搜索、i18n）可直接覆盖同名文件，再跑一次下面的验证
4. 迁移完成后更新本文件头部的 commit 与日期

## 10. 验证方式

```bash
# 后端
go build ./...
# 前端
cd web && bun run build
```

已验证项：

- `go build ./...` 通过；19 条新路由注册无冲突（Gin 路由冲突会在注册时 panic，
  验证方式：临时在 `router` 包里 `SetApiRouter(gin.New())` 后枚举 `engine.Routes()` 比对）
- 5 张表在 SQLite 上 AutoMigrate 成功，列结构符合设计
- Go 与上游 `normalize` + `sha256` 在 5 个用例上哈希完全一致
- `bun run build` 通过；prompt-studio 代码落在独立懒加载 chunk，未污染主包
- 新增代码 `prettier --check` 通过；改动到的 `.jsx` 文件 `eslint` 通过

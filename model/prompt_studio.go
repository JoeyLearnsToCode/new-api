package model

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Prompt Studio 相关模型
// 对应前端 web/src/prompt-studio，原实现使用浏览器 IndexedDB(Dexie) 存储，
// 这里改为落在服务端数据库，并按 user_id 做用户隔离。
//
// 约定：
// - 主键沿用前端生成的 UUID 字符串，便于导入导出时保持 ID 稳定
// - 时间统一使用毫秒级 Unix 时间戳，与前端 Date.now() 保持一致
// - 前端用 null 表示"根节点/根目录"，这里用空字符串存储，由 controller 层做转换

// PromptStudioProjectTags 项目的可选标签
type PromptStudioProjectTags struct {
	Model    string `json:"model,omitempty"`
	Platform string `json:"platform,omitempty"`
	Type     string `json:"type,omitempty"`
}

// Scan 实现 sql.Scanner，从 TEXT 列还原标签
func (t *PromptStudioProjectTags) Scan(value any) error {
	*t = PromptStudioProjectTags{}
	if value == nil {
		return nil
	}
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return errors.New("invalid prompt studio project tags")
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return common.UnmarshalJsonStr(raw, t)
}

// Value 实现 driver.Valuer，将标签序列化为 TEXT
func (t PromptStudioProjectTags) Value() (driver.Value, error) {
	raw, err := common.Marshal(t)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

// PromptStudioFolder 文件夹
type PromptStudioFolder struct {
	ID        string `json:"id" gorm:"primaryKey;size:64"`
	UserID    int    `json:"user_id" gorm:"index:idx_ps_folder_user,priority:1;not null"`
	Name      string `json:"name" gorm:"size:255;not null"`
	ParentID  string `json:"parentId" gorm:"size:64;index:idx_ps_folder_parent"`
	CreatedAt int64  `json:"createdAt" gorm:"bigint;index:idx_ps_folder_user,priority:2"`
}

func (PromptStudioFolder) TableName() string {
	return "prompt_studio_folders"
}

// BeforeCreate 主键是字符串、没有自增能力，创建前兜底生成，
// 避免写入空 ID 后互相冲突（空 ID 在业务语义上还等同于"根目录"，会引发更多问题）
func (f *PromptStudioFolder) BeforeCreate(_ *gorm.DB) error {
	if f.ID == "" {
		f.ID = common.GetUUID()
	}
	return nil
}

func (f *PromptStudioFolder) GetID() string    { return f.ID }
func (f *PromptStudioFolder) SetUserID(id int) { f.UserID = id }

// PromptStudioProject 项目
type PromptStudioProject struct {
	ID        string                   `json:"id" gorm:"primaryKey;size:64"`
	UserID    int                      `json:"user_id" gorm:"index:idx_ps_project_user,priority:1;not null"`
	FolderID  string                   `json:"folderId" gorm:"size:64;index:idx_ps_project_folder"`
	Name      string                   `json:"name" gorm:"size:255;not null"`
	Tags      *PromptStudioProjectTags `json:"tags,omitempty" gorm:"type:text"`
	CreatedAt int64                    `json:"createdAt" gorm:"bigint"`
	UpdatedAt int64                    `json:"updatedAt" gorm:"bigint;index:idx_ps_project_user,priority:2"`
}

func (PromptStudioProject) TableName() string {
	return "prompt_studio_projects"
}

// BeforeCreate 见 PromptStudioFolder.BeforeCreate
func (p *PromptStudioProject) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = common.GetUUID()
	}
	return nil
}

func (p *PromptStudioProject) GetID() string    { return p.ID }
func (p *PromptStudioProject) SetUserID(id int) { p.UserID = id }

// PromptStudioVersion 提示词版本
// NormalizedContent 是前端的运行时计算字段，不落库
type PromptStudioVersion struct {
	ID          string `json:"id" gorm:"primaryKey;size:64"`
	UserID      int    `json:"user_id" gorm:"index:idx_ps_version_user;not null"`
	ProjectID   string `json:"projectId" gorm:"size:64;index:idx_ps_version_project;not null"`
	ParentID    string `json:"parentId" gorm:"size:64;index:idx_ps_version_parent"`
	Name        string `json:"name" gorm:"size:255;index:idx_ps_version_name"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash" gorm:"size:64;index:idx_ps_version_hash"`
	Score       int    `json:"score"`
	Notes       string `json:"notes"`
	CreatedAt   int64  `json:"createdAt" gorm:"bigint"`
	UpdatedAt   int64  `json:"updatedAt" gorm:"bigint;index:idx_ps_version_updated"`
}

func (PromptStudioVersion) TableName() string {
	return "prompt_studio_versions"
}

// BeforeCreate 见 PromptStudioFolder.BeforeCreate
func (v *PromptStudioVersion) BeforeCreate(_ *gorm.DB) error {
	if v.ID == "" {
		v.ID = common.GetUUID()
	}
	return nil
}

func (v *PromptStudioVersion) GetID() string    { return v.ID }
func (v *PromptStudioVersion) SetUserID(id int) { v.UserID = id }

// PromptStudioAttachment 版本附件，文件内容以 base64 存放在 Content 列
// 列类型不写死 TEXT：MySQL 下 GORM 对无 size 约束的 string 会生成 longtext，
// PostgreSQL / SQLite 为 text，三种数据库均可容纳较大的附件
type PromptStudioAttachment struct {
	ID        string `json:"id" gorm:"primaryKey;size:64"`
	UserID    int    `json:"user_id" gorm:"index:idx_ps_attachment_user;not null"`
	VersionID string `json:"versionId" gorm:"size:64;index:idx_ps_attachment_version;not null"`
	FileName  string `json:"fileName" gorm:"size:255"`
	FileType  string `json:"fileType" gorm:"size:128"`
	Size      int64  `json:"size" gorm:"bigint"`
	Content   string `json:"content,omitempty"`
	CreatedAt int64  `json:"createdAt" gorm:"bigint"`
}

func (PromptStudioAttachment) TableName() string {
	return "prompt_studio_attachments"
}

// BeforeCreate 见 PromptStudioFolder.BeforeCreate
func (a *PromptStudioAttachment) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = common.GetUUID()
	}
	return nil
}

func (a *PromptStudioAttachment) GetID() string    { return a.ID }
func (a *PromptStudioAttachment) SetUserID(id int) { a.UserID = id }

// PromptStudioSnippet 片段，仅用于导入导出兼容，当前前端无实际入口
type PromptStudioSnippet struct {
	ID        string `json:"id" gorm:"primaryKey;size:64"`
	UserID    int    `json:"user_id" gorm:"index:idx_ps_snippet_user;not null"`
	Name      string `json:"name" gorm:"size:255;index:idx_ps_snippet_name"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"createdAt" gorm:"bigint"`
}

func (PromptStudioSnippet) TableName() string {
	return "prompt_studio_snippets"
}

// BeforeCreate 见 PromptStudioFolder.BeforeCreate
func (s *PromptStudioSnippet) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = common.GetUUID()
	}
	return nil
}

func (s *PromptStudioSnippet) GetID() string    { return s.ID }
func (s *PromptStudioSnippet) SetUserID(id int) { s.UserID = id }

// promptStudioSpaceRune 判断是否为需要剔除的空白/不可见字符
// 与前端 src/utils/normalize.ts 的正则保持一致：
//
//	\s \u0000-\u001F \u007F-\u009F \u00A0 \u1680 \u180E \u2000-\u200B \u202F \u205F \u3000 \uFEFF
func promptStudioSpaceRune(r rune) bool {
	switch {
	case r <= 0x001F || r == 0x0020:
		return true
	case r >= 0x007F && r <= 0x009F:
		return true
	case r == 0x00A0 || r == 0x1680 || r == 0x180E:
		return true
	case r >= 0x2000 && r <= 0x200B:
		return true
	case r == 0x2028 || r == 0x2029 || r == 0x202F || r == 0x205F:
		return true
	case r == 0x3000 || r == 0xFEFF:
		return true
	}
	return false
}

// promptStudioPunctRune 判断是否为需要剔除的标点符号
// 对应前端正则 [,.\-\-，。/\\()\[\]{}、（）【】|@；;""]
func promptStudioPunctRune(r rune) bool {
	switch r {
	case ',', '.', '-', '/', '\\', '(', ')', '[', ']', '{', '}', '|', '@', ';', '"',
		'，', '。', '、', '（', '）', '【', '】', '；':
		return true
	}
	return false
}

// NormalizePromptContent 标准化提示词内容
// 必须与前端 src/utils/normalize.ts 完全一致，否则 content_hash 会与前端算出的不一致，
// 导致重复检测失效
func NormalizePromptContent(text string) string {
	lowered := strings.ToLower(text)
	var builder strings.Builder
	builder.Grow(len(lowered))
	for _, r := range lowered {
		if promptStudioSpaceRune(r) || promptStudioPunctRune(r) {
			continue
		}
		builder.WriteRune(r)
	}
	return strings.TrimSpace(builder.String())
}

// ComputePromptContentHash 计算提示词内容哈希（标准化内容的 SHA-256 十六进制）
func ComputePromptContentHash(content string) string {
	sum := sha256.Sum256([]byte(NormalizePromptContent(content)))
	return hex.EncodeToString(sum[:])
}

// promptStudioNow 返回毫秒级时间戳，与前端 Date.now() 对齐
// 必须保留毫秒精度：前端按 updatedAt 排序并取最新版本，秒级精度会产生并列导致取错
func promptStudioNow() int64 {
	return time.Now().UnixMilli()
}

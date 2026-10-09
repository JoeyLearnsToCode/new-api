package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Prompt Studio 的批量操作：导出、导入、示例数据
// 前端不做逐条循环，导入导出都是一次性请求

// PromptStudioExportData 整库导出结果
type PromptStudioExportData struct {
	Folders     []*PromptStudioFolder     `json:"folders"`
	Projects    []*PromptStudioProject    `json:"projects"`
	Versions    []*PromptStudioVersion    `json:"versions"`
	Snippets    []*PromptStudioSnippet    `json:"snippets"`
	Attachments []*PromptStudioAttachment `json:"attachments"`
}

// PromptStudioExportAll 导出当前用户的全部数据，附件的 content 为 base64 字符串
func PromptStudioExportAll(userId int) (*PromptStudioExportData, error) {
	data := &PromptStudioExportData{}
	if err := DB.Where("user_id = ?", userId).Order("created_at asc").Find(&data.Folders).Error; err != nil {
		return nil, err
	}
	if err := DB.Where("user_id = ?", userId).Order("updated_at desc").Find(&data.Projects).Error; err != nil {
		return nil, err
	}
	if err := DB.Where("user_id = ?", userId).Order("created_at asc").Find(&data.Versions).Error; err != nil {
		return nil, err
	}
	if err := DB.Where("user_id = ?", userId).Order("created_at asc").Find(&data.Snippets).Error; err != nil {
		return nil, err
	}
	if err := DB.Where("user_id = ?", userId).Order("created_at asc").Find(&data.Attachments).Error; err != nil {
		return nil, err
	}
	return data, nil
}

// PromptStudioImportMode 导入模式
type PromptStudioImportMode string

const (
	// PromptStudioImportModeMerge 合并：已存在的 ID 跳过
	PromptStudioImportModeMerge PromptStudioImportMode = "merge"
	// PromptStudioImportModeOverwrite 覆盖：先清空再全量写入
	PromptStudioImportModeOverwrite PromptStudioImportMode = "overwrite"
)

// PromptStudioImportRequest 整包导入请求
// 字段与前端导出的 JSON 结构一一对应
type PromptStudioImportRequest struct {
	Mode        string                    `json:"mode"`
	Folders     []*PromptStudioFolder     `json:"folders"`
	Projects    []*PromptStudioProject    `json:"projects"`
	Versions    []*PromptStudioVersion    `json:"versions"`
	Snippets    []*PromptStudioSnippet    `json:"snippets"`
	Attachments []*PromptStudioAttachment `json:"attachments"`
}

// PromptStudioImportResult 导入结果统计
type PromptStudioImportResult struct {
	Folders     int `json:"folders"`
	Projects    int `json:"projects"`
	Versions    int `json:"versions"`
	Snippets    int `json:"snippets"`
	Attachments int `json:"attachments"`
}

// PromptStudioImportAll 整包导入
// merge：按主键过滤已存在的记录后批量写入；overwrite：先清空当前用户数据再全量写入
func PromptStudioImportAll(userId int, req *PromptStudioImportRequest) (*PromptStudioImportResult, error) {
	overwrite := PromptStudioImportMode(req.Mode) == PromptStudioImportModeOverwrite
	if req.Mode == "" {
		overwrite = false
	}
	result := &PromptStudioImportResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if overwrite {
			tables := []any{&PromptStudioAttachment{}, &PromptStudioVersion{}, &PromptStudioSnippet{}, &PromptStudioProject{}, &PromptStudioFolder{}}
			for _, table := range tables {
				if err := tx.Where("user_id = ?", userId).Delete(table).Error; err != nil {
					return err
				}
			}
		}
		folders := withPromptStudioUser(userId, req.Folders, overwrite, tx, &PromptStudioFolder{})
		if len(folders) > 0 {
			if err := tx.Create(&folders).Error; err != nil {
				return err
			}
		}
		projects := withPromptStudioUser(userId, req.Projects, overwrite, tx, &PromptStudioProject{})
		if len(projects) > 0 {
			if err := tx.Create(&projects).Error; err != nil {
				return err
			}
		}
		versions := withPromptStudioUser(userId, req.Versions, overwrite, tx, &PromptStudioVersion{})
		// 版本内容可能缺少哈希（老导出文件），这里统一补齐
		for _, v := range versions {
			if v.ContentHash == "" && v.Content != "" {
				v.ContentHash = ComputePromptContentHash(v.Content)
			}
		}
		if len(versions) > 0 {
			if err := tx.Create(&versions).Error; err != nil {
				return err
			}
		}
		snippets := withPromptStudioUser(userId, req.Snippets, overwrite, tx, &PromptStudioSnippet{})
		if len(snippets) > 0 {
			if err := tx.Create(&snippets).Error; err != nil {
				return err
			}
		}
		attachments := withPromptStudioUser(userId, req.Attachments, overwrite, tx, &PromptStudioAttachment{})
		if len(attachments) > 0 {
			if err := tx.Create(&attachments).Error; err != nil {
				return err
			}
		}
		result.Folders = len(folders)
		result.Projects = len(projects)
		result.Versions = len(versions)
		result.Snippets = len(snippets)
		result.Attachments = len(attachments)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// withPromptStudioUser 过滤掉无效记录、绑定 user_id，merge 模式下剔除已存在的 ID
// 由于四类主键都是 string，这里用 interface 约束统一处理
type promptStudioIDOwner interface {
	GetID() string
	SetUserID(int)
}

func withPromptStudioUser[T promptStudioIDOwner](userId int, items []T, overwrite bool, tx *gorm.DB, model any) []T {
	kept := make([]T, 0, len(items))
	for _, item := range items {
		if item.GetID() == "" {
			continue
		}
		item.SetUserID(userId)
		kept = append(kept, item)
	}
	if overwrite || len(kept) == 0 {
		return kept
	}
	ids := make([]string, 0, len(kept))
	for _, item := range kept {
		ids = append(ids, item.GetID())
	}
	var existing []string
	if err := tx.Model(model).Where("user_id = ? AND id IN ?", userId, ids).Pluck("id", &existing).Error; err != nil {
		return kept
	}
	if len(existing) == 0 {
		return kept
	}
	dup := make(map[string]struct{}, len(existing))
	for _, id := range existing {
		dup[id] = struct{}{}
	}
	filtered := make([]T, 0, len(kept))
	for _, item := range kept {
		if _, ok := dup[item.GetID()]; ok {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// PromptStudioSampleData 示例数据的文案，由前端按当前语言传入，缺省时使用中文
type PromptStudioSampleData struct {
	ProjectName    string `json:"projectName"`
	RootName       string `json:"rootName"`
	RootContent    string `json:"rootContent"`
	Branch1Name    string `json:"branch1Name"`
	Branch1Content string `json:"branch1Content"`
	Branch2Name    string `json:"branch2Name"`
	Branch2Content string `json:"branch2Content"`
}

func (s *PromptStudioSampleData) fillDefaults() {
	if s.ProjectName == "" {
		s.ProjectName = "示例项目"
	}
	if s.RootName == "" {
		s.RootName = "小狗嬉戏"
	}
	if s.RootContent == "" {
		s.RootContent = "一只可爱的小狗在春意盎然的公园草地上嬉戏"
	}
	if s.Branch1Name == "" {
		s.Branch1Name = "帅气小狗"
	}
	if s.Branch1Content == "" {
		s.Branch1Content = "一只威风凛凛帅气的德牧在春意盎然的公园草地上嬉戏"
	}
	if s.Branch2Name == "" {
		s.Branch2Name = "冬日小狗"
	}
	if s.Branch2Content == "" {
		s.Branch2Content = "一只可爱的小狗在冬季白雪覆盖的公园草地上嬉戏"
	}
}

// PromptStudioCreateSample 为全新用户创建示例项目（1 个项目 + 1 个根版本 + 2 个子版本）
func PromptStudioCreateSample(userId int, sample *PromptStudioSampleData) (*PromptStudioProject, *PromptStudioVersion, error) {
	if sample == nil {
		sample = &PromptStudioSampleData{}
	}
	sample.fillDefaults()
	now := promptStudioNow()
	project := &PromptStudioProject{
		ID:        common.GetUUID(),
		UserID:    userId,
		Name:      sample.ProjectName,
		CreatedAt: now,
		UpdatedAt: now,
	}
	root := &PromptStudioVersion{
		ID:          common.GetUUID(),
		UserID:      userId,
		ProjectID:   project.ID,
		Name:        sample.RootName,
		Content:     sample.RootContent,
		ContentHash: ComputePromptContentHash(sample.RootContent),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	branch1 := &PromptStudioVersion{
		ID:          common.GetUUID(),
		UserID:      userId,
		ProjectID:   project.ID,
		ParentID:    root.ID,
		Name:        sample.Branch1Name,
		Content:     sample.Branch1Content,
		ContentHash: ComputePromptContentHash(sample.Branch1Content),
		CreatedAt:   now + 1,
		UpdatedAt:   now + 1,
	}
	branch2 := &PromptStudioVersion{
		ID:          common.GetUUID(),
		UserID:      userId,
		ProjectID:   project.ID,
		ParentID:    root.ID,
		Name:        sample.Branch2Name,
		Content:     sample.Branch2Content,
		ContentHash: ComputePromptContentHash(sample.Branch2Content),
		CreatedAt:   now + 2,
		UpdatedAt:   now + 2,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}
		return tx.Create([]*PromptStudioVersion{root, branch1, branch2}).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return project, root, nil
}

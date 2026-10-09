package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Prompt Studio 项目的数据访问

// PromptStudioListProjects 列出项目
// folderId 为空表示不过滤目录，返回当前用户的全部项目
func PromptStudioListProjects(userId int, folderId string) ([]*PromptStudioProject, error) {
	var projects []*PromptStudioProject
	query := DB.Where("user_id = ?", userId)
	if folderId != "" {
		query = query.Where("folder_id = ?", folderId)
	}
	err := query.Order("updated_at desc").Find(&projects).Error
	return projects, err
}

// PromptStudioGetProject 按 id 获取项目（带用户校验）
func PromptStudioGetProject(userId int, id string) (*PromptStudioProject, error) {
	if id == "" {
		return nil, ErrPromptStudioNotFound
	}
	var project PromptStudioProject
	err := DB.Where("id = ? AND user_id = ?", id, userId).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// PromptStudioCreateProject 创建项目并在同一事务内创建其根版本
// 前端原实现 createProject 会顺带插入一个 content 为空的根版本，这里保持一致
func PromptStudioCreateProject(userId int, name string, folderId string) (*PromptStudioProject, *PromptStudioVersion, error) {
	now := promptStudioNow()
	project := &PromptStudioProject{
		ID:        common.GetUUID(),
		UserID:    userId,
		FolderID:  folderId,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	rootVersion := &PromptStudioVersion{
		ID:        common.GetUUID(),
		UserID:    userId,
		ProjectID: project.ID,
		ParentID:  "",
		Content:   "",
		// 与前端一致：根版本的 contentHash 为空串，而不是空内容的哈希
		ContentHash: "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}
		return tx.Create(rootVersion).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return project, rootVersion, nil
}

// PromptStudioUpdateProject 更新项目名称 / 标签 / 所属目录
// 只更新传入的非 nil 字段，并同步刷新 updatedAt
func PromptStudioUpdateProject(userId int, id string, name *string, tags *PromptStudioProjectTags, folderId *string) (*PromptStudioProject, error) {
	if _, err := PromptStudioGetProject(userId, id); err != nil {
		return nil, err
	}
	updates := map[string]any{
		"updated_at": promptStudioNow(),
	}
	if name != nil {
		updates["name"] = *name
	}
	if tags != nil {
		updates["tags"] = *tags
	}
	if folderId != nil {
		updates["folder_id"] = *folderId
	}
	if err := DB.Model(&PromptStudioProject{}).
		Where("id = ? AND user_id = ?", id, userId).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	return PromptStudioGetProject(userId, id)
}

// PromptStudioTouchProject 刷新项目的 updatedAt
func PromptStudioTouchProject(tx *gorm.DB, userId int, id string) error {
	return tx.Model(&PromptStudioProject{}).
		Where("id = ? AND user_id = ?", id, userId).
		Update("updated_at", promptStudioNow()).Error
}

// PromptStudioDeleteProject 删除项目，级联删除其全部版本与附件
func PromptStudioDeleteProject(userId int, id string) error {
	if _, err := PromptStudioGetProject(userId, id); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var versionIds []string
		if err := tx.Model(&PromptStudioVersion{}).
			Where("user_id = ? AND project_id = ?", userId, id).
			Pluck("id", &versionIds).Error; err != nil {
			return err
		}
		if len(versionIds) > 0 {
			if err := tx.Where("user_id = ? AND version_id IN ?", userId, versionIds).
				Delete(&PromptStudioAttachment{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ? AND project_id = ?", userId, id).
			Delete(&PromptStudioVersion{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userId).
			Delete(&PromptStudioProject{}).Error
	})
}

// PromptStudioCreateProject 创建项目

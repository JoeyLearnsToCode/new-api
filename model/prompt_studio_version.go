package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Prompt Studio 版本的数据访问

// PromptStudioDuplicatePrefix 重复内容错误前缀，前端据此弹出"仍然保存"确认框
const PromptStudioDuplicatePrefix = "DUPLICATE_DETECTED:"

// PromptStudioDuplicateError 检测到内容重复
type PromptStudioDuplicateError struct {
	VersionID string
}

func (e *PromptStudioDuplicateError) Error() string {
	return PromptStudioDuplicatePrefix + e.VersionID
}

// ErrPromptStudioRootExists 该项目已存在根版本
var ErrPromptStudioRootExists = errors.New("每个项目只能有一个根版本")

// ErrPromptStudioKeepRoot 至少需保留一个根版本
var ErrPromptStudioKeepRoot = errors.New("每个项目必须至少保留一个根版本")

// PromptStudioListVersions 列出版本
// projectId 为空表示不过滤项目，返回当前用户的全部版本（全局搜索用）
func PromptStudioListVersions(userId int, projectId string) ([]*PromptStudioVersion, error) {
	var versions []*PromptStudioVersion
	query := DB.Where("user_id = ?", userId)
	if projectId != "" {
		query = query.Where("project_id = ?", projectId)
	}
	err := query.Order("created_at asc").Find(&versions).Error
	return versions, err
}

// PromptStudioGetVersion 按 id 获取版本（带用户校验）
func PromptStudioGetVersion(userId int, id string) (*PromptStudioVersion, error) {
	if id == "" {
		return nil, ErrPromptStudioNotFound
	}
	var version PromptStudioVersion
	err := DB.Where("id = ? AND user_id = ?", id, userId).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// PromptStudioFindDuplicateVersion 按内容哈希查找重复版本
// 与前端 versionStore.checkDuplicate 一致：跨项目比对，只在当前用户范围内
func PromptStudioFindDuplicateVersion(userId int, contentHash string) (*PromptStudioVersion, error) {
	if contentHash == "" {
		return nil, nil
	}
	var version PromptStudioVersion
	err := DB.Where("user_id = ? AND content_hash = ?", userId, contentHash).First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &version, nil
}

// PromptStudioCreateVersionOptions 创建版本的入参
type PromptStudioCreateVersionOptions struct {
	ProjectID          string
	Content            string
	ParentID           string
	Name               string
	SkipDuplicateCheck bool
}

// PromptStudioCreateVersion 创建版本
// 校验根版本唯一性与内容重复，并在同一事务内刷新项目 updatedAt
func PromptStudioCreateVersion(userId int, opt PromptStudioCreateVersionOptions) (*PromptStudioVersion, error) {
	if opt.ParentID == "" {
		var rootCount int64
		if err := DB.Model(&PromptStudioVersion{}).
			Where("user_id = ? AND project_id = ? AND parent_id = ?", userId, opt.ProjectID, "").
			Count(&rootCount).Error; err != nil {
			return nil, err
		}
		if rootCount > 0 {
			return nil, ErrPromptStudioRootExists
		}
	}
	contentHash := ComputePromptContentHash(opt.Content)
	if !opt.SkipDuplicateCheck {
		duplicate, err := PromptStudioFindDuplicateVersion(userId, contentHash)
		if err != nil {
			return nil, err
		}
		if duplicate != nil {
			return nil, &PromptStudioDuplicateError{VersionID: duplicate.ID}
		}
	}
	now := promptStudioNow()
	version := &PromptStudioVersion{
		ID:          common.GetUUID(),
		UserID:      userId,
		ProjectID:   opt.ProjectID,
		ParentID:    opt.ParentID,
		Name:        opt.Name,
		Content:     opt.Content,
		ContentHash: contentHash,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		return PromptStudioTouchProject(tx, userId, opt.ProjectID)
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

// PromptStudioUpdateVersionOptions 更新版本的入参，nil 表示该字段不更新
type PromptStudioUpdateVersionOptions struct {
	Content *string
	Name    *string
	Score   *int
	Notes   *string
	// TouchProject 为 false 时不同步刷新项目 updatedAt（评分、备注属于轻量修改）
	TouchProject bool
}

// PromptStudioUpdateVersion 更新版本
func PromptStudioUpdateVersion(userId int, id string, opt PromptStudioUpdateVersionOptions) (*PromptStudioVersion, error) {
	version, err := PromptStudioGetVersion(userId, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if opt.Content != nil {
		updates["content"] = *opt.Content
		updates["content_hash"] = ComputePromptContentHash(*opt.Content)
		updates["updated_at"] = promptStudioNow()
	}
	if opt.Name != nil {
		updates["name"] = *opt.Name
		updates["updated_at"] = promptStudioNow()
	}
	if opt.Score != nil {
		updates["score"] = *opt.Score
	}
	if opt.Notes != nil {
		updates["notes"] = *opt.Notes
	}
	if len(updates) == 0 {
		return version, nil
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&PromptStudioVersion{}).
			Where("id = ? AND user_id = ?", id, userId).
			Updates(updates).Error; err != nil {
			return err
		}
		if opt.TouchProject {
			return PromptStudioTouchProject(tx, userId, version.ProjectID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return PromptStudioGetVersion(userId, id)
}

// PromptStudioDeleteVersion 删除版本
// 与前端 versionStore.deleteVersion 一致：子版本"接骨"到被删版本的父级，
// 同时删除该版本的附件；根版本在项目中仅剩一个时不允许删除
func PromptStudioDeleteVersion(userId int, id string) error {
	version, err := PromptStudioGetVersion(userId, id)
	if err != nil {
		return err
	}
	if version.ParentID == "" {
		var rootCount int64
		if err := DB.Model(&PromptStudioVersion{}).
			Where("user_id = ? AND project_id = ? AND parent_id = ?", userId, version.ProjectID, "").
			Count(&rootCount).Error; err != nil {
			return err
		}
		if rootCount <= 1 {
			return ErrPromptStudioKeepRoot
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&PromptStudioVersion{}).
			Where("user_id = ? AND parent_id = ?", userId, id).
			Update("parent_id", version.ParentID).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND version_id = ?", userId, id).
			Delete(&PromptStudioAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ? AND user_id = ?", id, userId).
			Delete(&PromptStudioVersion{}).Error; err != nil {
			return err
		}
		return PromptStudioTouchProject(tx, userId, version.ProjectID)
	})
}

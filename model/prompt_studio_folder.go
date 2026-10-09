package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Prompt Studio 文件夹的数据访问
// 所有方法都以 userId 限定，避免越权访问他人数据

// ErrPromptStudioNotFound 记录不存在
var ErrPromptStudioNotFound = errors.New("记录不存在")

// PromptStudioListFolders 列出当前用户的全部文件夹
func PromptStudioListFolders(userId int) ([]*PromptStudioFolder, error) {
	var folders []*PromptStudioFolder
	err := DB.Where("user_id = ?", userId).
		Order("created_at asc").
		Find(&folders).Error
	return folders, err
}

// PromptStudioGetFolder 按 id 获取文件夹（带用户校验）
func PromptStudioGetFolder(userId int, id string) (*PromptStudioFolder, error) {
	if id == "" {
		return nil, ErrPromptStudioNotFound
	}
	var folder PromptStudioFolder
	err := DB.Where("id = ? AND user_id = ?", id, userId).First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// PromptStudioCreateFolder 创建文件夹
func PromptStudioCreateFolder(folder *PromptStudioFolder) error {
	if folder.ID == "" {
		folder.ID = common.GetUUID()
	}
	if folder.CreatedAt == 0 {
		folder.CreatedAt = promptStudioNow()
	}
	return DB.Create(folder).Error
}

// PromptStudioUpdateFolderName 重命名文件夹
func PromptStudioUpdateFolderName(userId int, id string, name string) error {
	return DB.Model(&PromptStudioFolder{}).
		Where("id = ? AND user_id = ?", id, userId).
		Update("name", name).Error
}

// PromptStudioDeleteFolder 删除文件夹
// 与前端 projectStore.deleteFolder 行为一致：非级联删除，
// 子文件夹与子项目统一上提到被删文件夹的父级
func PromptStudioDeleteFolder(userId int, id string) error {
	folder, err := PromptStudioGetFolder(userId, id)
	if err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&PromptStudioFolder{}).
			Where("user_id = ? AND parent_id = ?", userId, id).
			Update("parent_id", folder.ParentID).Error; err != nil {
			return err
		}
		if err := tx.Model(&PromptStudioProject{}).
			Where("user_id = ? AND folder_id = ?", userId, id).
			Update("folder_id", folder.ParentID).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userId).
			Delete(&PromptStudioFolder{}).Error
	})
}

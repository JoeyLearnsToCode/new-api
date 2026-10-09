package model

import (
	"errors"
)

// Prompt Studio 附件的数据访问
// 文件内容以 base64 字符串存放在 content 列，列表查询统一 Omit("content") 避免拉取大字段

// PromptStudioMaxAttachmentSize 单个附件的大小上限（字节），超出直接拒绝
// 与前端 MainView 的附件校验阈值保持一致
const PromptStudioMaxAttachmentSize = 10 << 20 // 10MB

// PromptStudioListAttachments 列出某个版本下的附件（不含文件内容）
func PromptStudioListAttachments(userId int, versionId string) ([]*PromptStudioAttachment, error) {
	var attachments []*PromptStudioAttachment
	err := DB.Omit("content").
		Where("user_id = ? AND version_id = ?", userId, versionId).
		Order("created_at asc").
		Find(&attachments).Error
	return attachments, err
}

// PromptStudioGetAttachment 按 id 获取附件（含文件内容）
func PromptStudioGetAttachment(userId int, id string) (*PromptStudioAttachment, error) {
	if id == "" {
		return nil, ErrPromptStudioNotFound
	}
	var attachment PromptStudioAttachment
	err := DB.Where("id = ? AND user_id = ?", id, userId).First(&attachment).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// PromptStudioCreateAttachments 同版本批量创建附件
func PromptStudioCreateAttachments(attachments []*PromptStudioAttachment) error {
	if len(attachments) == 0 {
		return nil
	}
	now := promptStudioNow()
	for _, attachment := range attachments {
		if attachment.Size > PromptStudioMaxAttachmentSize {
			return errors.New("附件大小超出限制")
		}
		if attachment.CreatedAt == 0 {
			attachment.CreatedAt = now
		}
	}
	return DB.Create(&attachments).Error
}

// PromptStudioDeleteAttachment 删除附件
func PromptStudioDeleteAttachment(userId int, id string) error {
	result := DB.Where("id = ? AND user_id = ?", id, userId).
		Delete(&PromptStudioAttachment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPromptStudioNotFound
	}
	return nil
}

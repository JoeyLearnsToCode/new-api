package controller

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// Prompt Studio 附件接口
// 文件内容以 base64 存放，这里负责编解码，业务层只看到原始字节

// GetPromptStudioAttachments 列出某个版本下的附件（不含内容）
func GetPromptStudioAttachments(c *gin.Context) {
	versionID := c.Query("versionId")
	if versionID == "" {
		common.ApiErrorMsg(c, "versionId 不能为空")
		return
	}
	attachments, err := model.PromptStudioListAttachments(promptStudioUserId(c), versionID)
	promptStudioRespond(c, attachments, err)
}

// UploadPromptStudioAttachments 批量上传附件到指定版本
// 前端一次提交多个 file 分片，避免逐文件循环请求
func UploadPromptStudioAttachments(c *gin.Context) {
	versionID := c.Param("id")
	if versionID == "" {
		common.ApiErrorMsg(c, "versionId 不能为空")
		return
	}
	// 确认版本属于当前用户，避免往他人版本挂附件
	if _, err := model.PromptStudioGetVersion(promptStudioUserId(c), versionID); err != nil {
		promptStudioFail(c, err)
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		common.ApiErrorMsg(c, "读取上传文件失败")
		return
	}
	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		common.ApiErrorMsg(c, "未找到上传文件")
		return
	}
	userId := promptStudioUserId(c)
	attachments := make([]*model.PromptStudioAttachment, 0, len(fileHeaders))
	for _, fileHeader := range fileHeaders {
		if fileHeader.Size > model.PromptStudioMaxAttachmentSize {
			common.ApiErrorMsg(c, "附件大小超出限制："+fileHeader.Filename)
			return
		}
		reader, openErr := fileHeader.Open()
		if openErr != nil {
			common.ApiError(c, openErr)
			return
		}
		raw, readErr := io.ReadAll(reader)
		reader.Close()
		if readErr != nil {
			common.ApiError(c, readErr)
			return
		}
		fileType := fileHeader.Header.Get("Content-Type")
		if fileType == "" {
			fileType = "application/octet-stream"
		}
		attachments = append(attachments, &model.PromptStudioAttachment{
			ID:        common.GetUUID(),
			UserID:    userId,
			VersionID: versionID,
			FileName:  fileHeader.Filename,
			FileType:  fileType,
			Size:      int64(len(raw)),
			Content:   base64.StdEncoding.EncodeToString(raw),
		})
	}
	promptStudioRespond(c, attachments, model.PromptStudioCreateAttachments(attachments))
}

// DownloadPromptStudioAttachment 读取附件内容
// download=1 时以附件形式下载，否则直接内联返回（用于图片预览）
func DownloadPromptStudioAttachment(c *gin.Context) {
	attachment, err := model.PromptStudioGetAttachment(promptStudioUserId(c), c.Param("id"))
	if err != nil {
		promptStudioFail(c, err)
		return
	}
	raw, err := base64.StdEncoding.DecodeString(attachment.Content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	disposition := "inline"
	if c.Query("download") == "1" {
		disposition = "attachment"
	}
	c.Header("Content-Disposition", disposition+"; filename*=UTF-8''"+promptStudioEscapeFileName(attachment.FileName))
	c.Data(http.StatusOK, attachment.FileType, raw)
}

// DeletePromptStudioAttachment 删除附件
func DeletePromptStudioAttachment(c *gin.Context) {
	promptStudioRespond(c, nil, model.PromptStudioDeleteAttachment(promptStudioUserId(c), c.Param("id")))
}

// promptStudioEscapeFileName 去掉会破坏 Content-Disposition 的字符后做 URL 编码
func promptStudioEscapeFileName(name string) string {
	replacer := strings.NewReplacer("\r", "", "\n", "", "\"", "", "\\", "")
	return url.QueryEscape(replacer.Replace(name))
}

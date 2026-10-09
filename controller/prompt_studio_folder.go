package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// Prompt Studio 文件夹接口

type promptStudioCreateFolderRequest struct {
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

type promptStudioRenameFolderRequest struct {
	Name string `json:"name"`
}

// GetPromptStudioFolders 列出当前用户的全部文件夹
func GetPromptStudioFolders(c *gin.Context) {
	folders, err := model.PromptStudioListFolders(promptStudioUserId(c))
	promptStudioRespond(c, folders, err)
}

// CreatePromptStudioFolder 创建文件夹
func CreatePromptStudioFolder(c *gin.Context) {
	var req promptStudioCreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Name == "" {
		common.ApiErrorMsg(c, "文件夹名称不能为空")
		return
	}
	folder := &model.PromptStudioFolder{
		Name:     req.Name,
		ParentID: req.ParentID,
		UserID:   promptStudioUserId(c),
	}
	promptStudioRespond(c, folder, model.PromptStudioCreateFolder(folder))
}

// UpdatePromptStudioFolder 重命名文件夹
func UpdatePromptStudioFolder(c *gin.Context) {
	var req promptStudioRenameFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Name == "" {
		common.ApiErrorMsg(c, "文件夹名称不能为空")
		return
	}
	promptStudioRespond(c, nil, model.PromptStudioUpdateFolderName(promptStudioUserId(c), c.Param("id"), req.Name))
}

// DeletePromptStudioFolder 删除文件夹，子文件夹与子项目上提到父级
func DeletePromptStudioFolder(c *gin.Context) {
	promptStudioRespond(c, nil, model.PromptStudioDeleteFolder(promptStudioUserId(c), c.Param("id")))
}

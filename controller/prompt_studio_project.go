package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// Prompt Studio 项目接口

type promptStudioCreateProjectRequest struct {
	Name     string `json:"name"`
	FolderID string `json:"folderId"`
}

type promptStudioUpdateProjectRequest struct {
	Name     *string                        `json:"name"`
	Tags     *model.PromptStudioProjectTags `json:"tags"`
	FolderID *string                        `json:"folderId"`
}

// GetPromptStudioProjects 列出项目，folderId 为空时返回全部
func GetPromptStudioProjects(c *gin.Context) {
	projects, err := model.PromptStudioListProjects(promptStudioUserId(c), c.Query("folderId"))
	promptStudioRespond(c, projects, err)
}

// CreatePromptStudioProject 创建项目（同时创建根版本）
func CreatePromptStudioProject(c *gin.Context) {
	var req promptStudioCreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Name == "" {
		common.ApiErrorMsg(c, "项目名称不能为空")
		return
	}
	project, rootVersion, err := model.PromptStudioCreateProject(promptStudioUserId(c), req.Name, req.FolderID)
	if err != nil {
		promptStudioFail(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"project": project,
		"version": rootVersion,
	})
}

// UpdatePromptStudioProject 更新项目名称 / 标签 / 所属目录
func UpdatePromptStudioProject(c *gin.Context) {
	var req promptStudioUpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	project, err := model.PromptStudioUpdateProject(promptStudioUserId(c), c.Param("id"), req.Name, req.Tags, req.FolderID)
	promptStudioRespond(c, project, err)
}

// DeletePromptStudioProject 删除项目及其全部版本、附件
func DeletePromptStudioProject(c *gin.Context) {
	promptStudioRespond(c, nil, model.PromptStudioDeleteProject(promptStudioUserId(c), c.Param("id")))
}

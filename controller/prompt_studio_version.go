package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// Prompt Studio 版本接口

type promptStudioCreateVersionRequest struct {
	ProjectID          string `json:"projectId"`
	Content            string `json:"content"`
	ParentID           string `json:"parentId"`
	Name               string `json:"name"`
	SkipDuplicateCheck bool   `json:"skipDuplicateCheck"`
}

type promptStudioUpdateVersionRequest struct {
	Content      *string `json:"content"`
	Name         *string `json:"name"`
	Score        *int    `json:"score"`
	Notes        *string `json:"notes"`
	TouchProject *bool   `json:"touchProject"`
}

// GetPromptStudioVersions 列出版本，projectId 为空时返回当前用户的全部版本
func GetPromptStudioVersions(c *gin.Context) {
	versions, err := model.PromptStudioListVersions(promptStudioUserId(c), c.Query("projectId"))
	promptStudioRespond(c, versions, err)
}

// CreatePromptStudioVersion 创建版本
// 内容重复时返回 success:false 且 message 以 DUPLICATE_DETECTED: 开头
func CreatePromptStudioVersion(c *gin.Context) {
	var req promptStudioCreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.ProjectID == "" {
		common.ApiErrorMsg(c, "projectId 不能为空")
		return
	}
	version, err := model.PromptStudioCreateVersion(promptStudioUserId(c), model.PromptStudioCreateVersionOptions{
		ProjectID:          req.ProjectID,
		Content:            req.Content,
		ParentID:           req.ParentID,
		Name:               req.Name,
		SkipDuplicateCheck: req.SkipDuplicateCheck,
	})
	promptStudioRespond(c, version, err)
}

// UpdatePromptStudioVersion 更新版本内容 / 名称 / 评分 / 备注
func UpdatePromptStudioVersion(c *gin.Context) {
	var req promptStudioUpdateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	touchProject := true
	if req.TouchProject != nil {
		touchProject = *req.TouchProject
	}
	version, err := model.PromptStudioUpdateVersion(promptStudioUserId(c), c.Param("id"), model.PromptStudioUpdateVersionOptions{
		Content:      req.Content,
		Name:         req.Name,
		Score:        req.Score,
		Notes:        req.Notes,
		TouchProject: touchProject,
	})
	promptStudioRespond(c, version, err)
}

// DeletePromptStudioVersion 删除版本，子版本上提到父级
func DeletePromptStudioVersion(c *gin.Context) {
	promptStudioRespond(c, nil, model.PromptStudioDeleteVersion(promptStudioUserId(c), c.Param("id")))
}

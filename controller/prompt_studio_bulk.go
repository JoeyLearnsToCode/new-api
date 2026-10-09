package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// Prompt Studio 导入导出与示例数据
// 导入导出均为一次性批量请求，前端不做逐条循环

// ExportPromptStudioData 导出当前用户的全部数据，附件内容为 base64
func ExportPromptStudioData(c *gin.Context) {
	data, err := model.PromptStudioExportAll(promptStudioUserId(c))
	promptStudioRespond(c, data, err)
}

// ImportPromptStudioData 整包导入，mode 支持 merge（默认）与 overwrite
func ImportPromptStudioData(c *gin.Context) {
	var req model.PromptStudioImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	result, err := model.PromptStudioImportAll(promptStudioUserId(c), &req)
	promptStudioRespond(c, result, err)
}

// CreatePromptStudioSample 为全新用户创建示例项目
// 文案由前端按当前语言传入，缺省时使用中文
func CreatePromptStudioSample(c *gin.Context) {
	var req model.PromptStudioSampleData
	// 允许空 body
	_ = c.ShouldBindJSON(&req)
	project, version, err := model.PromptStudioCreateSample(promptStudioUserId(c), &req)
	if err != nil {
		promptStudioFail(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"project": project,
		"version": version,
	})
}

package controller

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Prompt Studio 控制器
// 路由组：/api/prompt-studio，统一挂载 middleware.UserAuth()
// 所有数据按 c.GetInt("id") 取到的当前用户隔离

// promptStudioUserId 取当前登录用户 ID
func promptStudioUserId(c *gin.Context) int {
	return c.GetInt("id")
}

// promptStudioRespond 统一的错误响应
// 业务错误沿用项目约定：HTTP 200 + success:false，
// 内容重复则通过 message 前缀 DUPLICATE_DETECTED: 交给前端弹出"仍然保存"确认
func promptStudioRespond(c *gin.Context, data any, err error) {
	if err != nil {
		promptStudioFail(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func promptStudioFail(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var duplicate *model.PromptStudioDuplicateError
	if errors.As(err, &duplicate) {
		common.ApiErrorMsg(c, duplicate.Error())
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, model.ErrPromptStudioNotFound) {
		common.ApiErrorMsg(c, "记录不存在")
		return
	}
	common.ApiError(c, err)
}

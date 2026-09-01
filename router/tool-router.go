package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

// SetToolRouter registers the public (no-auth) OpenAI stream/non-stream adapter.
// It is deliberately mounted outside /api so that gzip does not buffer the SSE
// responses produced by the non-stream -> stream conversion.
func SetToolRouter(router *gin.Engine) {
	toolRouter := router.Group("/tool")
	toolRouter.Use(middleware.RouteTag("tool"))
	toolRouter.Use(middleware.CORS())
	toolRouter.Use(middleware.CriticalRateLimit())
	{
		// 添加 /tool/stream 和 /tool/nonstream 用于转换仅流式、仅非流式上游API为两者兼容API
		// 使用方式：
		//   http://host:port/tool/stream/<目标URL>      上游只支持流式
		//   http://host:port/tool/nonstream/<目标URL>   上游只支持非流式
		// 格式兼容：
		//   /tool/nonstream/https://api.openai.com/v1/chat/completions   # 直接写
		//   /tool/nonstream/https:/api.openai.com/v1/chat/completions    # 单斜杠，自动补 //
		//   /tool/nonstream/api.openai.com/v1/chat/completions           # 省略 scheme，默认 https
		//   /tool/nonstream/https%3A%2F%2Fapi.openai.com%2Fv1%2F...      # 百分号编码
		toolRouter.Any("/stream/*target", controller.ToolProxyStream)
		toolRouter.Any("/nonstream/*target", controller.ToolProxyNonStream)
	}
}

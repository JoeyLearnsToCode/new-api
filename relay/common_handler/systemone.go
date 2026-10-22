package common_handler

import (
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// SystemOneHandler 处理 TypeSafe System One（/v1/systemone）响应：
// 原样回写上游响应，并把 usage 转成 new-api 的通用用量。
func SystemOneHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
	if common.DebugEnabled {
		println("systemone response body: ", string(responseBody))
	}

	var systemOneResp dto.SystemOneResponse
	if err = common.Unmarshal(responseBody, &systemOneResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.JSON(http.StatusOK, systemOneResp)

	// 上游没给用量时返回 nil，交给 postConsumeQuota 按预估值结算
	if systemOneResp.Usage.InputTokens == 0 && systemOneResp.Usage.OutputTokens == 0 {
		return nil, nil
	}
	return systemOneResp.ToUsage(), nil
}

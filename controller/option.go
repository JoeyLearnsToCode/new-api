package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func GetOptions(c *gin.Context) {
	var options []*model.Option
	common.OptionMapRWMutex.Lock()
	for k, v := range common.OptionMap {
		if strings.HasSuffix(k, "Token") || strings.HasSuffix(k, "Secret") || strings.HasSuffix(k, "Key") {
			continue
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: common.Interface2String(v),
		})
	}
	common.OptionMapRWMutex.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
	return
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// updateSingleOption validates and persists a single option.
// Returns an error message string if validation fails, empty string on success.
func updateSingleOption(key string, value any) string {
	// Normalize value to string
	var strValue string
	switch v := value.(type) {
	case bool:
		strValue = common.Interface2String(v)
	case float64:
		strValue = common.Interface2String(v)
	case int:
		strValue = common.Interface2String(v)
	default:
		strValue = fmt.Sprintf("%v", v)
	}

	// Validate
	switch key {
	case "GitHubOAuthEnabled":
		if strValue == "true" && common.GitHubClientId == "" {
			return "无法启用 GitHub OAuth，请先填入 GitHub Client Id 以及 GitHub Client Secret！"
		}
	case "discord.enabled":
		if strValue == "true" && system_setting.GetDiscordSettings().ClientId == "" {
			return "无法启用 Discord OAuth，请先填入 Discord Client Id 以及 Discord Client Secret！"
		}
	case "oidc.enabled":
		if strValue == "true" && system_setting.GetOIDCSettings().ClientId == "" {
			return "无法启用 OIDC 登录，请先填入 OIDC Client Id 以及 OIDC Client Secret！"
		}
	case "LinuxDOOAuthEnabled":
		if strValue == "true" && common.LinuxDOClientId == "" {
			return "无法启用 LinuxDO OAuth，请先填入 LinuxDO Client Id 以及 LinuxDO Client Secret！"
		}
	case "EmailDomainRestrictionEnabled":
		if strValue == "true" && len(common.EmailDomainWhitelist) == 0 {
			return "无法启用邮箱域名限制，请先填入限制的邮箱域名！"
		}
	case "WeChatAuthEnabled":
		if strValue == "true" && common.WeChatServerAddress == "" {
			return "无法启用微信登录，请先填入微信登录相关配置信息！"
		}
	case "TurnstileCheckEnabled":
		if strValue == "true" && common.TurnstileSiteKey == "" {
			return "无法启用 Turnstile 校验，请先填入 Turnstile 校验相关配置信息！"
		}
	case "TelegramOAuthEnabled":
		if strValue == "true" && common.TelegramBotToken == "" {
			return "无法启用 Telegram OAuth，请先填入 Telegram Bot Token！"
		}
	case "GroupRatio":
		if err := ratio_setting.CheckGroupRatio(strValue); err != nil {
			return err.Error()
		}
	case "ImageRatio":
		if err := ratio_setting.UpdateImageRatioByJSONString(strValue); err != nil {
			return "图片倍率设置失败: " + err.Error()
		}
	case "AudioRatio":
		if err := ratio_setting.UpdateAudioRatioByJSONString(strValue); err != nil {
			return "音频倍率设置失败: " + err.Error()
		}
	case "AudioCompletionRatio":
		if err := ratio_setting.UpdateAudioCompletionRatioByJSONString(strValue); err != nil {
			return "音频补全倍率设置失败: " + err.Error()
		}
	case "ModelRequestRateLimitGroup":
		if err := setting.CheckModelRequestRateLimitGroup(strValue); err != nil {
			return err.Error()
		}
	case "console_setting.api_info":
		if err := console_setting.ValidateConsoleSettings(strValue, "ApiInfo"); err != nil {
			return err.Error()
		}
	case "console_setting.announcements":
		if err := console_setting.ValidateConsoleSettings(strValue, "Announcements"); err != nil {
			return err.Error()
		}
	case "console_setting.faq":
		if err := console_setting.ValidateConsoleSettings(strValue, "FAQ"); err != nil {
			return err.Error()
		}
	case "console_setting.uptime_kuma_groups":
		if err := console_setting.ValidateConsoleSettings(strValue, "UptimeKumaGroups"); err != nil {
			return err.Error()
		}
	}

	err := model.UpdateOption(key, strValue)
	if err != nil {
		return err.Error()
	}
	return ""
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := json.NewDecoder(c.Request.Body).Decode(&option)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	if errMsg := updateSingleOption(option.Key, option.Value); errMsg != "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// ImportOptionsRequest is the request body for batch option import
type ImportOptionsRequest struct {
	Settings map[string]any `json:"settings"`
}

// ImportOptionResult represents the result of importing a single option
type ImportOptionResult struct {
	Key    string `json:"key"`
	Reason string `json:"reason,omitempty"`
}

// ImportOptions imports multiple options in a single request
// POST /api/option/import
func ImportOptions(c *gin.Context) {
	var req ImportOptionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	if len(req.Settings) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "没有可导入的设置",
		})
		return
	}

	successCount := 0
	failed := make([]ImportOptionResult, 0)

	for key, value := range req.Settings {
		if errMsg := updateSingleOption(key, value); errMsg != "" {
			failed = append(failed, ImportOptionResult{
				Key:    key,
				Reason: errMsg,
			})
			continue
		}
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"success_count": successCount,
			"fail_count":    len(failed),
			"failed":        failed,
		},
	})
}

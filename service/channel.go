package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

func formatNotifyType(channelId int, status int) string {
	return fmt.Sprintf("%s_%d_%d", dto.NotifyTypeChannelUpdate, channelId, status)
}

// disable & notify
func DisableChannel(channelError types.ChannelError, reason string) {
	common.SysLog(fmt.Sprintf("通道「%s」（#%d）发生错误，准备禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, reason))

	// 检查是否启用自动禁用功能
	if !channelError.AutoBan {
		common.SysLog(fmt.Sprintf("通道「%s」（#%d）未启用自动禁用功能，跳过禁用操作", channelError.ChannelName, channelError.ChannelId))
		return
	}

	success := model.UpdateChannelStatus(channelError.ChannelId, channelError.UsingKey, common.ChannelStatusAutoDisabled, reason)
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被禁用", channelError.ChannelName, channelError.ChannelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, reason)
		NotifyRootUser(formatNotifyType(channelError.ChannelId, common.ChannelStatusAutoDisabled), subject, content)
	}
}

func EnableChannel(channelId int, usingKey string, channelName string) {
	success := model.UpdateChannelStatus(channelId, usingKey, common.ChannelStatusEnabled, "")
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		NotifyRootUser(formatNotifyType(channelId, common.ChannelStatusEnabled), subject, content)
	}
}

func ShouldDisableChannel(channelType int, err *types.NewAPIError) bool {
	if !common.AutomaticDisableChannelEnabled {
		return false
	}
	if err == nil {
		return false
	}
	if types.IsChannelError(err) {
		return true
	}
	if types.IsSkipRetryError(err) {
		return false
	}
	if err.StatusCode == http.StatusUnauthorized {
		return true
	}
	if err.StatusCode == http.StatusForbidden {
		switch channelType {
		case constant.ChannelTypeGemini:
			return true
		}
	}
	oaiErr := err.ToOpenAIError()
	switch oaiErr.Code {
	case "invalid_api_key":
		return true
	case "account_deactivated":
		return true
	case "billing_not_active":
		return true
	case "pre_consume_token_quota_failed":
		return true
	case "Arrearage":
		return true
	}
	switch oaiErr.Type {
	case "insufficient_quota":
		return true
	case "insufficient_user_quota":
		return true
	// https://docs.anthropic.com/claude/reference/errors
	case "authentication_error":
		return true
	case "permission_error":
		return true
	case "forbidden":
		return true
	}

	lowerMessage := strings.ToLower(err.Error())
	search, _ := AcSearch(lowerMessage, operation_setting.AutomaticDisableKeywords, true)
	return search
}

func ShouldEnableChannel(newAPIError *types.NewAPIError, status int) bool {
	if !common.AutomaticEnableChannelEnabled {
		return false
	}
	if newAPIError != nil {
		return false
	}
	if status != common.ChannelStatusAutoDisabled {
		return false
	}
	return true
}

// IsChannelExpired checks if a channel has expired based on its settings
func IsChannelExpired(channel *model.Channel) bool {
	setting := channel.GetSetting()
	if setting.ExpirationTime == "" {
		return false // No expiration time set
	}
	
	// Parse the RFC3339 format time string with timezone
	expirationTime, err := time.Parse(time.RFC3339, setting.ExpirationTime)
	if err != nil {
		common.SysLog(fmt.Sprintf("Failed to parse expiration time for channel %d: %v", channel.Id, err))
		return false // If parsing fails, don't expire the channel
	}
	
	// Compare with current UTC time
	return time.Now().UTC().After(expirationTime.UTC())
}

// DisableExpiredChannel disables a channel due to expiration
func DisableExpiredChannel(channel *model.Channel) {
	setting := channel.GetSetting()
	expirationTime, err := time.Parse(time.RFC3339, setting.ExpirationTime)
	if err != nil {
		common.SysLog(fmt.Sprintf("Failed to parse expiration time for channel %d: %v", channel.Id, err))
		return
	}
	
	reason := fmt.Sprintf("Channel expired at %s", expirationTime.Format("2006-01-02 15:04:05 MST"))
	common.SysLog(fmt.Sprintf("通道「%s」（#%d）已过期，准备禁用，过期时间：%s", channel.Name, channel.Id, reason))
	
	success := model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusExpiredDisabled, reason)
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已过期禁用", channel.Name, channel.Id)
		content := fmt.Sprintf("通道「%s」（#%d）已过期禁用，%s", channel.Name, channel.Id, reason)
		NotifyRootUser(formatNotifyType(channel.Id, common.ChannelStatusExpiredDisabled), subject, content)
	}
}

// ScanAndDisableExpiredChannels scans all channels and disables expired ones
func ScanAndDisableExpiredChannels() {
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		common.SysLog(fmt.Sprintf("Failed to get channels for expiration check: %v", err))
		return
	}
	
	expiredCount := 0
	for _, channel := range channels {
		// Only check enabled channels for expiration
		if channel.Status == common.ChannelStatusEnabled && IsChannelExpired(channel) {
			DisableExpiredChannel(channel)
			expiredCount++
		}
	}
	
	if expiredCount > 0 {
		common.SysLog(fmt.Sprintf("Expired channel scan completed, disabled %d channels", expiredCount))
	}
}

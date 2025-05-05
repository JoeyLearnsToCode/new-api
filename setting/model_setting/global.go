package model_setting

import (
	"one-api/setting/config"
)

type GlobalSettings struct {
	PassThroughRequestEnabled bool                `json:"pass_through_request_enabled"`
	ModelMapping              map[string][]string `json:"model_mapping"`
}

// 默认配置
var defaultOpenaiSettings = GlobalSettings{
	PassThroughRequestEnabled: false,
	ModelMapping:              map[string][]string{},
}

// 全局实例
var globalSettings = defaultOpenaiSettings

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("global", &globalSettings)
}

func GetGlobalSettings() *GlobalSettings {
	return &globalSettings
}

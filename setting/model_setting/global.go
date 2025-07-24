package model_setting

import (
	"bytes"
	"encoding/json"
	"one-api/setting/config"
)

type GlobalSettings struct {
	PassThroughRequestEnabled bool               `json:"pass_through_request_enabled"`
	ModelMapping              GlobalModelMapping `json:"model_mapping"`
}

// 默认配置
var defaultOpenaiSettings = GlobalSettings{
	PassThroughRequestEnabled: false,
	ModelMapping:              GlobalModelMapping{},
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

type GlobalModelMapping struct {
	// 等效组，任意元素相互等效
	Equivalents [][]string `json:"equivalents"`
	// 单向映射，组名是入口模型，值是底层模型列表
	OneWayModelMappings map[string][]string `json:"-"`
}

// 除了 Equivalents 之外，其他字段都解析到 OneWayModelMappings 中
func (g *GlobalModelMapping) UnmarshalJSON(data []byte) error {
	// 定义临时结构体来捕获原始JSON数据
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if equivalentsData, ok := raw["equivalents"]; ok {
		var equivalents [][]string
		if err := json.Unmarshal(equivalentsData, &equivalents); err != nil {
			return err
		}
		g.Equivalents = equivalents
		delete(raw, "equivalents")
	}

	if len(raw) > 0 {
		g.OneWayModelMappings = make(map[string][]string, len(raw))
		for key, value := range raw {
			var underlyingModels []string
			if err := json.Unmarshal(value, &underlyingModels); err != nil {
				return err
			}
			g.OneWayModelMappings[key] = underlyingModels
		}
	}

	return nil
}

// 把 OneWayModelMappings 打平到 JSON 顶层
func (g *GlobalModelMapping) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")

	if len(g.Equivalents) > 0 {
		equivalents, err := json.Marshal(g.Equivalents)
		if err != nil {
			return nil, err
		}
		buf.WriteString(`"equivalents":`)
		buf.Write(equivalents)
	}

	for key, value := range g.OneWayModelMappings {
		if buf.Len() > 1 {
			buf.WriteString(",")
		}
		val, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		buf.WriteString(`"`)
		buf.WriteString(key)
		buf.WriteString(`":`)
		buf.Write(val)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

package model_setting

import (
	"bytes"
	"encoding/json"
	"one-api/common"
	"one-api/setting/config"
	"strings"

	"github.com/dlclark/regexp2"
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

// resolveSingleGlobalModelMapping 将单个模型映射到目标模型列表
func resolveSingleGlobalModelMapping(model string, globalModelMapping *GlobalModelMapping, regexCache map[string]*regexp2.Regexp) []string {
	// 优先检查单向映射
	if len(globalModelMapping.OneWayModelMappings) > 0 && len(globalModelMapping.OneWayModelMappings[model]) > 0 {
		return globalModelMapping.OneWayModelMappings[model]
	}

	// 检查等效映射
	if len(globalModelMapping.Equivalents) > 0 {
		for _, equivalent := range globalModelMapping.Equivalents {
			if common.StringsContains(equivalent, model) {
				return equivalent
			}

			for _, eq := range equivalent {
				// 不处理都是、都不是正则的情况
				isRe1 := strings.HasPrefix(eq, "/")
				isRe2 := strings.HasPrefix(model, "/")
				if isRe1 && isRe2 || !isRe1 && !isRe2 {
					continue
				}
				var pattern, str string
				if isRe1 {
					pattern = eq[1:]
					str = model
				} else {
					pattern = model[1:]
					str = eq
				}
				
				// 使用缓存避免重复编译正则表达式
				re, ok := regexCache[pattern]
				if !ok {
					re = regexp2.MustCompile(pattern, regexp2.None)
					regexCache[pattern] = re
				}
				
				if matched, err := re.MatchString(str); err != nil {
					panic(err)
				} else if matched {
					return equivalent
				}
			}
		}
	}

	// 没有找到映射，返回原模型
	return []string{model}
}

// ResolveGlobalModelMappings 递归解析模型映射，直到收敛或达到最大迭代次数
func ResolveGlobalModelMappings(model string) ([]string, bool) {
	globalModelMapping := &GetGlobalSettings().ModelMapping
	
	// 使用集合跟踪所有已处理的模型，避免重复和循环
	processedModels := make(map[string]bool)
	currentModels := []string{model}
	usingGlobalModelMapping := false
	
	// 创建正则表达式缓存，避免重复编译
	regexCache := make(map[string]*regexp2.Regexp)

	const maxIterations = 5
	for i := 0; i < maxIterations; i++ {
		var nextModels []string
		hasNewMappings := false

		// 对当前批次的每个模型进行映射
		for _, currentModel := range currentModels {
			if processedModels[currentModel] {
				continue // 跳过已处理的模型
			}

			mappedModels := resolveSingleGlobalModelMapping(currentModel, globalModelMapping, regexCache)
			processedModels[currentModel] = true

			// 检查是否有新的映射结果
			if len(mappedModels) == 1 && mappedModels[0] == currentModel {
				// 没有映射，保留原模型
				nextModels = append(nextModels, currentModel)
			} else {
				// 有映射，标记使用了全局映射
				usingGlobalModelMapping = true
				hasNewMappings = true

				// 添加新的映射结果（排除已处理的）
				for _, mappedModel := range mappedModels {
					if !processedModels[mappedModel] {
						nextModels = append(nextModels, mappedModel)
					}
				}
			}
		}

		// 如果没有新的映射产生，说明已经收敛
		if !hasNewMappings {
			break
		}

		currentModels = nextModels
	}

	// 收集所有已处理的模型作为最终结果
	var finalModels []string
	for processedModel := range processedModels {
		finalModels = append(finalModels, processedModel)
	}

	// 如果没有使用映射，返回原始模型
	if !usingGlobalModelMapping {
		return []string{model}, false
	}

	return finalModels, true
}

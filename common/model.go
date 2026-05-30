package common

import (
	"strings"

	"github.com/dlclark/regexp2"
)

var (
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"gpt-image-1",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageGenerationModels {
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

// RegexMatchModel matches models available with regex (regexp2)
//  targetModels 可能包含正则、普通字符串的模型列表
//  modelsAvailable 普通字符串的模型列表
func RegexMatchModel(targetModels, modelsAvailable []string) (actuallAvailable []string) {
	for _, targetModel := range targetModels {
		if strings.HasPrefix(targetModel, "/") {
			// 正则表达式匹配
			re, err := regexp2.Compile(targetModel[1:], regexp2.None)
			if err != nil {
				continue
			}
			for _, availableModel := range modelsAvailable {
				if ok, err := re.MatchString(availableModel); err == nil && ok {
					actuallAvailable = append(actuallAvailable, availableModel)
				}
			}
		} else {
			// 普通字符串匹配
			for _, availableModel := range modelsAvailable {
				if targetModel == availableModel {
					actuallAvailable = append(actuallAvailable, availableModel)
					break
				}
			}
		}
	}
	return
}

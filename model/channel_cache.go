package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"one-api/common"
	"one-api/constant"
	"one-api/setting"
	"one-api/setting/model_setting"
	"one-api/setting/ratio_setting"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var group2model2channels map[string]map[string][]int // enabled channel
var channelsIDM map[int]*Channel                     // all channels include disabled
var channelSyncLock sync.RWMutex

func InitChannelCache() {
	if !common.MemoryCacheEnabled {
		return
	}
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}
	var abilities []*Ability
	DB.Find(&abilities)
	groups := make(map[string]bool)
	for _, ability := range abilities {
		groups[ability.Group] = true
	}
	newGroup2model2channels := make(map[string]map[string][]int)
	for group := range groups {
		newGroup2model2channels[group] = make(map[string][]int)
	}
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue // skip disabled channels
		}
		groups := strings.Split(channel.Group, ",")
		for _, group := range groups {
			models := strings.Split(channel.Models, ",")
			for _, model := range models {
				if _, ok := newGroup2model2channels[group][model]; !ok {
					newGroup2model2channels[group][model] = make([]int, 0)
				}
				newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], channel.Id)
			}
		}
	}

	// sort by priority
	for group, model2channels := range newGroup2model2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return newChannelId2channel[channels[i]].GetPriority() > newChannelId2channel[channels[j]].GetPriority()
			})
			newGroup2model2channels[group][model] = channels
		}
	}

	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	//channelsIDM = newChannelId2channel
	for i, channel := range newChannelId2channel {
		if channel.ChannelInfo.IsMultiKey {
			channel.Keys = channel.GetKeys()
			if channel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
				if oldChannel, ok := channelsIDM[i]; ok {
					// 存在旧的渠道，如果是多key且轮询，保留轮询索引信息
					if oldChannel.ChannelInfo.IsMultiKey && oldChannel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
						channel.ChannelInfo.MultiKeyPollingIndex = oldChannel.ChannelInfo.MultiKeyPollingIndex
					}
				}
			}
		}
	}
	channelsIDM = newChannelId2channel
	channelSyncLock.Unlock()
	common.SysLog("channels synced from database")
}

func SyncChannelCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func CacheGetRandomSatisfiedChannel(c *gin.Context, group string, model string, retry int) (*Channel, string, error) {
	var channel *Channel
	var err error
	selectGroup := group
	if group == "auto" {
		if len(setting.AutoGroups) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		for _, autoGroup := range setting.AutoGroups {
			if common.DebugEnabled {
				println("autoGroup:", autoGroup)
			}
			channel, _ = getRandomSatisfiedChannel(autoGroup, model, retry)
			if channel == nil {
				continue
			} else {
				c.Set("auto_group", autoGroup)
				selectGroup = autoGroup
				if common.DebugEnabled {
					println("selectGroup:", selectGroup)
				}
				break
			}
		}
	} else {
		channel, err = getRandomSatisfiedChannel(group, model, retry)
		if err != nil {
			return nil, group, err
		}
	}
	return channel, selectGroup, nil
}

// resolveSingleGlobalModelMapping 将单个模型映射到目标模型列表
func resolveSingleGlobalModelMapping(model string, globalModelMapping *model_setting.GlobalModelMapping) []string {
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
		}
	}

	// 没有找到映射，返回原模型
	return []string{model}
}

// resolveGlobalModelMappings 递归解析模型映射，直到收敛或达到最大迭代次数
func resolveGlobalModelMappings(model string, globalModelMapping *model_setting.GlobalModelMapping) ([]string, bool) {
	// 使用集合跟踪所有已处理的模型，避免重复和循环
	processedModels := make(map[string]bool)
	currentModels := []string{model}
	usingGlobalModelMapping := false
	
	const maxIterations = 5
	for i := 0; i < maxIterations; i++ {
		var nextModels []string
		hasNewMappings := false

		// 对当前批次的每个模型进行映射
		for _, currentModel := range currentModels {
			if processedModels[currentModel] {
				continue // 跳过已处理的模型
			}

			mappedModels := resolveSingleGlobalModelMapping(currentModel, globalModelMapping)
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

func getRandomSatisfiedChannel(group string, model string, retry int) (*Channel, error) {
	// 应用全局模型映射
	globalModelMapping := &model_setting.GetGlobalSettings().ModelMapping
	targetModels, usingGlobalModelMapping := resolveGlobalModelMappings(model, globalModelMapping)

	// if memory cache is disabled, get channel directly from database
	if !common.MemoryCacheEnabled {
		if usingGlobalModelMapping {
			channel, ability, err := GetRandomSatisfiedChannel(group, targetModels, retry)
			if err != nil {
				return channel, err
			}

			if common.StringsContains(channel.GetModels(), model) && common.StringsContains(targetModels, model) {
				return channel, nil
			}

			if model != ability.Model {
				modelMap := channel.MustGetModelMappingMap()
				modelMap[model] = ability.Model
				modelMappingBytes, _ := json.Marshal(modelMap)
				channel.ModelMapping = common.GetPointer[string](string(modelMappingBytes))
			}
			return channel, nil
		}
		channel, _, err := GetRandomSatisfiedChannel(group, targetModels, retry)
		return channel, err
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	var channels []int
	if usingGlobalModelMapping {
		for _, targetModel := range targetModels {
			channels = append(channels, group2model2channels[group][targetModel]...)
		}
		// 去重
		uniqueChannels := make(map[int]*Channel)
		for _, channelId := range channels {
			if uniqueChannels[channelId] == nil {
				uniqueChannels[channelId] = channelsIDM[channelId]
			}
		}
		channels = make([]int, 0, len(uniqueChannels))
		for channelId := range uniqueChannels {
			channels = append(channels, channelId)
		}
	} else {
		channels = group2model2channels[group][model]
	}

	if len(channels) == 0 {
		normalizedModel := ratio_setting.FormatMatchingModelName(model)
		channels = group2model2channels[group][normalizedModel]
	}
	if len(channels) == 0 {
		return nil, nil
	}

	selectedChannel, err := selectChannelByPriorityAndWeight(channels, retry)
	if err != nil {
		return nil, err
	}

	channelModels := selectedChannel.GetModels()
	if !usingGlobalModelMapping || (common.StringsContains(channelModels, model) && common.StringsContains(targetModels, model)) {
		return selectedChannel, nil
	}

	acceptableModels := common.StringsIntersection(channelModels, targetModels)
	if len(acceptableModels) == 0 {
		return nil, errors.New("no acceptable model left after global model mapping")
	}
	// 不修改原channel，复制一份
	copyChannel := *selectedChannel
	modelMap := copyChannel.MustGetModelMappingMap()
	modelMap[model] = acceptableModels[rand.Intn(len(acceptableModels))]
	modelMappingBytes, _ := json.Marshal(modelMap)
	copyChannel.ModelMapping = common.GetPointer[string](string(modelMappingBytes))
	return &copyChannel, nil
}

// selectChannelByPriorityAndWeight 根据优先级和权重随机选择channel
func selectChannelByPriorityAndWeight(channels []int, retry int) (*Channel, error) {
	if len(channels) == 1 {
		if channel, ok := channelsIDM[channels[0]]; ok {
			return channel, nil
		}
		return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channels[0])
	}

	// 获取所有唯一的优先级
	uniquePriorities := make(map[int]bool)
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			uniquePriorities[int(channel.GetPriority())] = true
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}

	// 将优先级从高到低排序
	var sortedUniquePriorities []int
	for priority := range uniquePriorities {
		sortedUniquePriorities = append(sortedUniquePriorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedUniquePriorities)))

	// 根据重试次数确定目标优先级
	if retry >= len(uniquePriorities) {
		retry = len(uniquePriorities) - 1
	}
	targetPriority := int64(sortedUniquePriorities[retry])

	// get the priority for the given retry number
	var targetChannels []*Channel
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			if channel.GetPriority() == targetPriority {
				targetChannels = append(targetChannels, channel)
			}
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}

	// 平滑系数
	smoothingFactor := 10
	// Calculate the total weight of all channels up to endIdx
	totalWeight := 0
	for _, channel := range targetChannels {
		totalWeight += channel.GetWeight() + smoothingFactor
	}
	// Generate a random value in the range [0, totalWeight)
	randomWeight := rand.Intn(totalWeight)

	// Find a channel based on its weight
	for _, channel := range targetChannels {
		randomWeight -= channel.GetWeight() + smoothingFactor
		if randomWeight < 0 {
			return channel, nil
		}
	}

	return nil, errors.New("channel not found")
}

func CacheGetChannel(id int) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return GetChannelById(id, true)
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return c, nil
}

func CacheGetChannelInfo(id int) (*ChannelInfo, error) {
	if !common.MemoryCacheEnabled {
		channel, err := GetChannelById(id, true)
		if err != nil {
			return nil, err
		}
		return &channel.ChannelInfo, nil
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return &c.ChannelInfo, nil
}

func CacheUpdateChannelStatus(id int, status int) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel, ok := channelsIDM[id]; ok {
		channel.Status = status
	}
	if status != common.ChannelStatusEnabled {
		// delete the channel from group2model2channels
		for group, model2channels := range group2model2channels {
			for model, channels := range model2channels {
				for i, channelId := range channels {
					if channelId == id {
						// remove the channel from the slice
						group2model2channels[group][model] = append(channels[:i], channels[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func CacheUpdateChannel(channel *Channel) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel == nil {
		return
	}

	println("CacheUpdateChannel:", channel.Id, channel.Name, channel.Status, channel.ChannelInfo.MultiKeyPollingIndex)

	println("before:", channelsIDM[channel.Id].ChannelInfo.MultiKeyPollingIndex)
	channelsIDM[channel.Id] = channel
	println("after :", channelsIDM[channel.Id].ChannelInfo.MultiKeyPollingIndex)
}

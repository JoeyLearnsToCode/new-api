package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"one-api/common"
	"one-api/setting/model_setting"
	"sort"
	"strings"
	"sync"
	"time"
)

var group2model2channels map[string]map[string][]*Channel
var channelsIDM map[int]*Channel
var channelSyncLock sync.RWMutex

func InitChannelCache() {
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Where("status = ?", common.ChannelStatusEnabled).Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}
	var abilities []*Ability
	DB.Find(&abilities)
	groups := make(map[string]bool)
	for _, ability := range abilities {
		groups[ability.Group] = true
	}
	newGroup2model2channels := make(map[string]map[string][]*Channel)
	newChannelsIDM := make(map[int]*Channel)
	for group := range groups {
		newGroup2model2channels[group] = make(map[string][]*Channel)
	}
	for _, channel := range channels {
		newChannelsIDM[channel.Id] = channel
		groups := strings.Split(channel.Group, ",")
		for _, group := range groups {
			models := strings.Split(channel.Models, ",")
			for _, model := range models {
				if _, ok := newGroup2model2channels[group][model]; !ok {
					newGroup2model2channels[group][model] = make([]*Channel, 0)
				}
				newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], channel)
			}
		}
	}

	// sort by priority
	for group, model2channels := range newGroup2model2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return channels[i].GetPriority() > channels[j].GetPriority()
			})
			newGroup2model2channels[group][model] = channels
		}
	}

	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	channelsIDM = newChannelsIDM
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

func CacheGetRandomSatisfiedChannel(group string, model string, retry int) (*Channel, error) {
	if strings.HasPrefix(model, "gpt-4-gizmo") {
		model = "gpt-4-gizmo-*"
	}
	if strings.HasPrefix(model, "gpt-4o-gizmo") {
		model = "gpt-4o-gizmo-*"
	}

	// 尝试从全局模型重定向里把传入 model 替换为等效模型 targetModels，然后参与渠道匹配
	var (
		targetModels            []string
		usingGlobalModelMapping bool
	)
	if len(model_setting.GetGlobalSettings().ModelMapping) > 0 && len(model_setting.GetGlobalSettings().ModelMapping[model]) > 0 {
		usingGlobalModelMapping = true
		targetModels = model_setting.GetGlobalSettings().ModelMapping[model]
	} else {
		usingGlobalModelMapping = false
		targetModels = []string{model}
	}

	// if memory cache is disabled, get channel directly from database
	if !common.MemoryCacheEnabled {
		if usingGlobalModelMapping {
			channel, ability, err := GetRandomSatisfiedChannel(group, targetModels, retry)
			if err != nil {
				return channel, err
			}

			if model != ability.Model {
				modelMap := make(map[string]string)
				err := json.Unmarshal([]byte(channel.GetModelMapping()), &modelMap)
				if err != nil {
					return nil, fmt.Errorf("unmarshal_model_mapping_failed")
				}
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
	var channels []*Channel
	if usingGlobalModelMapping {
		for _, targetModel := range targetModels {
			channels = append(channels, group2model2channels[group][targetModel]...)
		}
		// 去重
		uniqueChannels := make(map[int]*Channel)
		for _, channel := range channels {
			if uniqueChannels[channel.Id] == nil {
				uniqueChannels[channel.Id] = channel
			}
		}
		channels = make([]*Channel, 0, len(uniqueChannels))
		for _, channel := range uniqueChannels {
			channels = append(channels, channel)
		}
	} else {
		channels = group2model2channels[group][model]
	}
	if len(channels) == 0 {
		return nil, errors.New("channel not found")
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
	modelMap := make(map[string]string)
	err = json.Unmarshal([]byte(copyChannel.GetModelMapping()), &modelMap)
	if err != nil {
		return nil, fmt.Errorf("unmarshal_model_mapping_failed")
	}
	modelMap[model] = acceptableModels[rand.Intn(len(acceptableModels))]
	modelMappingBytes, _ := json.Marshal(modelMap)
	copyChannel.ModelMapping = common.GetPointer[string](string(modelMappingBytes))
	return &copyChannel, nil
}

// selectChannelByPriorityAndWeight 根据优先级和权重随机选择channel
func selectChannelByPriorityAndWeight(channels []*Channel, retry int) (*Channel, error) {
	// 获取所有唯一的优先级
	uniquePriorities := make(map[int]bool)
	for _, channel := range channels {
		uniquePriorities[int(channel.GetPriority())] = true
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
	for _, channel := range channels {
		if channel.GetPriority() == targetPriority {
			targetChannels = append(targetChannels, channel)
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
		return nil, errors.New(fmt.Sprintf("当前渠道# %d，已不存在", id))
	}
	return c, nil
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
}

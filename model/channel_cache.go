package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"one-api/common"
	"one-api/setting"
	"one-api/setting/model_setting"
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
	if channel == nil {
		return nil, group, errors.New("channel not found")
	}
	return channel, selectGroup, nil
}

func getRandomSatisfiedChannel(group string, model string, retry int) (*Channel, error) {
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
	globalModelMapping := model_setting.GetGlobalSettings().ModelMapping
	if len(globalModelMapping.OneWayModelMappings) > 0 && len(globalModelMapping.OneWayModelMappings[model]) > 0 {
		usingGlobalModelMapping = true
		targetModels = globalModelMapping.OneWayModelMappings[model]
	} else if len(globalModelMapping.Equivalents) > 0 {
		for _, equivalent := range globalModelMapping.Equivalents {
			if common.StringsContains(equivalent, model) {
				usingGlobalModelMapping = true
				targetModels = equivalent
				break
			}
		}
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
	if c.Status != common.ChannelStatusEnabled {
		return nil, fmt.Errorf("渠道# %d，已被禁用", id)
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
	if c.Status != common.ChannelStatusEnabled {
		return nil, fmt.Errorf("渠道# %d，已被禁用", id)
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

package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ExportChannels exports channel data as JSON
// GET /api/channel/export?ids=1,2,3
func ExportChannels(c *gin.Context) {
	idsParam := c.Query("ids")
	var channels []model.Channel
	var err error

	if idsParam != "" {
		// Export specific channels by IDs
		idStrs := strings.Split(idsParam, ",")
		ids := make([]int, 0, len(idStrs))
		for _, idStr := range idStrs {
			id, err := strconv.Atoi(strings.TrimSpace(idStr))
			if err != nil {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			common.ApiErrorMsg(c, "未提供有效的渠道ID")
			return
		}
		err = model.DB.Where("id IN ?", ids).Find(&channels).Error
	} else {
		// Export all channels
		err = model.DB.Find(&channels).Error
	}

	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"channels": channels,
	})
}

// ImportPreviewRequest is the request body for import preview
type ImportPreviewRequest struct {
	IDs []int `json:"ids"`
}

// ImportPreviewChannels checks which channel IDs already exist
// POST /api/channel/import/preview
func ImportPreviewChannels(c *gin.Context) {
	req := ImportPreviewRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.IDs) == 0 {
		common.ApiSuccess(c, gin.H{
			"existing_ids": []int{},
		})
		return
	}

	var existingIDs []int
	err := model.DB.Model(&model.Channel{}).Where("id IN ?", req.IDs).Pluck("id", &existingIDs).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"existing_ids": existingIDs,
	})
}

// ImportChannelsRequest is the request body for channel import
type ImportChannelsRequest struct {
	Channels []model.Channel `json:"channels"`
	Mode     string          `json:"mode"` // "create_only", "overwrite", or "upsert"
}

// ImportChannelResult represents the result of importing a single channel
type ImportChannelResult struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

// ImportChannels imports channels from JSON data
// POST /api/channel/import
func ImportChannels(c *gin.Context) {
	req := ImportChannelsRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.Channels) == 0 {
		common.ApiErrorMsg(c, "没有可导入的渠道数据")
		return
	}

	if req.Mode != "create_only" && req.Mode != "overwrite" && req.Mode != "upsert" {
		req.Mode = "create_only"
	}

	created := make([]int, 0)
	skipped := make([]int, 0)
	updated := make([]int, 0)
	failed := make([]ImportChannelResult, 0)

	// Get all existing IDs in one query for efficiency
	allIDs := make([]int, 0, len(req.Channels))
	for _, ch := range req.Channels {
		if ch.Id > 0 {
			allIDs = append(allIDs, ch.Id)
		}
	}

	existingIDSet := make(map[int]bool)
	if len(allIDs) > 0 {
		var existingIDs []int
		model.DB.Model(&model.Channel{}).Where("id IN ?", allIDs).Pluck("id", &existingIDs)
		for _, id := range existingIDs {
			existingIDSet[id] = true
		}
	}

	// For upsert mode: build a map of (id, name) -> exists from DB
	// to check if both ID and name match
	type idNameKey struct {
		ID   int
		Name string
	}
	existingIDNameSet := make(map[idNameKey]bool)
	if req.Mode == "upsert" && len(allIDs) > 0 {
		var channels []model.Channel
		model.DB.Where("id IN ?", allIDs).Find(&channels)
		for _, ch := range channels {
			existingIDNameSet[idNameKey{ID: ch.Id, Name: ch.Name}] = true
		}
	}

	for _, channel := range req.Channels {
		if channel.Key == "" {
			failed = append(failed, ImportChannelResult{
				ID:     channel.Id,
				Name:   channel.Name,
				Reason: "密钥不能为空",
			})
			continue
		}

		switch req.Mode {
		case "create_only":
			if existingIDSet[channel.Id] {
				skipped = append(skipped, channel.Id)
				continue
			}
			channel.CreatedTime = common.GetTimestamp()
			if err := channel.Insert(); err != nil {
				failed = append(failed, ImportChannelResult{
					ID:     channel.Id,
					Name:   channel.Name,
					Reason: err.Error(),
				})
				continue
			}
			created = append(created, channel.Id)

		case "overwrite":
			if existingIDSet[channel.Id] {
				if err := channel.Update(); err != nil {
					failed = append(failed, ImportChannelResult{
						ID:     channel.Id,
						Name:   channel.Name,
						Reason: err.Error(),
					})
					continue
				}
				updated = append(updated, channel.Id)
			} else {
				channel.CreatedTime = common.GetTimestamp()
				if err := channel.Insert(); err != nil {
					failed = append(failed, ImportChannelResult{
						ID:     channel.Id,
						Name:   channel.Name,
						Reason: err.Error(),
					})
					continue
				}
				created = append(created, channel.Id)
			}

		case "upsert":
			// If both ID and name match → overwrite; otherwise → append as new
			if existingIDNameSet[idNameKey{ID: channel.Id, Name: channel.Name}] {
				if err := channel.Update(); err != nil {
					failed = append(failed, ImportChannelResult{
						ID:     channel.Id,
						Name:   channel.Name,
						Reason: err.Error(),
					})
					continue
				}
				updated = append(updated, channel.Id)
			} else {
				// Append as new channel, let DB auto-generate ID
				channel.Id = 0
				channel.CreatedTime = common.GetTimestamp()
				if err := channel.Insert(); err != nil {
					failed = append(failed, ImportChannelResult{
						ID:     channel.Id,
						Name:   channel.Name,
						Reason: err.Error(),
					})
					continue
				}
				created = append(created, channel.Id)
			}
		}
	}

	// Refresh cache after import
	model.InitChannelCache()

	common.ApiSuccess(c, gin.H{
		"created": created,
		"skipped": skipped,
		"updated": updated,
		"failed":  failed,
	})
}

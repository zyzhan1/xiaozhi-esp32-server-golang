package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AgentChatSessionController 智能体聊天记录控制器
type AgentChatSessionController struct {
	DB *gorm.DB
}

// ============================================================
// 请求结构体
// ============================================================

// CreateAgentChatSessionRequest 创建智能体会话请求
type CreateAgentChatSessionRequest struct {
	AgentID        uint64      `json:"agent_id" binding:"required"`
	AgentName      string      `json:"agent_name" binding:"required"`
	ModelName      string      `json:"model_name" binding:"required"`
	ModelID        *uint64     `json:"model_id"`
	Title          string      `json:"title"`
	Tools          interface{} `json:"tools"`
	KnowledgeBases interface{} `json:"knowledge_bases"`
	TTSConfig      interface{} `json:"tts_config"`
	MCPServices    interface{} `json:"mcp_services"`
}

// UpdateAgentChatSessionRequest 更新智能体会话统计请求
type UpdateAgentChatSessionRequest struct {
	Summary        *string     `json:"summary"`
	TotalRounds    *uint32     `json:"total_rounds"`
	TotalTokens    *uint64     `json:"total_tokens"`
	TotalDuration  *uint64     `json:"total_duration"`
	Tools          interface{} `json:"tools"`
	KnowledgeBases interface{} `json:"knowledge_bases"`
	TTSConfig      interface{} `json:"tts_config"`
	MCPServices    interface{} `json:"mcp_services"`
}

// AppendAgentChatMessageItem 单条消息结构
type AppendAgentChatMessageItem struct {
	Role           string      `json:"role" binding:"required,oneof=user assistant system tool"`
	Content        string      `json:"content" binding:"required"`
	ModelName      *string     `json:"model_name"`
	Duration       *uint64     `json:"duration"`
	Tokens         *uint32     `json:"tokens"`
	Tools          interface{} `json:"tools"`
	KnowledgeBases interface{} `json:"knowledge_bases"`
	TTSConfig      interface{} `json:"tts_config"`
	MCPServices    interface{} `json:"mcp_services"`
}

// AppendAgentChatMessagesRequest 批量追加消息请求
type AppendAgentChatMessagesRequest struct {
	Messages []AppendAgentChatMessageItem `json:"messages" binding:"required,min=1,dive"`
}

// ============================================================
// 3.1 创建智能体会话（懒创建）
// ============================================================

// CreateAgentChatSession 创建智能体聊天会话
func (c *AgentChatSessionController) CreateAgentChatSession(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	var req CreateAgentChatSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title := req.Title
	if title == "" {
		title = "新对话"
	}

	session := &models.AgentChatSession{
		UserID:         userID,
		AgentID:        req.AgentID,
		AgentName:      req.AgentName,
		ModelName:      req.ModelName,
		ModelID:        req.ModelID,
		Title:          title,
		Tools:          toInterfaceSlice(req.Tools),
		KnowledgeBases: toInterfaceSlice(req.KnowledgeBases),
		TTSConfig:      req.TTSConfig,
		MCPServices:    toInterfaceSlice(req.MCPServices),
	}

	if err := c.DB.Create(session).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, session)
}

// ============================================================
// 3.2 追加智能体消息（批量插入 + 统计自动累加）
// ============================================================

// AppendAgentChatMessages 批量追加消息到智能体会话
func (c *AgentChatSessionController) AppendAgentChatMessages(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	// 验证会话存在且属于当前用户
	var session models.AgentChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var req AppendAgentChatMessagesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 批量创建消息并累计统计
	var totalTokens uint64
	var totalDuration uint64
	var rounds uint32
	var lastAssistantContent string

	for i := range req.Messages {
		item := &req.Messages[i]

		message := &models.AgentChatMessage{
			SessionID:      session.ID,
			UserID:         userID,
			Role:           item.Role,
			Content:        item.Content,
			ModelName:      item.ModelName,
			Duration:       item.Duration,
			Tokens:         item.Tokens,
			Tools:          toInterfaceSlice(item.Tools),
			KnowledgeBases: toInterfaceSlice(item.KnowledgeBases),
			TTSConfig:      item.TTSConfig,
			MCPServices:    toInterfaceSlice(item.MCPServices),
		}

		if err := c.DB.Create(message).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "保存消息失败: " + err.Error()})
			return
		}

		// 仅 assistant 消息参与统计
		if item.Role == "assistant" {
			if item.Tokens != nil {
				totalTokens += uint64(*item.Tokens)
			}
			if item.Duration != nil {
				totalDuration += *item.Duration
			}
			if item.Tokens != nil && *item.Tokens > 0 {
				rounds++
			}
			lastAssistantContent = item.Content
		}
	}

	// 更新会话统计信息（使用 gorm.Expr 原子累加，避免并发竞态）
	sessionUpdates := map[string]interface{}{}
	if totalTokens > 0 {
		sessionUpdates["total_tokens"] = gorm.Expr("total_tokens + ?", totalTokens)
	}
	if totalDuration > 0 {
		sessionUpdates["total_duration"] = gorm.Expr("total_duration + ?", totalDuration)
	}
	if rounds > 0 {
		sessionUpdates["total_rounds"] = gorm.Expr("total_rounds + ?", rounds)
	}
	// 更新摘要（取最后一条 assistant 消息内容的前 100 个字符）
	if lastAssistantContent != "" {
		runes := []rune(lastAssistantContent)
		if len(runes) > 100 {
			lastAssistantContent = string(runes[:100])
		}
		sessionUpdates["summary"] = lastAssistantContent
	}

	if len(sessionUpdates) > 0 {
		c.DB.Model(&models.AgentChatSession{}).Where("id = ?", session.ID).Updates(sessionUpdates)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":        true,
		"inserted_count": len(req.Messages),
	})
}

// ============================================================
// 3.3 获取智能体会话列表（分页，不返回大体积 JSON 字段）
// ============================================================

// GetAgentChatSessions 获取智能体会话列表
func (c *AgentChatSessionController) GetAgentChatSessions(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 列表接口排除大体积 JSON 字段
	query := c.DB.Model(&models.AgentChatSession{}).
		Where("user_id = ?", userID).
		Omit("tools", "knowledge_bases", "tts_config", "mcp_services")

	// 可选按 agent_id 筛选
	if agentIDStr := ctx.Query("agent_id"); agentIDStr != "" {
		if agentID, err := strconv.ParseUint(agentIDStr, 10, 64); err == nil {
			query = query.Where("agent_id = ?", agentID)
		}
	}

	var total int64
	query.Count(&total)

	var sessions []models.AgentChatSession
	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&sessions).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      sessions,
	})
}

// ============================================================
// 3.4 获取会话详情及历史消息（含 JSON 字段，消息分页）
// ============================================================

// GetAgentChatSessionDetail 获取智能体会话详情及消息列表
func (c *AgentChatSessionController) GetAgentChatSessionDetail(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	// 查询完整会话（含 JSON 字段）
	var session models.AgentChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 消息分页
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var total int64
	c.DB.Model(&models.AgentChatMessage{}).
		Where("session_id = ?", session.ID).
		Count(&total)

	var messages []models.AgentChatMessage
	offset := (page - 1) * pageSize
	c.DB.Where("session_id = ?", session.ID).
		Order("created_at ASC").
		Limit(pageSize).Offset(offset).
		Find(&messages)

	ctx.JSON(http.StatusOK, gin.H{
		"session":  session,
		"messages": messages,
		"total":    total,
	})
}

// ============================================================
// 3.5 更新会话统计数据
// ============================================================

// UpdateAgentChatSession 更新智能体会话统计及附加上下文
func (c *AgentChatSessionController) UpdateAgentChatSession(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	var session models.AgentChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var req UpdateAgentChatSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}

	// 普通字段
	if req.Summary != nil {
		updates["summary"] = *req.Summary
	}
	// 统计字段
	if req.TotalRounds != nil {
		updates["total_rounds"] = *req.TotalRounds
	}
	if req.TotalTokens != nil {
		updates["total_tokens"] = *req.TotalTokens
	}
	if req.TotalDuration != nil {
		updates["total_duration"] = *req.TotalDuration
	}

	// JSON 字段：直接在 controller 内序列化（避免手动调用 BeforeSave hook 的副作用）
	if req.Tools != nil {
		data, err := json.Marshal(req.Tools)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "tools 序列化失败"})
			return
		}
		updates["tools"] = string(data)
	}
	if req.KnowledgeBases != nil {
		data, err := json.Marshal(req.KnowledgeBases)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "knowledge_bases 序列化失败"})
			return
		}
		updates["knowledge_bases"] = string(data)
	}
	if req.TTSConfig != nil {
		data, err := json.Marshal(req.TTSConfig)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "tts_config 序列化失败"})
			return
		}
		updates["tts_config"] = string(data)
	}
	if req.MCPServices != nil {
		data, err := json.Marshal(req.MCPServices)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "mcp_services 序列化失败"})
			return
		}
		updates["mcp_services"] = string(data)
	}

	// 强制刷新 updated_at（因为 Updates map 不会自动触发 GORM 的 updatedAt）
	updates["updated_at"] = time.Now()

	if len(updates) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	if err := c.DB.Model(&models.AgentChatSession{}).
		Where("id = ?", session.ID).
		Updates(updates).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================
// 3.6 删除智能体会话（级联删除关联消息）
// ============================================================

// DeleteAgentChatSession 删除智能体会话及关联消息
func (c *AgentChatSessionController) DeleteAgentChatSession(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	// 验证会话存在且属于当前用户
	var session models.AgentChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 先删除关联的消息（兼容未启用外键 CASCADE 的场景）
	c.DB.Where("session_id = ?", session.ID).Delete(&models.AgentChatMessage{})

	// 删除会话
	if err := c.DB.Delete(&session).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================
// 辅助方法
// ============================================================

// getUserID 从 JWT 上下文中获取用户ID
func (c *AgentChatSessionController) getUserID(ctx *gin.Context) uint64 {
	id, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return 0
	}
	// user_id 可能是 uint、uint64 或 float64，统一转换
	switch v := id.(type) {
	case uint:
		return uint64(v)
	case uint64:
		return v
	case float64:
		return uint64(v)
	case int:
		return uint64(v)
	default:
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户ID类型异常"})
		return 0
	}
}

// toInterfaceSlice 将 interface{} 转为 []interface{}，若值不是 slice 则返回 nil
func toInterfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	if s, ok := v.([]interface{}); ok {
		return s
	}
	return nil
}

package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChatSessionController struct {
	DB *gorm.DB
}

// ============================================================
// 请求/响应结构体
// ============================================================

// CreateChatSessionRequest 创建会话请求
type CreateChatSessionRequest struct {
	ChatType     string   `json:"chat_type" binding:"required,oneof=model agent"` // model-模型对话 / agent-智能体对话
	TargetID     uint64   `json:"target_id" binding:"required"`                   // 模型配置ID 或 智能体ID
	TargetName   *string  `json:"target_name"`                                    // 模型名称 或 智能体名称
	Title        *string  `json:"title"`                                          // 会话标题
	ToolsUsed    []string `json:"tools_used"`                                     // 使用的工具列表
	KnowledgeIDs []int64  `json:"knowledge_ids"`                                  // 关联的知识库ID列表
	TTSAudioURL  *string  `json:"tts_audio_url"`                                  // TTS 语音文件地址
	ASRAudioURL  *string  `json:"asr_audio_url"`                                  // ASR 语音文件地址
}

// UpdateChatSessionRequest 更新会话请求
type UpdateChatSessionRequest struct {
	Title         *string  `json:"title"`
	Summary       *string  `json:"summary"`
	TargetName    *string  `json:"target_name"`
	ToolsUsed     []string `json:"tools_used"`
	KnowledgeIDs  []int64  `json:"knowledge_ids"`
	TotalTokens   *int64   `json:"total_tokens"`
	TotalDuration *int64   `json:"total_duration"`
	MessageCount  *int     `json:"message_count"`
	TotalBytes    *int64   `json:"total_bytes"`
}

// AppendChatMessageItem 单条消息结构
type AppendChatMessageItem struct {
	Role        string  `json:"role" binding:"required,oneof=user assistant system"`
	Content     string  `json:"content" binding:"required"`
	ModelName   *string `json:"model_name,omitempty"` // AI回答时使用的模型名称，用户消息应传null或不传
	Tokens      int     `json:"tokens"`
	Duration    int64   `json:"duration"`
	TTSAudioURL *string `json:"tts_audio_url"`
	ASRAudioURL *string `json:"asr_audio_url"`
}

// AppendChatMessagesRequest 批量追加消息请求
type AppendChatMessagesRequest struct {
	Messages []AppendChatMessageItem `json:"messages" binding:"required,min=1,dive"`
}

// ============================================================
// 会话 CRUD
// ============================================================

// GetChatSessions 获取会话列表（分页）
func (c *ChatSessionController) GetChatSessions(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	chatType := ctx.Query("chat_type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := c.DB.Model(&models.ChatSession{}).Where("user_id = ?", userID)
	if chatType != "" {
		query = query.Where("chat_type = ?", chatType)
	}

	var total int64
	query.Count(&total)

	var sessions []models.ChatSession
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

// CreateChatSession 创建会话
func (c *ChatSessionController) CreateChatSession(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	var req CreateChatSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session := &models.ChatSession{
		UserID:       userID,
		ChatType:     req.ChatType,
		TargetID:     req.TargetID,
		TargetName:   req.TargetName,
		Title:        req.Title,
		ToolsUsed:    req.ToolsUsed,
		KnowledgeIDs: req.KnowledgeIDs,
		TTSAudioURL:  req.TTSAudioURL,
		ASRAudioURL:  req.ASRAudioURL,
	}

	if err := c.DB.Create(session).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, session)
}

// UpdateChatSession 更新会话
func (c *ChatSessionController) UpdateChatSession(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	var session models.ChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败: " + err.Error()})
		return
	}

	var req UpdateChatSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}

	// 普通字段
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Summary != nil {
		updates["summary"] = *req.Summary
	}
	if req.TargetName != nil {
		updates["target_name"] = *req.TargetName
	}

	// 统计字段
	if req.TotalTokens != nil {
		updates["total_tokens"] = *req.TotalTokens
	}
	if req.TotalDuration != nil {
		updates["total_duration"] = *req.TotalDuration
	}
	if req.MessageCount != nil {
		updates["message_count"] = *req.MessageCount
	}
	if req.TotalBytes != nil {
		updates["total_bytes"] = *req.TotalBytes
	}

	// JSON 字段：直接在 controller 内序列化，避免手动调用 BeforeSave hook 的副作用
	if req.ToolsUsed != nil {
		data, err := json.Marshal(req.ToolsUsed)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "tools_used 序列化失败"})
			return
		}
		updates["tools_used"] = string(data)
	}
	if req.KnowledgeIDs != nil {
		data, err := json.Marshal(req.KnowledgeIDs)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "knowledge_ids 序列化失败"})
			return
		}
		updates["knowledge_ids"] = string(data)
	}

	if len(updates) == 0 {
		ctx.JSON(http.StatusOK, session)
		return
	}

	if err := c.DB.Model(&session).Updates(updates).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		return
	}

	// 重新查询返回完整数据
	c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session)
	ctx.JSON(http.StatusOK, session)
}

// DeleteChatSession 删除会话（同时删除关联消息）
func (c *ChatSessionController) DeleteChatSession(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	var session models.ChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 先删除关联的消息
	c.DB.Where("session_id = ?", session.ID).Delete(&models.ChatSessionMessage{})

	// 删除会话
	if err := c.DB.Delete(&session).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// GetChatSessionDetail 获取会话详情（包含消息列表）
func (c *ChatSessionController) GetChatSessionDetail(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	var session models.ChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 查询关联消息（时间正序，旧 -> 新）
	var messages []models.ChatSessionMessage
	c.DB.Where("session_id = ?", session.ID).
		Order("created_at ASC").
		Find(&messages)

	ctx.JSON(http.StatusOK, gin.H{
		"session":  session,
		"messages": messages,
	})
}

// ============================================================
// 消息操作
// ============================================================

// GetChatSessionMessages 获取指定会话的消息列表（分页）
func (c *ChatSessionController) GetChatSessionMessages(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	// 验证会话存在且属于当前用户
	var session models.ChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "50"))
	role := ctx.Query("role")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	query := c.DB.Model(&models.ChatSessionMessage{}).
		Where("session_id = ?", session.ID)

	if role != "" {
		query = query.Where("role = ?", role)
	}

	var total int64
	query.Count(&total)

	var messages []models.ChatSessionMessage
	offset := (page - 1) * pageSize
	if err := query.Order("created_at ASC").
		Limit(pageSize).Offset(offset).
		Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      messages,
	})
}

// AppendChatMessages 批量追加消息到会话
func (c *ChatSessionController) AppendChatMessages(ctx *gin.Context) {
	userID := c.getUserID(ctx)
	if userID == 0 {
		return
	}

	id := ctx.Param("id")

	// 验证会话存在且属于当前用户
	var session models.ChatSession
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var req AppendChatMessagesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 批量创建消息并累计统计
	var createdMessages []models.ChatSessionMessage
	var totalTokens int
	var totalDuration int64
	var lastContent string

	for i := range req.Messages {
		item := &req.Messages[i]

		// 业务规则：用户消息 model_name 为 NULL，assistant 消息记录实际模型名称
		var modelName *string
		if item.Role != "user" && item.ModelName != nil {
			modelName = item.ModelName
		}

		message := &models.ChatSessionMessage{
			SessionID:   session.ID,
			Role:        item.Role,
			Content:     item.Content,
			ModelName:   modelName,
			Tokens:      item.Tokens,
			Duration:    item.Duration,
			TTSAudioURL: item.TTSAudioURL,
			ASRAudioURL: item.ASRAudioURL,
		}

		if err := c.DB.Create(message).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存第 %d 条消息失败: %v", i+1, err)})
			return
		}

		createdMessages = append(createdMessages, *message)
		totalTokens += item.Tokens
		totalDuration += item.Duration
		lastContent = item.Content
	}

	// 更新会话统计信息
	sessionUpdates := map[string]interface{}{
		"message_count":  gorm.Expr("message_count + ?", len(req.Messages)),
		"total_tokens":   gorm.Expr("total_tokens + ?", totalTokens),
		"total_duration": gorm.Expr("total_duration + ?", totalDuration),
	}

	// 更新摘要（取最后一条消息内容的前100个字符）
	if lastContent != "" {
		if len(lastContent) > 100 {
			sessionUpdates["summary"] = lastContent[:100]
		} else {
			sessionUpdates["summary"] = lastContent
		}
	}

	// 如果是第一条消息且会话没有标题，自动用第一条用户消息设置标题
	if session.Title == nil || *session.Title == "" {
		for i := range req.Messages {
			if req.Messages[i].Role == "user" {
				title := req.Messages[i].Content
				if len(title) > 50 {
					title = title[:50]
				}
				sessionUpdates["title"] = title
				break
			}
		}
	}

	c.DB.Model(&session).Updates(sessionUpdates)

	ctx.JSON(http.StatusCreated, gin.H{
		"messages": createdMessages,
		"count":    len(createdMessages),
	})
}

// ============================================================
// 辅助方法
// ============================================================

// getUserID 从 JWT 上下文中获取用户ID
func (c *ChatSessionController) getUserID(ctx *gin.Context) uint64 {
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

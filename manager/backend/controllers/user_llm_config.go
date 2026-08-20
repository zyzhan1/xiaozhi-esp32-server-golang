package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
)

// BoolOrInt8 自定义类型，JSON 反序列化时同时兼容布尔值（true/false）和整数值（1/0）
type BoolOrInt8 int8

func (b *BoolOrInt8) UnmarshalJSON(data []byte) error {
	// 尝试整数
	var i int8
	if err := json.Unmarshal(data, &i); err == nil {
		*b = BoolOrInt8(i)
		return nil
	}
	// 尝试布尔
	var bo bool
	if err := json.Unmarshal(data, &bo); err == nil {
		if bo {
			*b = BoolOrInt8(1)
		} else {
			*b = BoolOrInt8(0)
		}
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into BoolOrInt8", string(data))
}

// UserLLMConfigPayload 创建/更新用户LLM配置的请求体
type UserLLMConfigPayload struct {
	Name      string      `json:"name" binding:"required"`
	Type      string      `json:"type" binding:"required"` // LLM 提供商/协议类型，如 deepSeek、openai、ollama、dify、coze 等
	BaseURL   string      `json:"base_url" binding:"required"`
	ModelName string      `json:"model_name"`
	APIKey    *string     `json:"api_key"`
	Status    *BoolOrInt8 `json:"status"`
}

// ListUserLLMConfigs 获取当前用户的LLM配置列表
func (uc *UserController) ListUserLLMConfigs(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效用户上下文"})
		return
	}

	var configs []models.UserLLMConfig
	if err := uc.DB.Where("user_id = ?", uid).Order("id DESC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取配置列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": configs})
}

// CreateUserLLMConfig 添加用户LLM配置
func (uc *UserController) CreateUserLLMConfig(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效用户上下文"})
		return
	}

	var req UserLLMConfigPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	config := models.UserLLMConfig{
		UserID:    uint64(uid),
		Name:      strings.TrimSpace(req.Name),
		Type:      strings.TrimSpace(req.Type),
		BaseURL:   strings.TrimSpace(req.BaseURL),
		ModelName: strings.TrimSpace(req.ModelName),
		Status:    1,
	}

	if req.APIKey != nil {
		config.APIKey = strings.TrimSpace(*req.APIKey)
	}
	if req.Status != nil {
		config.Status = int8(*req.Status)
	}

	if err := uc.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建配置失败"})
		return
	}

	// 不返回 api_key 明文
	config.APIKey = maskAPIKey(config.APIKey)

	c.JSON(http.StatusCreated, gin.H{"data": config})
}

// UpdateUserLLMConfig 更新用户LLM配置
func (uc *UserController) UpdateUserLLMConfig(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效用户上下文"})
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var existing models.UserLLMConfig
	if err := uc.DB.Where("id = ? AND user_id = ?", id, uid).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在或不属于当前用户"})
		return
	}

	var req UserLLMConfigPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.Type = strings.TrimSpace(req.Type)
	existing.BaseURL = strings.TrimSpace(req.BaseURL)
	existing.ModelName = strings.TrimSpace(req.ModelName)

	if req.APIKey != nil {
		existing.APIKey = strings.TrimSpace(*req.APIKey)
	}
	if req.Status != nil {
		existing.Status = int8(*req.Status)
	}

	if err := uc.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新配置失败"})
		return
	}

	// 不返回 api_key 明文
	existing.APIKey = maskAPIKey(existing.APIKey)

	c.JSON(http.StatusOK, gin.H{"data": existing})
}

// DeleteUserLLMConfig 删除用户LLM配置
func (uc *UserController) DeleteUserLLMConfig(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效用户上下文"})
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	result := uc.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&models.UserLLMConfig{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除配置失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在或不属于当前用户"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": nil})
}

// GetUserLLMConfigBalance 查询用户LLM配置的余额信息
func (uc *UserController) GetUserLLMConfigBalance(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效用户上下文"})
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var config models.UserLLMConfig
	if err := uc.DB.Where("id = ? AND user_id = ?", id, uid).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在或不属于当前用户"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"balance":      config.Balance,
			"used_tokens":  config.UsedTokens,
			"total_tokens": config.TotalTokens,
		},
	})
}

// maskAPIKey 将 API Key 脱敏，仅保留前4位和后4位
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

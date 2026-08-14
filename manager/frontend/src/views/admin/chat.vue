<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import {
  PROVIDER_LABELS,
  parseLLMConfigRow,
  requestModelDirect,
  formatRequestError,
  nowTime
} from '../../utils/llmDirectChat'

const conversations = ref([
  { id: 1, name: '示例会话 1', active: true },
  { id: 2, name: '示例会话 2', active: false },
  { id: 3, name: '示例会话 3', active: false },
])

const messages = ref([
  {
    id: 1,
    role: 'assistant',
    content: '你好！我是智能助手，有什么可以帮助你的吗？',
    time: '10:00',
  },
  {
    id: 2,
    role: 'user',
    content: '请介绍一下这个系统的功能。',
    time: '10:01',
  },
  {
    id: 3,
    role: 'assistant',
    content: '这是一个基于 ESP32 的智能语音交互系统，支持 ASR 语音识别、LLM 大模型对话、TTS 语音合成等功能，可以通过管理后台进行配置和管理。',
    time: '10:01',
  },
])

const inputText = ref('')
const isSending = ref(false)
const messageListRef = ref(null)
let nextId = 4

const hasMessages = computed(() => messages.value.length > 0)
const canSend = computed(
  () => inputText.value.trim().length > 0 && !isSending.value && !!selectedConfig.value
)

// ================= LLM 配置（来源：GET /admin/llm-configs） =================
const llmConfigs = ref([])
const selectedConfigId = ref('')
const loadingConfigs = ref(false)

const selectedConfig = computed(
  () => llmConfigs.value.find((item) => item.config_id === selectedConfigId.value) || null
)

const providerLabel = (provider) => PROVIDER_LABELS[provider] || provider || '未知提供商'

const loadLLMConfigs = async () => {
  loadingConfigs.value = true
  try {
    const response = await api.get('/admin/llm-configs')
    const rows = Array.isArray(response.data?.data) ? response.data.data : []
    // 仅保留 enabled === true 的配置
    llmConfigs.value = rows
      .map(parseLLMConfigRow)
      .filter((item) => item && item.enabled)
    // 默认选择：优先 is_default，其次第一个启用配置
    const target = llmConfigs.value.find((item) => item.is_default) || llmConfigs.value[0]
    selectedConfigId.value = target ? target.config_id : ''
  } catch (error) {
    llmConfigs.value = []
    selectedConfigId.value = ''
    ElMessage.error('加载 LLM 配置失败')
  } finally {
    loadingConfigs.value = false
  }
}

// ================= 直连模型 API（已提取到 utils/llmDirectChat.js） =================
onMounted(() => {
  loadLLMConfigs()
})

const selectConversation = (conv) => {
  conversations.value.forEach((c) => (c.active = false))
  conv.active = true
}

const handleSend = async () => {
  if (!canSend.value) return
  if (!selectedConfig.value) {
    ElMessage.warning('暂无可用的 LLM 配置，请先在 LLM 配置页面添加并启用配置')
    return
  }

  const content = inputText.value.trim()
  messages.value.push({
    id: nextId++,
    role: 'user',
    content,
    time: nowTime(),
  })
  inputText.value = ''
  scrollToBottom()

  const payloadMessages = messages.value
    .filter((msg) => msg.role === 'user' || msg.role === 'assistant')
    .map((msg) => ({ role: msg.role, content: msg.content }))

  isSending.value = true
  try {
    const reply = await requestModelDirect(selectedConfig.value, payloadMessages)
    messages.value.push({
      id: nextId++,
      role: 'assistant',
      content: reply || '（模型未返回有效内容）',
      time: nowTime(),
    })
  } catch (error) {
    messages.value.push({
      id: nextId++,
      role: 'assistant',
      content: `请求出错：${formatRequestError(error)}`,
      time: nowTime(),
    })
  } finally {
    isSending.value = false
    scrollToBottom()
  }
}

const handleKeydown = (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

const clearMessages = () => {
  messages.value = []
}

const resetExample = () => {
  messages.value = [
    {
      id: nextId++,
      role: 'assistant',
      content: '你好！我是智能助手，有什么可以帮助你的吗？',
      time: '10:00',
    },
    {
      id: nextId++,
      role: 'user',
      content: '请介绍一下这个系统的功能。',
      time: '10:01',
    },
    {
      id: nextId++,
      role: 'assistant',
      content: '这是一个基于 ESP32 的智能语音交互系统，支持 ASR 语音识别、LLM 大模型对话、TTS 语音合成等功能，可以通过管理后台进行配置和管理。',
      time: '10:01',
    },
  ]
  scrollToBottom()
}

const newConversation = () => {
  const id = conversations.value.length + 1
  conversations.value.push({
    id,
    name: `示例会话 ${id}`,
    active: true,
  })
  conversations.value.forEach((c) => {
    if (c.id !== id) c.active = false
  })
  messages.value = []
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messageListRef.value) {
      const el = messageListRef.value.$el || messageListRef.value
      if (el?.querySelector) {
        const wrap = el.querySelector('.el-scrollbar__wrap')
        if (wrap) wrap.scrollTop = wrap.scrollHeight
      }
    }
  })
}
</script>

<template>
  <div class="chat-page">
    <div class="chat-layout">
      <aside class="chat-sidebar">
        <div class="sidebar-header">
          <el-button type="primary" class="new-chat-btn" @click="newConversation">
            <el-icon><Plus /></el-icon>
            新建会话
          </el-button>
        </div>
        <el-scrollbar class="conversation-list">
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="conversation-item"
            :class="{ active: conv.active }"
            @click="selectConversation(conv)"
          >
            <el-icon class="conv-icon"><ChatDotRound /></el-icon>
            <span class="conv-name">{{ conv.name }}</span>
          </div>
        </el-scrollbar>
      </aside>

      <main class="chat-main">
        <header class="chat-header">
          <div class="header-left">
            <h2 class="page-title">聊天界面</h2>
            <p class="page-desc">基于已启用的 LLM 配置进行对话测试</p>
          </div>
          <div class="header-actions">
            <el-button size="small" @click="resetExample">
              <el-icon><Refresh /></el-icon>
              刷新示例
            </el-button>
            <el-button size="small" type="danger" plain @click="clearMessages" :disabled="!hasMessages">
              <el-icon><Delete /></el-icon>
              清空消息
            </el-button>
          </div>
        </header>

        <div class="model-bar">
          <span class="model-bar-label">模型配置</span>
          <el-select
            v-model="selectedConfigId"
            placeholder="请选择 LLM 配置"
            :loading="loadingConfigs"
            :disabled="loadingConfigs || llmConfigs.length === 0"
            class="model-select"
          >
            <el-option
              v-for="item in llmConfigs"
              :key="item.config_id"
              :label="item.label"
              :value="item.config_id"
            >
              <div class="model-option">
                <span class="model-option-name">{{ item.label }}</span>
                <el-tag v-if="item.is_default" size="small" type="warning" effect="plain">默认</el-tag>
              </div>
            </el-option>
          </el-select>
          <el-tag
            v-if="selectedConfig"
            size="small"
            effect="plain"
            type="info"
            class="model-base-url"
          >
            {{ selectedConfig.base_url || '未配置 Base URL' }}
          </el-tag>
          <el-alert
            v-if="!loadingConfigs && llmConfigs.length === 0"
            title="暂无可用的 LLM 配置"
            description="请先在 LLM 配置页面添加并启用至少一条配置"
            type="warning"
            :closable="false"
            show-icon
            class="no-config-alert"
          />
        </div>

        <div class="chat-body">
          <el-scrollbar ref="messageListRef" class="message-scrollbar">
            <div class="message-list">
              <el-empty
                v-if="!hasMessages"
                description="暂无消息，发送一条消息开始对话吧"
                :image-size="80"
              />
              <template v-else>
                <div
                  v-for="msg in messages"
                  :key="msg.id"
                  class="message-row"
                  :class="msg.role"
                >
                  <div class="message-avatar">
                    <el-avatar
                      :size="36"
                      :style="{
                        background: msg.role === 'user' ? '#409eff' : '#f0f2f5',
                        color: msg.role === 'user' ? '#fff' : '#606266',
                        fontSize: '14px'
                      }"
                    >
                      {{ msg.role === 'user' ? '我' : 'AI' }}
                    </el-avatar>
                  </div>
                  <div class="message-content">
                    <div class="message-meta">
                      <el-tag
                        :type="msg.role === 'user' ? 'primary' : 'info'"
                        size="small"
                        effect="plain"
                        class="role-tag"
                      >
                        {{ msg.role === 'user' ? '用户' : '助手' }}
                      </el-tag>
                      <span class="message-time">{{ msg.time }}</span>
                    </div>
                    <div class="message-bubble">{{ msg.content }}</div>
                  </div>
                </div>
              </template>
            </div>
          </el-scrollbar>
        </div>

        <footer class="chat-footer">
          <div class="input-area">
            <el-input
              v-model="inputText"
              type="textarea"
              :rows="2"
              placeholder="输入消息，按 Enter 发送，Shift+Enter 换行..."
              resize="none"
              @keydown="handleKeydown"
            />
            <div class="input-actions">
              <el-button
                size="small"
                text
                @click="inputText = ''"
                :disabled="!inputText"
              >
                清空输入
              </el-button>
              <el-button
                type="primary"
                :disabled="!canSend"
                :loading="isSending"
                @click="handleSend"
              >
                发送
              </el-button>
            </div>
          </div>
        </footer>
      </main>
    </div>
  </div>
</template>

<style scoped>
.chat-page {
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.chat-layout {
  display: flex;
  gap: 16px;
  height: calc(100vh - 140px);
  min-height: 500px;
}

/* Sidebar */
.chat-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #f0f0f0;
  overflow: hidden;
}

.sidebar-header {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.new-chat-btn {
  width: 100%;
}

.conversation-list {
  flex: 1;
}

.conversation-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
  border-bottom: 1px solid #f5f5f5;
}

.conversation-item:hover {
  background: #f0f7ff;
}

.conversation-item.active {
  background: #e8f4ff;
  border-left: 3px solid #409eff;
}

.conv-icon {
  color: #909399;
  font-size: 16px;
}

.conversation-item.active .conv-icon {
  color: #409eff;
}

.conv-name {
  font-size: 13px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Main Area */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.page-desc {
  margin: 0;
  font-size: 12px;
  color: #909399;
}

.header-actions {
  display: flex;
  gap: 8px;
}

/* Model Bar */
.model-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 12px 20px;
  border-bottom: 1px solid #f0f0f0;
  background: #fff;
}

.model-bar-label {
  font-size: 13px;
  font-weight: 500;
  color: #606266;
}

.model-select {
  width: 320px;
}

.model-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.model-option-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-base-url {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.no-config-alert {
  width: 100%;
}

/* Message Area */
.chat-body {
  flex: 1;
  overflow: hidden;
}

.message-scrollbar {
  height: 100%;
}

.message-list {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message-row {
  display: flex;
  gap: 12px;
  max-width: 80%;
}

.message-row.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.message-row.assistant {
  align-self: flex-start;
}

.message-avatar {
  flex-shrink: 0;
}

.message-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.message-row.user .message-content {
  align-items: flex-end;
}

.message-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.message-row.user .message-meta {
  flex-direction: row-reverse;
}

.role-tag {
  font-size: 11px;
}

.message-time {
  font-size: 11px;
  color: #c0c4cc;
}

.message-bubble {
  padding: 12px 16px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.message-row.assistant .message-bubble {
  background: #f5f7fa;
  color: #303133;
  border-top-left-radius: 4px;
}

.message-row.user .message-bubble {
  background: #409eff;
  color: #fff;
  border-top-right-radius: 4px;
}

/* Footer Input */
.chat-footer {
  padding: 16px 20px;
  border-top: 1px solid #f0f0f0;
  background: #fafafa;
}

.input-area {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.input-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.input-actions :deep(.el-textarea__inner) {
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}
</style>

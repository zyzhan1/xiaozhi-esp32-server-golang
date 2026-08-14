<template>
  <el-dialog
    :model-value="visible"
    title="对话记录"
    width="1100px"
    class="agent-chat-dialog"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="chat-dialog-header">
      <span class="header-label">当前智能体</span>
      <strong class="header-agent-name">{{ agentName || '未命名智能体' }}</strong>
    </div>

    <el-tabs v-model="activeTab" class="chat-tabs" @tab-change="handleTabChange">
      <!-- 历史记录 Tab -->
      <el-tab-pane label="历史记录" name="history">
        <div class="history-tab-content">
          <div class="history-toolbar">
            <span class="header-count" v-if="total > 0">共 {{ total }} 条消息</span>
            <el-button class="header-export-btn" @click="handleExport" :loading="exporting" size="small">
              <el-icon><Download /></el-icon>
              导出
            </el-button>
          </div>

          <!-- 筛选 -->
          <div class="filter-bar">
            <el-select v-model="filters.role" placeholder="角色" clearable size="small" style="width: 110px">
              <el-option label="全部" value="" />
              <el-option label="用户" value="user" />
              <el-option label="机器人" value="assistant" />
            </el-select>
            <el-select v-model="filters.device_id" placeholder="设备" clearable size="small" style="width: 140px">
              <el-option label="全部" value="" />
              <el-option v-for="d in devices" :key="d.id" :label="d.device_name || d.device_code" :value="d.device_name" />
            </el-select>
            <el-date-picker v-model="filters.start_date" type="date" placeholder="开始日期" format="YYYY-MM-DD" value-format="YYYY-MM-DD" size="small" style="width: 140px" clearable />
            <el-date-picker v-model="filters.end_date" type="date" placeholder="结束日期" format="YYYY-MM-DD" value-format="YYYY-MM-DD" size="small" style="width: 140px" clearable />
            <el-button type="primary" size="small" @click="handleSearch">查询</el-button>
            <el-button size="small" @click="handleReset">重置</el-button>
          </div>

          <!-- 消息列表 -->
          <div class="chat-container" v-loading="loading">
            <div v-if="messages.length === 0 && !loading" class="empty-state">
              <el-empty description="暂无聊天记录" :image-size="80" />
            </div>
            <div v-else class="chat-messages" ref="chatMessagesRef">
              <div v-for="(message, index) in messages" :key="message.id" class="message-wrapper" :class="{ 'message-right': message.role === 'user', 'message-left': message.role === 'assistant' }">
                <div v-if="shouldShowTime(message, index)" class="message-time-divider">
                  {{ formatTimeShort(message.created_at) }}
                </div>
                <div class="message-bubble-wrapper">
                  <template v-if="message.role === 'assistant'">
                    <div class="message-bubble message-bubble-left">
                      <div class="message-content-wrapper">
                        <div v-if="message.content" class="message-text">{{ message.content }}</div>
                        <div v-if="message.audio_path" class="audio-bubble">
                          <audio :ref="el => audioRefs[message.id] = el" :src="audioBlobUrls[message.id]" @ended="handleAudioEnded(message.id)" @error="handleAudioError(message.id)" />
                          <el-button :icon="playingAudioId === message.id ? VideoPause : VideoPlay" circle size="small" @click="toggleAudio(message.id)" class="audio-play-btn-simple" />
                        </div>
                        <div class="message-meta">
                          <span class="message-time-small">{{ formatTimeShort(message.created_at) }}</span>
                          <el-dropdown trigger="click" @command="handleMessageAction">
                            <el-icon class="message-more"><MoreFilled /></el-icon>
                            <template #dropdown>
                              <el-dropdown-menu>
                                <el-dropdown-item :command="{action: 'delete', id: message.id}">删除</el-dropdown-item>
                              </el-dropdown-menu>
                            </template>
                          </el-dropdown>
                        </div>
                      </div>
                    </div>
                  </template>
                  <template v-else>
                    <div class="message-bubble message-bubble-right">
                      <div class="message-content-wrapper">
                        <div v-if="message.content" class="message-text">{{ message.content }}</div>
                        <div v-if="message.audio_path" class="audio-bubble">
                          <audio :ref="el => audioRefs[message.id] = el" :src="audioBlobUrls[message.id]" @ended="handleAudioEnded(message.id)" @error="handleAudioError(message.id)" />
                          <el-button :icon="playingAudioId === message.id ? VideoPause : VideoPlay" circle size="small" @click="toggleAudio(message.id)" class="audio-play-btn-simple" />
                        </div>
                        <div class="message-meta">
                          <el-dropdown trigger="click" @command="handleMessageAction">
                            <el-icon class="message-more"><MoreFilled /></el-icon>
                            <template #dropdown>
                              <el-dropdown-menu>
                                <el-dropdown-item :command="{action: 'delete', id: message.id}">删除</el-dropdown-item>
                              </el-dropdown-menu>
                            </template>
                          </el-dropdown>
                          <span class="message-time-small">{{ formatTimeShort(message.created_at) }}</span>
                        </div>
                      </div>
                    </div>
                  </template>
                </div>
              </div>
            </div>

            <!-- 分页 -->
            <div class="pagination" v-if="total > 0">
              <el-pagination
                v-model:current-page="pagination.page"
                v-model:page-size="pagination.pageSize"
                :total="total"
                :page-sizes="[20, 50, 100]"
                layout="total, sizes, prev, pager, next"
                size="small"
                @size-change="handleSizeChange"
                @current-change="handlePageChange"
              />
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 实时聊天 Tab -->
      <el-tab-pane label="实时聊天" name="live">
        <div class="live-chat-content">
          <!-- 模型信息 -->
          <div class="live-model-info" v-if="agentLLMConfig">
            <el-tag type="success" effect="plain" size="small">
              {{ agentLLMConfig.model_name }} ({{ agentLLMConfig.provider }})
            </el-tag>
            <el-tag type="info" effect="plain" size="small" class="model-url-tag">
              {{ agentLLMConfig.base_url || '未配置 Base URL' }}
            </el-tag>
          </div>
          <el-alert
            v-else-if="!loadingLLMConfig"
            title="该智能体未绑定 LLM 配置"
            description="请先在智能体配置中绑定 LLM 模型，才能使用实时聊天功能"
            type="warning"
            :closable="false"
            show-icon
          />

          <!-- 聊天消息区 -->
          <div class="live-chat-container">
            <div v-if="liveMessages.length === 0" class="empty-state">
              <el-empty description="暂无消息，发送一条消息开始对话吧" :image-size="80" />
            </div>
            <div v-else class="live-chat-messages" ref="liveChatMessagesRef">
              <div
                v-for="msg in liveMessages"
                :key="msg.id"
                class="live-message-row"
                :class="msg.role"
              >
                <div class="live-message-avatar">
                  <el-avatar
                    :size="36"
                    :style="{
                      background: msg.role === 'user' ? '#007aff' : '#f0f2f5',
                      color: msg.role === 'user' ? '#fff' : '#606266',
                      fontSize: '14px'
                    }"
                  >
                    {{ msg.role === 'user' ? '我' : 'AI' }}
                  </el-avatar>
                </div>
                <div class="live-message-content">
                  <div class="live-message-meta">
                    <el-tag
                      :type="msg.role === 'user' ? 'primary' : 'info'"
                      size="small"
                      effect="plain"
                      class="live-role-tag"
                    >
                      {{ msg.role === 'user' ? '用户' : '助手' }}
                    </el-tag>
                    <span class="live-message-time">{{ msg.time }}</span>
                  </div>
                  <div class="live-message-bubble" :class="{ 'is-thinking': msg.thinking }">
                    <template v-if="msg.thinking">
                      <span class="thinking-dots">
                        <span class="thinking-dot"></span>
                        <span class="thinking-dot"></span>
                        <span class="thinking-dot"></span>
                      </span>
                    </template>
                    <template v-else>{{ msg.content }}</template>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 输入区 -->
          <div class="live-chat-footer">
            <div class="live-input-area">
              <el-input
                v-model="liveInputText"
                type="textarea"
                :rows="2"
                placeholder="输入消息，按 Enter 发送，Shift+Enter 换行..."
                resize="none"
                @keydown="handleLiveKeydown"
                :disabled="!agentLLMConfig"
              />
              <div class="live-input-actions">
                <el-button
                  size="small"
                  text
                  @click="liveInputText = ''"
                  :disabled="!liveInputText || !agentLLMConfig"
                >
                  清空输入
                </el-button>
                <el-button
                  size="small"
                  type="danger"
                  plain
                  @click="clearLiveMessages"
                  :disabled="liveMessages.length === 0"
                >
                  清空消息
                </el-button>
                <el-button
                  type="primary"
                  :disabled="!canLiveSend"
                  :loading="isLiveSending"
                  @click="handleLiveSend"
                >
                  发送
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, onBeforeUnmount, nextTick, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, VideoPlay, VideoPause, MoreFilled } from '@element-plus/icons-vue'
import userApi from '@/utils/userApi'
import {
  parseLLMConfigRow,
  requestModelDirect,
  formatRequestError,
  nowTime
} from '@/utils/llmDirectChat'

const props = defineProps({
  visible: { type: Boolean, default: false },
  agentId: { type: [Number, String], default: null }
})

const emit = defineEmits(['update:visible'])

const agentId = computed(() => props.agentId ? String(props.agentId) : null)
const agentName = ref('')
const activeTab = ref('live')

// ================= 历史记录 Tab 相关 =================
const loading = ref(false)
const exporting = ref(false)
const messages = ref([])
const total = ref(0)
const devices = ref([])

const filters = reactive({ role: '', device_id: '', start_date: '', end_date: '' })
const pagination = reactive({ page: 1, pageSize: 50 })

// 音频
const audioRefs = ref({})
const playingAudioId = ref(null)
const chatMessagesRef = ref(null)
const audioBlobUrls = ref({})

// ================= 实时聊天 Tab 相关 =================
const agentLLMConfig = ref(null)
const loadingLLMConfig = ref(false)
const liveMessages = ref([])
const liveInputText = ref('')
const isLiveSending = ref(false)
const liveChatMessagesRef = ref(null)
let liveNextId = 1

const canLiveSend = computed(
  () => liveInputText.value.trim().length > 0 && !isLiveSending.value && !!agentLLMConfig.value
)

// ================= 智能体加载 =================
const loadAgent = async () => {
  if (!agentId.value) return
  try {
    const response = await userApi.get(`/user/agents/${agentId.value}`)
    agentName.value = response.data.data?.name || '智能体'
  } catch { /* silent */ }
}

const loadDevices = async () => {
  if (!agentId.value) return
  try {
    const response = await userApi.get(`/user/agents/${agentId.value}/devices`)
    devices.value = response.data.data || []
  } catch { /* silent */ }
}

// ================= 获取智能体 LLM 配置 =================
const loadAgentLLMConfig = async () => {
  if (!agentId.value) return
  loadingLLMConfig.value = true
  agentLLMConfig.value = null
  try {
    // 获取智能体详情，从中提取 llm_config 字段
    const agentRes = await userApi.get(`/user/agents/${agentId.value}`)
    const agentData = agentRes.data.data
    if (!agentData) return

    const llmConfig = agentData.llm_config
    if (!llmConfig) {
      console.warn('该智能体未绑定 LLM 配置')
      return
    }

    // llm_config 结构与 admin/llm-configs 条目一致：
    // 顶层含 config_id、provider、name、id、enabled 等
    // json_data 为 JSON 字符串，内含 model_name、api_key、base_url、max_tokens 等
    // 直接复用 parseLLMConfigRow 完成 json_data 解析、provider 标准化、base_url 兜底等
    agentLLMConfig.value = parseLLMConfigRow(llmConfig)
  } catch (error) {
    console.warn('获取智能体 LLM 配置失败:', error)
  } finally {
    loadingLLMConfig.value = false
  }
}

// ================= 历史记录功能 =================
const loadMessages = async () => {
  if (!agentId.value) return
  loading.value = true
  try {
    const params = { page: pagination.page, page_size: pagination.pageSize }
    if (filters.role) params.role = filters.role
    if (filters.device_id) params.device_id = filters.device_id
    if (filters.start_date) params.start_date = filters.start_date
    if (filters.end_date) params.end_date = filters.end_date

    const response = await userApi.get(`/user/history/agents/${agentId.value}/messages`, { params })
    const data = response.data.data || []
    messages.value = [...data].reverse()
    total.value = response.data.total || 0
    await preloadAudioMessages()
    await nextTick()
    scrollToBottom()
  } catch (error) {
    ElMessage.error('加载消息失败: ' + (error.response?.data?.error || error.message))
    messages.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { pagination.page = 1; loadMessages() }
const handleReset = () => {
  filters.role = ''; filters.device_id = ''; filters.start_date = ''; filters.end_date = ''
  pagination.page = 1; loadMessages()
}
const handlePageChange = () => { loadMessages() }
const handleSizeChange = () => { pagination.page = 1; loadMessages() }

const handleDelete = async (messageId) => {
  try {
    await ElMessageBox.confirm('确定要删除这条消息吗？', '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
    await userApi.delete(`/user/history/messages/${messageId}`)
    ElMessage.success('删除成功')
    loadMessages()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error('删除失败')
  }
}

const handleExport = async () => {
  exporting.value = true
  try {
    const params = { agent_id: agentId.value }
    if (filters.role) params.role = filters.role
    if (filters.device_id) params.device_id = filters.device_id
    if (filters.start_date) params.start_date = filters.start_date
    if (filters.end_date) params.end_date = filters.end_date
    const response = await userApi.get('/user/history/export', { params, responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `chat_history_${new Date().toISOString().slice(0, 10)}.json`)
    document.body.appendChild(link); link.click(); link.remove()
    window.URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch { ElMessage.error('导出失败') }
  finally { exporting.value = false }
}

const handleMessageAction = (command) => {
  if (command.action === 'delete') handleDelete(command.id)
}

// ================= 实时聊天功能 =================
const handleLiveSend = async () => {
  if (!canLiveSend.value) return
  if (!agentLLMConfig.value) {
    ElMessage.warning('该智能体未绑定 LLM 配置，无法发送消息')
    return
  }

  const content = liveInputText.value.trim()
  liveMessages.value.push({
    id: liveNextId++,
    role: 'user',
    content,
    time: nowTime()
  })
  liveInputText.value = ''

  // 追加"正在思考"占位消息
  const thinkingId = liveNextId++
  liveMessages.value.push({
    id: thinkingId,
    role: 'assistant',
    content: '',
    time: nowTime(),
    thinking: true
  })
  scrollLiveToBottom()

  // 构建历史消息时排除占位消息（thinking 标识）
  const payloadMessages = liveMessages.value
    .filter((msg) => (msg.role === 'user' || msg.role === 'assistant') && !msg.thinking)
    .map((msg) => ({ role: msg.role, content: msg.content }))

  isLiveSending.value = true
  try {
    const reply = await requestModelDirect(agentLLMConfig.value, payloadMessages)
    // 将占位消息更新为真实回复
    const thinkingMsg = liveMessages.value.find((m) => m.id === thinkingId)
    if (thinkingMsg) {
      thinkingMsg.content = reply || '（模型未返回有效内容）'
      thinkingMsg.thinking = false
    }
  } catch (error) {
    // 将占位消息更新为错误提示
    const thinkingMsg = liveMessages.value.find((m) => m.id === thinkingId)
    if (thinkingMsg) {
      thinkingMsg.content = `请求出错：${formatRequestError(error)}`
      thinkingMsg.thinking = false
    }
  } finally {
    isLiveSending.value = false
    scrollLiveToBottom()
  }
}

const handleLiveKeydown = (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleLiveSend()
  }
}

const clearLiveMessages = () => {
  liveMessages.value = []
}

const scrollLiveToBottom = () => {
  nextTick(() => {
    if (liveChatMessagesRef.value) {
      liveChatMessagesRef.value.scrollTop = liveChatMessagesRef.value.scrollHeight
    }
  })
}

// ================= Tab 切换 =================
const handleTabChange = (tab) => {
  if (tab === 'live' && !agentLLMConfig.value && !loadingLLMConfig.value) {
    loadAgentLLMConfig()
  }
}

// ================= 时间格式化 =================
const formatTimeShort = (dateString) => {
  const date = new Date(dateString)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const msgDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  if (msgDate.getTime() === today.getTime()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  const yesterday = new Date(today); yesterday.setDate(yesterday.getDate() - 1)
  if (msgDate.getTime() === yesterday.getTime()) return '昨天 ' + date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1}月${date.getDate()}日 ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  }
  return date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const shouldShowTime = (message, index) => {
  if (index === 0) return true
  return (new Date(message.created_at).getTime() - new Date(messages.value[index - 1].created_at).getTime()) > 5 * 60 * 1000
}

// ================= 滚动 =================
const scrollToBottom = () => {
  if (chatMessagesRef.value) {
    nextTick(() => { chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight })
  }
}

// ================= 音频 =================
const getAudioUrl = async (messageId) => {
  if (audioBlobUrls.value[messageId]) return audioBlobUrls.value[messageId]
  try {
    const response = await userApi.get(`/user/history/messages/${messageId}/audio`, { responseType: 'blob' })
    const blobUrl = URL.createObjectURL(response.data)
    audioBlobUrls.value[messageId] = blobUrl
    return blobUrl
  } catch (error) { console.warn('加载音频失败:', messageId, error); return null }
}

const preloadAudioMessages = async () => {
  const audioMessages = messages.value.filter(msg => msg.audio_path)
  await Promise.all(audioMessages.slice(0, 10).map(msg => getAudioUrl(msg.id).catch(() => null)))
}

const handleAudioEnded = (messageId) => { playingAudioId.value = null }
const handleAudioError = async (messageId) => {
  console.warn('音频加载失败:', messageId)
  try { const url = await getAudioUrl(messageId); if (url) { const audio = audioRefs.value[messageId]; if (audio) audio.load() } } catch { /* silent */ }
}

const toggleAudio = async (messageId) => {
  const audio = audioRefs.value[messageId]
  if (!audio) return
  if (!audioBlobUrls.value[messageId]) {
    const url = await getAudioUrl(messageId)
    if (!url) return
    await new Promise((resolve) => { audio.onloadeddata = resolve; audio.load() })
  }
  if (playingAudioId.value && playingAudioId.value !== messageId) {
    const other = audioRefs.value[playingAudioId.value]
    if (other) { other.pause(); other.currentTime = 0 }
  }
  if (playingAudioId.value === messageId) { audio.pause(); playingAudioId.value = null }
  else { try { await audio.play(); playingAudioId.value = messageId } catch { /* silent */ } }
}

// ================= 生命周期 =================
const handleClose = () => {
  activeTab.value = 'live'
  emit('update:visible', false)
}

watch(() => props.visible, (val) => {
  if (val && props.agentId) {
    messages.value = []; total.value = 0; agentName.value = ''
    filters.role = ''; filters.device_id = ''; filters.start_date = ''; filters.end_date = ''
    pagination.page = 1; pagination.pageSize = 50
    activeTab.value = 'live'
    liveMessages.value = []
    liveInputText.value = ''
    agentLLMConfig.value = null
    Promise.all([loadAgent(), loadDevices(), loadMessages(), loadAgentLLMConfig()])
  }
})

onBeforeUnmount(() => {
  Object.values(audioBlobUrls.value).forEach(url => { if (url) URL.revokeObjectURL(url) })
  audioBlobUrls.value = {}
})
</script>

<style scoped>
.chat-dialog-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}
.header-label { color: var(--apple-text-secondary); font-size: 13px; }
.header-agent-name { color: var(--apple-text); font-size: 16px; }

/* ================= Tabs ================= */
.chat-tabs {
  margin-top: 8px;
}
.chat-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}

/* ================= 历史记录 Tab ================= */
.history-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.header-count { color: var(--apple-text-secondary); font-size: 12px; margin-left: auto; }
.header-export-btn { margin-left: 8px; }

.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 14px;
  flex-wrap: wrap;
  align-items: center;
}

.chat-container {
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  border-radius: 18px;
  overflow: hidden;
  min-height: 400px;
}

.chat-messages {
  padding: 16px;
  max-height: 50vh;
  overflow-y: auto;
}

.empty-state { padding: 40px 0; }

.message-wrapper { display: flex; flex-direction: column; margin-bottom: 14px; }
.message-time-divider { text-align: center; margin: 12px 0; font-size: 11px; color: var(--apple-text-tertiary); }
.message-bubble-wrapper { display: flex; align-items: flex-start; max-width: 75%; }
.message-right { margin-left: auto; justify-content: flex-end; width: 100%; display: flex; }
.message-left { margin-right: auto; justify-content: flex-start; width: 100%; display: flex; }

.message-bubble {
  position: relative; padding: 10px 14px; border-radius: 18px;
  word-wrap: break-word; word-break: break-word;
  box-shadow: 0 8px 16px rgba(15, 23, 42, 0.05); max-width: 100%;
}
.message-bubble-left { background: rgba(255, 255, 255, 0.94); border-top-left-radius: 8px; }
.message-bubble-right { background: rgba(0, 122, 255, 0.12); border: 1px solid rgba(0, 122, 255, 0.16); border-top-right-radius: 8px; margin-left: auto; }

.message-content-wrapper { display: flex; flex-direction: column; gap: 6px; }
.message-text { color: var(--apple-text); line-height: 1.5; white-space: pre-wrap; word-break: break-word; font-size: 13px; }

.audio-bubble { margin: 4px 0; display: flex; align-items: center; }
.audio-play-btn-simple { flex-shrink: 0; }

.message-meta { display: flex; align-items: center; gap: 6px; margin-top: 2px; opacity: 0.7; }
.message-meta:hover { opacity: 1; }
.message-time-small { font-size: 11px; color: var(--apple-text-tertiary); }
.message-more { font-size: 14px; color: var(--apple-text-tertiary); cursor: pointer; padding: 2px; border-radius: 8px; }
.message-more:hover { background: rgba(0, 122, 255, 0.08); color: var(--apple-primary); }

.pagination {
  padding: 12px;
  display: flex;
  justify-content: center;
  background: rgba(255, 255, 255, 0.88);
  border-top: 1px solid rgba(229, 229, 234, 0.72);
}

.chat-messages::-webkit-scrollbar { width: 5px; }
.chat-messages::-webkit-scrollbar-track { background: rgba(229, 229, 234, 0.52); border-radius: 3px; }
.chat-messages::-webkit-scrollbar-thumb { background: rgba(142, 142, 147, 0.58); border-radius: 3px; }

/* ================= 实时聊天 Tab ================= */
.live-chat-content {
  display: flex;
  flex-direction: column;
  height: 65vh;
  min-height: 450px;
}

.live-model-info {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  border-radius: 12px;
  margin-bottom: 14px;
}
.model-url-tag {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.live-chat-container {
  flex: 1;
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  border-radius: 18px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.live-chat-messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.live-message-row {
  display: flex;
  gap: 12px;
  max-width: 80%;
}
.live-message-row.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}
.live-message-row.assistant {
  align-self: flex-start;
}

.live-message-avatar {
  flex-shrink: 0;
}

.live-message-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.live-message-row.user .live-message-content {
  align-items: flex-end;
}

.live-message-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}
.live-message-row.user .live-message-meta {
  flex-direction: row-reverse;
}

.live-role-tag {
  font-size: 11px;
}

.live-message-time {
  font-size: 11px;
  color: #c0c4cc;
}

.live-message-bubble {
  padding: 12px 16px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  white-space: pre-wrap;
}
.live-message-row.assistant .live-message-bubble {
  background: #f5f7fa;
  color: #303133;
  border-top-left-radius: 4px;
}
.live-message-row.user .live-message-bubble {
  background: #007aff;
  color: #fff;
  border-top-right-radius: 4px;
}

/* ================= 正在思考动画 ================= */
.thinking-dots {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
}
.thinking-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #909399;
  animation: thinking-bounce 1.4s infinite ease-in-out both;
}
.thinking-dot:nth-child(1) { animation-delay: 0s; }
.thinking-dot:nth-child(2) { animation-delay: 0.16s; }
.thinking-dot:nth-child(3) { animation-delay: 0.32s; }
@keyframes thinking-bounce {
  0%, 80%, 100% { transform: scale(0.6); opacity: 0.4; }
  40% { transform: scale(1); opacity: 1; }
}

.live-message-bubble.is-thinking {
  padding: 10px 18px;
}

.live-chat-messages::-webkit-scrollbar { width: 5px; }
.live-chat-messages::-webkit-scrollbar-track { background: rgba(229, 229, 234, 0.52); border-radius: 3px; }
.live-chat-messages::-webkit-scrollbar-thumb { background: rgba(142, 142, 147, 0.58); border-radius: 3px; }

.live-chat-footer {
  padding: 14px 0 0;
  border-top: 1px solid rgba(229, 229, 234, 0.72);
  margin-top: 14px;
}

.live-input-area {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.live-input-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
}
</style>

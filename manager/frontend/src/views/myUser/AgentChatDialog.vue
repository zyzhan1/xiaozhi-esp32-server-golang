<template>
  <el-dialog
    :model-value="visible"
    title="对话记录"
    width="860px"
    class="agent-chat-dialog"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="chat-dialog-header">
      <span class="header-label">当前智能体</span>
      <strong class="header-agent-name">{{ agentName || '未命名智能体' }}</strong>
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
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, onBeforeUnmount, nextTick, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, VideoPlay, VideoPause, MoreFilled } from '@element-plus/icons-vue'
import userApi from '@/utils/userApi'

const props = defineProps({
  visible: { type: Boolean, default: false },
  agentId: { type: [Number, String], default: null }
})

const emit = defineEmits(['update:visible'])

const agentId = computed(() => props.agentId ? String(props.agentId) : null)
const agentName = ref('')
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

// 时间格式化
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

// 滚动
const scrollToBottom = () => {
  if (chatMessagesRef.value) {
    nextTick(() => { chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight })
  }
}

// 音频
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

const handleClose = () => { emit('update:visible', false) }

watch(() => props.visible, (val) => {
  if (val && props.agentId) {
    messages.value = []; total.value = 0; agentName.value = ''
    filters.role = ''; filters.device_id = ''; filters.start_date = ''; filters.end_date = ''
    pagination.page = 1; pagination.pageSize = 50
    Promise.all([loadAgent(), loadDevices(), loadMessages()])
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
  max-height: 55vh;
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
</style>

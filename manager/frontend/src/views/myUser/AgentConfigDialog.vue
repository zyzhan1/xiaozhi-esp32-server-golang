<template>
  <el-dialog
    :model-value="visible"
    title="配置智能体"
    width="720px"
    class="agent-config-dialog"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div v-loading="loadingAgent">
      <!-- 角色选择条 -->
      <div class="role-strip" v-loading="rolesLoading">
        <button
          v-for="role in allRoles"
          :key="role.id"
          type="button"
          class="role-chip"
          :class="{ active: selectedRoleId === role.id }"
          @click="applyRoleConfig(role)"
        >
          <span>{{ role.name }}</span>
          <small>{{ role.role_type === 'global' ? '全局' : '我的' }}</small>
        </button>
        <span v-if="!rolesLoading && allRoles.length === 0" class="role-empty">暂无可用角色</span>
      </div>

      <!-- 智能体表单 -->
      <div class="form-card">
        <AgentForm ref="agentFormRef" v-model="form" mode="edit" />
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          {{ saving ? '保存中...' : '保存配置' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import userApi from '@/utils/userApi'
import AgentForm from '@/components/common/AgentForm.vue'
import { agentToForm, createDefaultAgentForm } from '@/composables/useAgentFormOptions'

const props = defineProps({
  visible: { type: Boolean, default: false },
  agentId: { type: [Number, String], default: null }
})

const emit = defineEmits(['update:visible', 'saved'])

const form = ref(createDefaultAgentForm())
const agentFormRef = ref(null)
const saving = ref(false)
const loadingAgent = ref(false)
const rolesLoading = ref(false)
const selectedRoleId = ref(null)
const applyingRoleConfig = ref(false)
const globalRoles = ref([])
const userRoles = ref([])

const isRoleEnabled = (role) => role?.status === 'active' || !role?.status
const allRoles = computed(() => [...globalRoles.value, ...userRoles.value].filter(isRoleEnabled))

const loadAgent = async () => {
  if (!props.agentId) return
  loadingAgent.value = true
  try {
    const response = await userApi.get(`/user/agents/${props.agentId}`)
    form.value = agentToForm(response.data.data || {})
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '加载智能体配置失败')
  } finally {
    loadingAgent.value = false
  }
}

const loadRoles = async () => {
  rolesLoading.value = true
  try {
    const response = await userApi.get('/user/roles')
    globalRoles.value = response.data.data?.global_roles || []
    userRoles.value = response.data.data?.user_roles || []
  } catch (error) {
    globalRoles.value = []
    userRoles.value = []
  } finally {
    rolesLoading.value = false
  }
}

const applyRoleConfig = async (role) => {
  if (!role) return
  applyingRoleConfig.value = true
  try {
    selectedRoleId.value = role.id
    await agentFormRef.value?.reloadOptions?.()
    form.value.custom_prompt = role.prompt || ''

    if (role.llm_config_id && agentFormRef.value?.hasLlmConfig?.(role.llm_config_id)) {
      form.value.llm_config_id = role.llm_config_id
    }

    if (role.tts_config_id && agentFormRef.value?.hasTtsConfig?.(role.tts_config_id)) {
      await agentFormRef.value?.setTtsConfig?.(role.tts_config_id, { clearInvalid: true })
    } else {
      await agentFormRef.value?.setTtsConfig?.(null, { clearInvalid: true })
    }

    form.value.voice = role.voice || null
    ElMessage.info('已填充角色配置，请点击"保存配置"提交')
  } finally {
    applyingRoleConfig.value = false
  }
}

const handleSave = async () => {
  if (applyingRoleConfig.value) {
    ElMessage.info('当前正在填充角色配置，请稍后保存')
    return
  }
  if (!agentFormRef.value) return
  const valid = await agentFormRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    await userApi.put(`/user/agents/${props.agentId}`, agentFormRef.value.buildPayload())
    ElMessage.success('保存成功')
    emit('update:visible', false)
    emit('saved')
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

const handleClose = () => {
  emit('update:visible', false)
}

// 弹窗打开时加载数据
watch(() => props.visible, (val) => {
  if (val && props.agentId) {
    form.value = createDefaultAgentForm()
    selectedRoleId.value = null
    Promise.all([loadAgent(), loadRoles()])
  }
})
</script>

<style scoped>
.role-strip {
  min-height: 42px;
  margin-bottom: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  padding: 2px 0 6px;
}

.role-chip {
  border: 1px solid rgba(0, 122, 255, 0.18);
  border-radius: 8px;
  background: #fff;
  color: var(--apple-text, #1d1d1f);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: none;
  font-size: 13px;
  line-height: 1;
  padding: 9px 11px;
}

.role-chip small {
  color: var(--apple-text-secondary, #6b7280);
}

.role-chip.active {
  border-color: #409eff;
  background: #ecf5ff;
  color: #1677d2;
}

.role-empty {
  color: var(--apple-text-secondary, #6b7280);
  font-size: 13px;
}

.form-card {
  padding: 0;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>

<template>
  <div class="user-layout">
    <!-- 左侧菜单 -->
    <aside class="user-sidebar">
      <div class="sidebar-card apple-surface">
        <div class="brand-panel">
          <img class="brand-mark" :src="appLogo" alt="Zuto-AI Cloud" />
          <div class="brand-copy">
            <h3>Zuto-AI Cloud</h3>
            <p>用户工作台</p>
          </div>
        </div>

        <div class="sidebar-meta">
          <span class="apple-chip is-primary">用户模式</span>
          <span class="apple-chip is-success">在线中</span>
        </div>

        <el-scrollbar class="sidebar-scroll">
          <el-menu
            :default-active="activeMenu"
            class="sidebar-menu"
            unique-opened
            :collapse-transition="false"
            @select="handleMenuSelect"
          >
            <el-menu-item index="agents">
              <el-icon><Connection /></el-icon>
              <span>智能体管理</span>
            </el-menu-item>

            <el-menu-item index="devices">
              <el-icon><Iphone /></el-icon>
              <span>设备列表</span>
            </el-menu-item>

            <el-menu-item index="roles">
              <el-icon><User /></el-icon>
              <span>我的角色</span>
            </el-menu-item>

            <el-menu-item index="speakers">
              <el-icon><Microphone /></el-icon>
              <span>声纹管理</span>
            </el-menu-item>

            <el-menu-item index="voice-clones">
              <el-icon><Microphone /></el-icon>
              <span>声音复刻</span>
            </el-menu-item>

            <el-menu-item index="knowledge-bases">
              <el-icon><Document /></el-icon>
              <span>我的知识库</span>
            </el-menu-item>
          </el-menu>
        </el-scrollbar>
      </div>
    </aside>

    <!-- 右侧内容区 -->
    <div class="user-content">
      <header class="content-header">
        <div class="header-left">
          <span class="eyebrow">User Workspace</span>
          <h2 class="page-title">{{ menuTitleMap[activeMenu] || '智能体管理' }}</h2>
        </div>
        <div class="header-right">
          <span class="profile-avatar">{{ usernameInitial }}</span>
          <span class="profile-username">{{ currentUser.username || '--' }}</span>
        </div>
      </header>

      <main class="main-area">
        <component
          :is="menuComponentMap[activeMenu]"
          v-if="menuComponentMap[activeMenu]"
          :key="menuComponentKey"
        />
        <div v-else class="placeholder-page">
          <el-empty :description="`${menuTitleMap[activeMenu] || ''}功能开发中...`" />
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, provide } from 'vue'
import { ElMessage } from 'element-plus'
import {
  User,
  Iphone,
  Connection,
  Microphone,
  Document
} from '@element-plus/icons-vue'
import { clearUserAuth, isUserLoggedIn, getUserInfo } from './utils/userAuth.js'
import UserAgents from './views/myUser/UserAgents.vue'
import UserRoles from './views/myUser/UserRoles.vue'
import UserDevices from './views/myUser/UserDevices.vue'
import UserSpeakers from './views/myUser/UserSpeakers.vue'
import UserVoiceClones from './views/myUser/UserVoiceClones.vue'
import UserKnowledgeBases from './views/myUser/UserKnowledgeBases.vue'
import appLogo from '@/assets/brand/zutoAicloud.png'

// 智能体筛选状态（供设备列表组件注入）
const pendingDeviceAgentId = ref('')

provide('navigateToDevices', (agentId) => {
  pendingDeviceAgentId.value = agentId || ''
  activeMenu.value = 'devices'
  menuComponentKey.value++
  const hash = window.location.hash || ''
  const pathPart = hash.split('?')[0] || '#/'
  window.location.hash = `${pathPart}?menu=devices`
})

provide('pendingDeviceAgentId', pendingDeviceAgentId)

const menuTitleMap = {
  agents: '智能体管理',
  devices: '设备列表',
  roles: '我的角色',
  speakers: '声纹管理',
  'voice-clones': '声音复刻',
  'knowledge-bases': '我的知识库'
}

const menuComponentMap = {
  agents: UserAgents,
  devices: UserDevices,
  roles: UserRoles,
  speakers: UserSpeakers,
  'voice-clones': UserVoiceClones,
  'knowledge-bases': UserKnowledgeBases
}

// ---- 从 URL hash 中解析菜单状态 ----
// createWebHashHistory 下完整 hash 形如：#/user/agents?menu=roles
// Vue Router 可能不会正确解析 hash 内的 query，因此直接解析 window.location.hash
const parseHashMenu = () => {
  const hash = window.location.hash || ''
  const queryIndex = hash.indexOf('?')
  if (queryIndex === -1) return null
  const params = new URLSearchParams(hash.slice(queryIndex))
  return params.get('menu')
}

const validMenuKeys = Object.keys(menuComponentMap)
const hashMenu = parseHashMenu()
const initialMenu = (hashMenu && validMenuKeys.includes(hashMenu)) ? hashMenu : 'agents'

const activeMenu = ref(initialMenu)
const currentUser = ref({})
const menuComponentKey = ref(0)

const usernameInitial = computed(() => {
  const username = currentUser.value?.username || 'U'
  return username.slice(0, 1).toUpperCase()
})

const handleMenuSelect = (index) => {
  activeMenu.value = index
  menuComponentKey.value++
  const hash = window.location.hash || ''
  const pathPart = hash.split('?')[0] || '#/'
  const newHash = `${pathPart}?menu=${index}`
  window.location.hash = newHash
}

const handleLogout = () => {
  clearUserAuth()
  currentUser.value = {}
  ElMessage.success('已退出登录')
  // 刷新页面回到登录态检查
  window.location.reload()
}

onMounted(() => {
  // 直接从 localStorage 读取登录信息（管理员已提前写入）
  if (isUserLoggedIn()) {
    currentUser.value = getUserInfo() || {}
    return
  }

  // 没有登录信息，提示需要从管理员页面跳转
  ElMessage.warning('缺少登录凭证，请从管理员页面跳转访问')
})
</script>

<style scoped>
.user-layout {
  height: 100dvh;
  min-height: 0;
  padding: 20px;
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 16px;
  overflow: hidden;
  background: var(--apple-bg, #f5f5f7);
}

/* 左侧菜单 */
.user-sidebar {
  min-width: 0;
}

.sidebar-card {
  height: calc(100dvh - 40px);
  padding: 12px;
  border-radius: 26px;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(40px) saturate(1.8);
  -webkit-backdrop-filter: blur(40px) saturate(1.8);
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.06);
}

.brand-panel {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 14px;
}

.brand-mark {
  width: 40px;
  border-radius: 14px;
  display: block;
  object-fit: cover;
}

.brand-copy h3 {
  margin: 0;
  font-size: 15px;
  line-height: 1.2;
}

.brand-copy {
  min-width: 0;
}

.brand-copy h3,
.brand-copy p {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-copy p {
  margin: 3px 0 0;
  color: #86868b;
  font-size: 11px;
  line-height: 1.35;
}

.sidebar-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 14px;
}

.apple-chip {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 8px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.apple-chip.is-primary {
  background: rgba(0, 122, 255, 0.1);
  color: #007aff;
}

.apple-chip.is-success {
  background: rgba(52, 199, 89, 0.1);
  color: #34c759;
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
  margin: 0 -4px -4px;
  padding-right: 4px;
}

.sidebar-menu {
  background: transparent;
  border-right: 0;
  padding: 2px 0 12px;
}

.sidebar-menu :deep(.el-menu-item) {
  height: 40px;
  margin-bottom: 4px;
  border-radius: 14px;
  color: #86868b;
  font-weight: 600;
  padding: 0 12px !important;
  font-size: 13px;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
}

.sidebar-menu :deep(.el-menu-item span) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  color: #1d1d1f;
  background: rgba(255, 255, 255, 0.82);
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  color: #007aff;
  background: rgba(0, 122, 255, 0.1);
  box-shadow: inset 0 0 0 1px rgba(0, 122, 255, 0.08);
}

.sidebar-menu :deep(.el-menu-item .el-icon) {
  margin-right: 8px;
  font-size: 15px;
  flex: none;
}

/* 右侧内容区 */
.user-content {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.content-header {
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(40px) saturate(1.8);
  -webkit-backdrop-filter: blur(40px) saturate(1.8);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.04);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.header-left {
  min-width: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: none;
}

.profile-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, #eff6ff 0%, #dcebff 100%);
  color: #007aff;
  font-size: 13px;
  font-weight: 700;
  flex: none;
}

.profile-username {
  font-size: 14px;
  font-weight: 600;
  color: #1d1d1f;
  white-space: nowrap;
}

.eyebrow {
  display: block;
  color: #007aff;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  margin-bottom: 4px;
}

.page-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: #1d1d1f;
}

.main-area {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 0 4px 4px 0;
  scrollbar-gutter: stable;
  -webkit-overflow-scrolling: touch;
}

.placeholder-page {
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(40px) saturate(1.8);
  -webkit-backdrop-filter: blur(40px) saturate(1.8);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.04);
  padding: 60px 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

/* 响应式布局 */
@media (max-width: 1360px) {
  .user-layout {
    grid-template-columns: 208px minmax(0, 1fr);
  }
}
</style>

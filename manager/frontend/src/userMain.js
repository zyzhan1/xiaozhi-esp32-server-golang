import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import UserApp from './UserApp.vue'
import './styles/apple-light.css'

// 用户模块专用路由（无管理员鉴权守卫）
const userRoutes = [
  {
    path: '/user/agents/:id/edit',
    component: () => import('./views/user/AgentEdit.vue')
  },
  {
    path: '/user/agents/:id/history',
    component: () => import('./views/user/AgentHistory.vue')
  },
  {
    path: '/user/devices',
    component: () => import('./views/user/AgentDevices.vue')
  },
  {
    path: '/user/agents',
    component: UserApp
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/user/agents'
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes: userRoutes
})

// 用户模块路由守卫：检查普通用户登录态
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('userToken')
  if (!token && to.path !== '/user/agents') {
    next('/user/agents')
    return
  }
  next()
})

const app = createApp(UserApp)

// 注册所有 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus)
app.use(router)
app.mount('#app')

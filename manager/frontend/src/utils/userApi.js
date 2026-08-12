import axios from 'axios'
import { ElMessage } from 'element-plus'
import { getUserToken, clearUserAuth } from './userAuth.js'

/**
 * 普通用户专用 axios 实例
 * 请求拦截器从 user_token 读取 token，与管理员 api.js 的 token 完全隔离
 */
const userApi = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// 请求拦截器：使用普通用户的 token
userApi.interceptors.request.use(
  (config) => {
    const token = getUserToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器：401 时清除普通用户登录态
userApi.interceptors.response.use(
  (response) => response,
  (error) => {
    const silentError = error.config?.silentError === true
    if (error.response?.status === 401) {
      clearUserAuth()
      // 跳回当前页面（触发重新登录）
      window.location.reload()
    } else if (!silentError) {
      ElMessage.error(error.response?.data?.error || '请求失败')
    }
    return Promise.reject(error)
  }
)

export default userApi

/**
 * 普通用户认证模块
 * 使用独立的 localStorage key（userToken / myUser），与管理员的 token / user 完全隔离
 */

const TOKEN_KEY = 'userToken'
const USER_KEY = 'myUser'

/**
 * 获取普通用户 token
 */
export const getUserToken = () => localStorage.getItem(TOKEN_KEY)

/**
 * 获取普通用户信息
 */
export const getUserInfo = () => {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || 'null')
  } catch {
    return null
  }
}

/**
 * 保存登录信息（完整覆盖，防止串号）
 */
export const saveUserAuth = (token, userInfo) => {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify(userInfo))
}

/**
 * 清除普通用户登录信息
 */
export const clearUserAuth = () => {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

/**
 * 判断普通用户是否已登录
 */
export const isUserLoggedIn = () => !!getUserToken()

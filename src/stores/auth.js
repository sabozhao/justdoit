import { defineStore } from 'pinia'
import { authAPI } from '../api'
import { ElMessage } from 'element-plus'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    token: localStorage.getItem('token'),
    isAuthenticated: false,
    loading: false
  }),

  getters: {
    isLoggedIn: (state) => !!state.token && !!state.user,
    currentUser: (state) => state.user
  },

  actions: {
    // 初始化认证状态
    async initAuth() {
      if (this.token) {
        try {
          await this.getCurrentUser()
        } catch (error) {
          this.logout()
        }
      }
    },

    // 用户注册
    async register(userData) {
      try {
        this.loading = true
        const response = await authAPI.register(userData)
        
        this.token = response.token
        this.user = response.user
        this.isAuthenticated = true
        
        localStorage.setItem('token', response.token)
        ElMessage.success('注册成功！')
        
        return response
      } catch (error) {
        ElMessage.error(error.message || '注册失败')
        throw error
      } finally {
        this.loading = false
      }
    },

    // 用户登录
    async login(credentials) {
      try {
        this.loading = true
        const response = await authAPI.login(credentials)
        
        this.token = response.token
        this.user = response.user
        this.isAuthenticated = true
        
        localStorage.setItem('token', response.token)
        ElMessage.success('登录成功！')
        
        return response
      } catch (error) {
        ElMessage.error(error.message || '登录失败')
        throw error
      } finally {
        this.loading = false
      }
    },

    // 获取当前用户信息
    async getCurrentUser() {
      try {
        const user = await authAPI.getCurrentUser()
        this.user = user
        this.isAuthenticated = true
        return user
      } catch (error) {
        this.logout()
        throw error
      }
    },

    // 用户登出
    logout() {
      this.user = null
      this.token = null
      this.isAuthenticated = false
      
      localStorage.removeItem('token')
      ElMessage.success('已退出登录')
    },

    // 更新token
    setToken(token) {
      this.token = token
      localStorage.setItem('token', token)
    }
  }
})
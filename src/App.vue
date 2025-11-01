<template>
  <div id="app">
    <el-container class="app-container">
      <el-header class="app-header" v-if="!isLoginPage">
        <div class="header-content">
          <div class="logo">
            <el-icon><Document /></el-icon>
            <span class="desktop-only">智能刷题平台</span>
            <span class="mobile-only">刷题平台</span>
          </div>
          <div class="header-right">
            <!-- 桌面端菜单 -->
            <el-menu
              :default-active="$route.path"
              mode="horizontal"
              router
              class="header-menu desktop-menu"
            >
              <el-menu-item index="/">
                <el-icon><House /></el-icon>
                <span>首页</span>
              </el-menu-item>
              <el-menu-item index="/library">
                <el-icon><FolderOpened /></el-icon>
                <span>题库管理</span>
              </el-menu-item>
              <el-menu-item index="/practice">
                <el-icon><Edit /></el-icon>
                <span>开始刷题</span>
              </el-menu-item>
              <el-menu-item index="/wrong-questions">
                <el-icon><Warning /></el-icon>
                <span>错题库</span>
              </el-menu-item>
            </el-menu>
            
            <!-- 移动端菜单 -->
            <el-dropdown class="mobile-menu" @command="handleMobileMenuCommand">
              <el-button type="text" class="mobile-menu-btn">
                <el-icon><Menu /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="/">
                    <el-icon><House /></el-icon>
                    首页
                  </el-dropdown-item>
                  <el-dropdown-item command="/library">
                    <el-icon><FolderOpened /></el-icon>
                    题库管理
                  </el-dropdown-item>
                  <el-dropdown-item command="/practice">
                    <el-icon><Edit /></el-icon>
                    开始刷题
                  </el-dropdown-item>
                  <el-dropdown-item command="/wrong-questions">
                    <el-icon><Warning /></el-icon>
                    错题库
                  </el-dropdown-item>
                  <el-dropdown-item v-if="authStore.user && authStore.user.is_admin" command="/admin">
                    <el-icon><Setting /></el-icon>
                    管理员面板
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            
            <div class="user-menu" v-if="authStore.isLoggedIn">
              <el-dropdown @command="handleUserCommand">
                <span class="user-info">
                  <el-icon><User /></el-icon>
                  <span class="desktop-only">{{ authStore.currentUser?.username }}</span>
                  <span class="mobile-only">{{ authStore.currentUser?.username?.substring(0, 3) }}</span>
                  <el-icon><ArrowDown /></el-icon>
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="logout">
                      <el-icon><SwitchButton /></el-icon>
                      退出登录
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </div>
      </el-header>
      
      <el-main class="app-main" :class="{ 'login-main': isLoginPage }">
        <router-view />
      </el-main>
      
      <el-footer class="app-footer">
        <div class="footer-content">
          <a href="https://beian.miit.gov.cn" target="_blank" rel="noopener noreferrer" class="beian-link">
            粤ICP备2025487041号
          </a>
        </div>
      </el-footer>
    </el-container>
  </div>
</template>

<script>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

export default {
  name: 'App',
  setup() {
    const route = useRoute()
    const router = useRouter()
    const authStore = useAuthStore()

    const isLoginPage = computed(() => route.path === '/login')

    const handleUserCommand = (command) => {
      if (command === 'logout') {
        authStore.logout()
        router.push('/login')
      }
    }

    const handleMobileMenuCommand = (command) => {
      router.push(command)
    }

    return {
      authStore,
      isLoginPage,
      handleUserCommand,
      handleMobileMenuCommand
    }
  }
}
</script>

<style>
/* 基础样式重置 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', '微软雅黑', Arial, sans-serif;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  min-height: 100vh;
  overflow-x: hidden;
}

/* 应用容器 */
.app-container {
  min-height: 100vh;
}

/* 头部样式 */
.app-header {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 0;
  position: relative;
  z-index: 1000;
  height: 60px;
  line-height: 60px;
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 20px;
  height: 100%;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}

.logo {
  display: flex;
  align-items: center;
  font-size: 20px;
  font-weight: bold;
  color: #409eff;
}

.logo .el-icon {
  margin-right: 8px;
  font-size: 24px;
}

.header-menu {
  border: none;
  background: transparent;
  display: flex;
  align-items: center;
}

.app-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.app-main.login-main {
  max-width: none;
  padding: 0;
}

.user-menu {
  margin-left: 20px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #409eff;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 6px;
  transition: background-color 0.3s;
}

.user-info:hover {
  background-color: rgba(64, 158, 255, 0.1);
}

/* 移动端菜单按钮样式 */
.mobile-menu-btn {
  color: #409eff !important;
  font-size: 20px !important;
  padding: 8px !important;
}

.mobile-menu-btn:hover {
  background-color: rgba(64, 158, 255, 0.1) !important;
}

/* 移动端下拉菜单样式 */
.mobile-only .el-dropdown-menu {
  min-width: 150px;
}

.mobile-only .el-dropdown-menu .el-dropdown-item {
  padding: 12px 20px;
  font-size: 14px;
}

.mobile-only .el-dropdown-menu .el-dropdown-item .el-icon {
  margin-right: 8px;
}

/* 响应式断点 */
.mobile-only {
  display: none;
}

.desktop-only {
  display: block;
}

.desktop-menu {
  display: flex;
}

.mobile-menu {
  display: none;
}

/* 移动端适配 */
@media (max-width: 768px) {
  body {
    font-size: 14px;
  }
  
  .header-content {
    padding: 0 15px;
    flex-wrap: wrap;
    min-height: 60px;
  }
  
  .logo {
    font-size: 16px;
  }
  
  .logo .el-icon {
    font-size: 20px;
  }
  
  .header-right {
    gap: 10px;
  }
  
  .desktop-menu {
    display: none !important;
  }
  
  .mobile-menu {
    display: block !important;
  }
  
  .user-menu {
    margin-left: 0;
  }
  
  .user-info {
    padding: 6px 10px;
    font-size: 14px;
  }
  
  .app-main {
    padding: 15px;
  }
  
  .mobile-only {
    display: block;
  }
  
  .desktop-only {
    display: none;
  }
}

@media (max-width: 480px) {
  .header-content {
    padding: 0 10px;
  }
  
  .logo {
    font-size: 14px;
  }
  
  .app-main {
    padding: 10px;
  }
}

/* Element Plus 组件样式覆盖 */
:deep(.el-header) {
  padding: 0 !important;
  height: 60px !important;
  line-height: 60px !important;
}

:deep(.el-menu--horizontal) {
  border-bottom: none !important;
}

:deep(.el-menu--horizontal .el-menu-item) {
  height: 60px !important;
  line-height: 60px !important;
  border-bottom: none !important;
}

:deep(.el-menu--horizontal .el-menu-item:hover) {
  background-color: rgba(64, 158, 255, 0.1) !important;
}

:deep(.el-menu--horizontal .el-menu-item.is-active) {
  background-color: rgba(64, 158, 255, 0.1) !important;
  border-bottom: none !important;
}

/* Footer样式 */
.app-footer {
  padding: 0;
  height: auto;
  background: transparent;
  text-align: center;
}

.footer-content {
  padding: 15px 20px;
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  max-width: 1200px;
  margin: 0 auto;
}

.beian-link {
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  transition: color 0.3s;
  font-size: 12px;
}

.beian-link:hover {
  color: rgba(255, 255, 255, 0.9);
  text-decoration: underline;
}

/* 移动端Footer适配 */
@media (max-width: 768px) {
  .footer-content {
    padding: 12px 15px;
    font-size: 11px;
  }
  
  .beian-link {
    font-size: 11px;
  }
}

@media (max-width: 480px) {
  .footer-content {
    padding: 10px 12px;
    font-size: 10px;
  }
  
  .beian-link {
    font-size: 10px;
  }
}

/* Element Plus Footer样式覆盖 */
:deep(.el-footer) {
  padding: 0 !important;
  height: auto !important;
}

/* 触摸设备优化 */
@media (hover: none) and (pointer: coarse) {
  .user-info:hover {
    background-color: transparent;
  }
  
  :deep(.el-menu--horizontal .el-menu-item:hover) {
    background-color: transparent !important;
  }
  
  .beian-link:hover {
    color: rgba(255, 255, 255, 0.7);
  }
}
</style>
<template>
  <div class="home">
    <div class="hero-section">
      <div class="hero-content">
        <h1 class="hero-title">智能刷题平台</h1>
        <p class="hero-subtitle">提升学习效率，掌握知识要点</p>
        <p class="hero-free">永久免费 · 无限制使用</p>
        <div class="hero-buttons">
          <el-button type="primary" size="large" @click="$router.push('/library')">
            <el-icon><FolderOpened /></el-icon>
            <span class="desktop-only">管理题库</span>
            <span class="mobile-only">题库</span>
          </el-button>
          <el-button type="success" size="large" @click="$router.push('/practice')">
            <el-icon><Edit /></el-icon>
            <span class="desktop-only">开始刷题</span>
            <span class="mobile-only">刷题</span>
          </el-button>
          <el-button v-if="authStore.user && authStore.user.is_admin" type="warning" size="large" @click="$router.push('/admin')" class="admin-btn">
            <el-icon><Setting /></el-icon>
            <span class="desktop-only">管理员面板</span>
            <span class="mobile-only">管理</span>
          </el-button>
        </div>
      </div>
    </div>

    <div class="stats-section">
      <el-row :gutter="20">
        <el-col :span="8">
          <el-card class="stats-card">
            <div class="stats-item">
              <el-icon class="stats-icon"><FolderOpened /></el-icon>
              <div class="stats-content">
                <div class="stats-number">{{ questionBanks?.length || 0 }}</div>
                <div class="stats-label">题库数量</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card class="stats-card">
            <div class="stats-item">
              <el-icon class="stats-icon"><Document /></el-icon>
              <div class="stats-content">
                <div class="stats-number">{{ totalQuestions || 0 }}</div>
                <div class="stats-label">总题目数</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card class="stats-card">
            <div class="stats-item">
              <el-icon class="stats-icon"><Warning /></el-icon>
              <div class="stats-content">
                <div class="stats-number">{{ wrongQuestions?.length || 0 }}</div>
                <div class="stats-label">错题数量</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <div class="features-section">
      <h2 class="section-title">平台特色</h2>
      <el-row :gutter="30">
        <el-col :span="8">
          <div class="feature-card">
            <el-icon class="feature-icon"><Upload /></el-icon>
            <h3>自定义题库</h3>
            <p>支持上传JSON、Excel、CSV格式的题库文件，灵活管理各类题目</p>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="feature-card">
            <el-icon class="feature-icon"><TrendCharts /></el-icon>
            <h3>智能评分</h3>
            <p>自动统计答题情况，实时显示得分和正确率</p>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="feature-card">
            <el-icon class="feature-icon"><Collection /></el-icon>
            <h3>错题管理</h3>
            <p>自动收集错题，支持针对性复习和巩固</p>
          </div>
        </el-col>
      </el-row>
    </div>

    <div class="contact-section">
      <h2 class="section-title">联系我们</h2>
      <div class="contact-info">
        <p>如有问题或建议，欢迎联系我们：</p>
        <p class="contact-email">
          <el-icon><Message /></el-icon>
          邮箱：867368106@QQ.com
        </p>
      </div>
    </div>
  </div>
</template>

<script>
import { computed, onMounted } from 'vue'
import { useExamStore } from '../stores/exam'
import { useAuthStore } from '../stores/auth'
import { Message } from '@element-plus/icons-vue'

export default {
  name: 'Home',
  setup() {
    const examStore = useExamStore()
    const authStore = useAuthStore()

    const questionBanks = computed(() => examStore.questionBanks || [])
    const wrongQuestions = computed(() => examStore.wrongQuestions || [])
    const totalQuestions = computed(() => {
      const banks = examStore.questionBanks || []
      return banks.reduce((total, bank) => {
        return total + (bank.question_count || 0)
      }, 0)
    })



    // 页面加载时获取数据
    onMounted(async () => {
      console.log('Home mounted, authStore state:', {
        token: authStore.token,
        user: authStore.user,
        isAuthenticated: authStore.isAuthenticated
      })
      
      // 确保用户信息已加载
      if (authStore.token && !authStore.user) {
        try {
          console.log('Initializing auth...')
          await authStore.initAuth()
          console.log('Auth initialized successfully')
        } catch (error) {
          console.error('Failed to initialize auth:', error)
        }
      }
      
      await examStore.loadQuestionBanks()
      await examStore.loadWrongQuestions()
    })

    return {
      questionBanks,
      wrongQuestions,
      totalQuestions,
      authStore
    }
  }
}
</script>

<style scoped>
.home {
  min-height: calc(100vh - 120px);
}

.hero-section {
  text-align: center;
  padding: 60px 0;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  margin-bottom: 40px;
  backdrop-filter: blur(10px);
}

.hero-title {
  font-size: 48px;
  font-weight: bold;
  color: white;
  margin-bottom: 16px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.hero-subtitle {
  font-size: 20px;
  color: rgba(255, 255, 255, 0.9);
  margin-bottom: 20px;
}

.hero-free {
  font-size: 18px;
  color: #52c41a;
  font-weight: bold;
  margin-bottom: 40px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.hero-buttons {
  display: flex;
  gap: 20px;
  justify-content: center;
}

.stats-section {
  margin-bottom: 60px;
}

.stats-card {
  border-radius: 15px;
  border: none;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
  transition: transform 0.3s ease;
}

.stats-card:hover {
  transform: translateY(-5px);
}

.stats-item {
  display: flex;
  align-items: center;
  padding: 20px;
}

.stats-icon {
  font-size: 40px;
  color: #409eff;
  margin-right: 20px;
}

.stats-content {
  flex: 1;
}

.stats-number {
  font-size: 32px;
  font-weight: bold;
  color: #303133;
  line-height: 1;
}

.stats-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.features-section {
  text-align: center;
}

.section-title {
  font-size: 32px;
  color: white;
  margin-bottom: 40px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.feature-card {
  background: rgba(255, 255, 255, 0.95);
  padding: 40px 30px;
  border-radius: 20px;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
  transition: transform 0.3s ease;
  height: 100%;
}

.feature-card:hover {
  transform: translateY(-10px);
}

.feature-icon {
  font-size: 50px;
  color: #409eff;
  margin-bottom: 20px;
}

.feature-card h3 {
  font-size: 20px;
  color: #303133;
  margin-bottom: 15px;
}

.feature-card p {
  color: #606266;
  line-height: 1.6;
}

.contact-section {
  text-align: center;
  margin-top: 60px;
  padding: 40px 0;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 20px;
}

.contact-info {
  max-width: 600px;
  margin: 0 auto;
}

.contact-info p {
  color: rgba(255, 255, 255, 0.9);
  font-size: 16px;
  line-height: 1.8;
  margin-bottom: 10px;
}

.contact-email {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  font-size: 18px;
  font-weight: 500;
  color: #409eff;
  margin-top: 20px;
}

/* 管理员按钮样式 */
.admin-btn {
  margin-left: 10px;
}

@media (max-width: 768px) {
  .admin-btn {
    margin-left: 0;
  }
}
</style>
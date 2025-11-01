<template>
  <div class="practice">
    <div class="practice-header">
      <h1>选择题库</h1>
      <p>选择一个题库开始刷题练习</p>
    </div>

    <div class="practice-content">
      <div v-if="displayedBanks.length > 0" class="banks-grid">
        <div v-for="bank in displayedBanks" :key="bank.id" class="bank-item">
          <el-card class="practice-card" @click="startExam(bank.id)">
            <div class="card-content">
              <div class="card-icon">
                <el-icon><Document /></el-icon>
              </div>
              <div class="card-info">
                <h3>{{ bank.name }}</h3>
                <p>{{ bank.description || '暂无描述' }}</p>
                <div class="card-stats">
                  <span class="stat">{{ bank.question_count || 0 }} 道题</span>
                  <span class="stat">{{ formatDate(bank.created_at) }}</span>
                </div>
              </div>
              <div class="card-action">
                <el-button type="primary" size="large">
                  开始练习
                  <el-icon><ArrowRight /></el-icon>
                </el-button>
              </div>
            </div>
          </el-card>
        </div>
      </div>

      <div v-else class="empty-state">
        <el-empty description="暂无题库">
          <el-button type="primary" @click="$router.push('/library')">
            去上传题库
          </el-button>
        </el-empty>
      </div>
    </div>

    <!-- 错题库练习 -->
    <div class="wrong-questions-section" v-if="wrongQuestions.length > 0">
      <h2>错题练习</h2>
      <el-card class="wrong-questions-card" @click="startWrongQuestionsExam">
        <div class="card-content">
          <div class="card-icon error">
            <el-icon><Warning /></el-icon>
          </div>
          <div class="card-info">
            <h3>错题集合</h3>
            <p>复习之前做错的题目，巩固知识点</p>
            <div class="card-stats">
              <span class="stat">{{ wrongQuestions.length }} 道错题</span>
            </div>
          </div>
          <div class="card-action">
            <el-button type="danger" size="large">
              错题练习
              <el-icon><ArrowRight /></el-icon>
            </el-button>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script>
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useExamStore } from '../stores/exam'

export default {
  name: 'Practice',
  setup() {
    const router = useRouter()
    const examStore = useExamStore()

    const questionBanks = computed(() => examStore.questionBanks || [])
    const wrongQuestions = computed(() => examStore.wrongQuestions || [])

    const startExam = (bankId) => {
      router.push(`/exam/${bankId}`)
    }

    const startWrongQuestionsExam = () => {
      router.push('/exam/wrong-questions')
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleDateString('zh-CN')
    }

    // 页面加载时获取数据
    onMounted(async () => {
      await examStore.loadQuestionBanks()
      await examStore.loadWrongQuestions()
    })

    // 显示所有题库，不限制总数
    const displayedBanks = computed(() => questionBanks.value)

    return {
      questionBanks,
      displayedBanks,
      wrongQuestions,
      startExam,
      startWrongQuestionsExam,
      formatDate
    }
  }
}
</script>

<style scoped>
.practice {
  min-height: calc(100vh - 120px);
}

.practice-header {
  text-align: center;
  margin-bottom: 40px;
  padding: 40px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  backdrop-filter: blur(10px);
}

.practice-header h1 {
  color: white;
  font-size: 32px;
  margin-bottom: 10px;
}

.practice-header p {
  color: rgba(255, 255, 255, 0.8);
  font-size: 16px;
}

.practice-content {
  margin-bottom: 60px;
}

.banks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
  width: 100%;
  max-width: none;
}

.bank-item {
  width: 100%;
}

@media (max-width: 768px) {
  .banks-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}

.bank-item {
  width: 100%;
}

.practice-card {
  border-radius: 15px;
  border: none;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
  cursor: pointer;
  margin-bottom: 20px;
}

.practice-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.15);
}

.card-content {
  display: flex;
  align-items: center;
  padding: 30px;
}

.card-icon {
  width: 60px;
  height: 60px;
  border-radius: 15px;
  background: linear-gradient(135deg, #409eff, #66b1ff);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 20px;
  flex-shrink: 0;
}

.card-icon.error {
  background: linear-gradient(135deg, #f56c6c, #f78989);
}

.card-icon .el-icon {
  font-size: 28px;
  color: white;
}

.card-info {
  flex: 1;
}

.card-info h3 {
  font-size: 20px;
  color: #303133;
  margin-bottom: 8px;
}

.card-info p {
  color: #606266;
  margin-bottom: 12px;
  line-height: 1.5;
}

.card-stats {
  display: flex;
  gap: 20px;
}

.stat {
  color: #909399;
  font-size: 14px;
}

.card-action {
  flex-shrink: 0;
}

.empty-state {
  text-align: center;
  padding: 60px 0;
}

.wrong-questions-section {
  margin-top: 60px;
}

.wrong-questions-section h2 {
  color: white;
  font-size: 24px;
  margin-bottom: 20px;
  text-align: center;
}

.wrong-questions-card {
  border-radius: 15px;
  border: none;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
  cursor: pointer;
}

.wrong-questions-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.15);
}
</style>
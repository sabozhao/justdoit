<template>
  <div class="exam">
    <div class="exam-header" v-if="!showResult">
      <div class="exam-info">
        <h2>{{ examTitle }}</h2>
        <div class="exam-progress">
          <span>第 {{ currentQuestionIndex + 1 }} 题 / 共 {{ questions.length }} 题</span>
          <el-progress :percentage="progressPercentage" :show-text="false" />
        </div>
      </div>
      <div class="exam-timer">
        <el-icon><Clock /></el-icon>
        <span>{{ formatTime(elapsedTime) }}</span>
      </div>
    </div>

    <!-- 题目导航 -->
    <div class="question-navigator" v-if="!showResult">
      <div class="navigator-header">
        <h3>题目导航</h3>
        <span class="legend">
          <span class="legend-item">
            <span class="legend-dot completed"></span>
            已完成
          </span>
          <span class="legend-item">
            <span class="legend-dot current"></span>
            当前题目
          </span>
          <span class="legend-item">
            <span class="legend-dot"></span>
            未完成
          </span>
        </span>
      </div>
      <div class="navigator-grid">
        <div
          v-for="(question, index) in questions"
          :key="index"
          class="question-nav-item"
          :class="{
            'completed': userAnswers[index] !== null,
            'current': index === currentQuestionIndex
          }"
          @click="jumpToQuestion(index)"
        >
          {{ index + 1 }}
        </div>
      </div>
    </div>

    <!-- 答题界面 -->
    <div class="exam-content" v-if="!showResult && currentQuestion">
      <el-card class="question-card">
        <div class="question-header">
          <span class="question-number">第 {{ currentQuestionIndex + 1 }} 题</span>
        </div>
        
        <div class="question-content">
          <h3>{{ currentQuestion.question }}</h3>
        </div>

        <div class="options-container">
          <div
            v-for="(option, index) in currentQuestion.options"
            :key="index"
            class="option-item"
            :class="{ 'selected': selectedAnswer === index }"
            @click="selectAnswer(index)"
          >
            <div class="option-label">{{ String.fromCharCode(65 + index) }}</div>
            <div class="option-text">{{ option }}</div>
          </div>
        </div>

        <div class="question-actions">
          <el-button
            v-if="currentQuestionIndex > 0"
            @click="previousQuestion"
          >
            上一题
          </el-button>
          <el-button
            type="primary"
            @click="nextQuestion"
            :disabled="selectedAnswer === null"
          >
            {{ currentQuestionIndex === questions.length - 1 ? '提交答案' : '下一题' }}
          </el-button>
        </div>
      </el-card>
    </div>

    <!-- 结果页面 -->
    <div class="exam-result" v-if="showResult">
      <el-card class="result-card">
        <div class="result-header">
          <el-icon class="result-icon" :class="resultIconClass">
            <component :is="resultIcon" />
          </el-icon>
          <h2>考试完成</h2>
        </div>

        <div class="result-stats">
          <div class="stat-item">
            <div class="stat-value">{{ score }}%</div>
            <div class="stat-label">得分</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ correctCount }}</div>
            <div class="stat-label">正确</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ wrongCount }}</div>
            <div class="stat-label">错误</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ formatTime(totalTime) }}</div>
            <div class="stat-label">用时</div>
          </div>
        </div>

        <div class="result-actions">
          <el-button @click="reviewAnswers">查看答案</el-button>
          <el-button type="primary" @click="restartExam">重新开始</el-button>
          <el-button @click="$router.push('/practice')">返回练习</el-button>
        </div>
      </el-card>

      <!-- 答案回顾 -->
      <div class="answer-review" v-if="showReview">
        <h3>答案回顾</h3>
        <div v-for="(question, index) in questions" :key="index" class="review-item">
          <div class="review-header">
            <span class="review-number">第 {{ index + 1 }} 题</span>
            <el-tag :type="userAnswers[index] === question.answer ? 'success' : 'danger'">
              {{ userAnswers[index] === question.answer ? '正确' : '错误' }}
            </el-tag>
          </div>
          <div class="review-question">{{ question.question }}</div>
          <div class="review-options">
            <div
              v-for="(option, optionIndex) in question.options"
              :key="optionIndex"
              class="review-option"
              :class="{
                'correct': optionIndex === question.answer,
                'wrong': optionIndex === userAnswers[index] && optionIndex !== question.answer,
                'user-selected': optionIndex === userAnswers[index]
              }"
            >
              <span class="option-label">{{ String.fromCharCode(65 + optionIndex) }}</span>
              <span>{{ option }}</span>
            </div>
          </div>
          <div v-if="question.explanation" class="review-explanation">
            <strong>解析：</strong>{{ question.explanation }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useExamStore } from '../stores/exam'
import { ElMessage } from 'element-plus'

export default {
  name: 'Exam',
  setup() {
    const route = useRoute()
    const router = useRouter()
    const examStore = useExamStore()

    const questions = ref([])
    const currentQuestionIndex = ref(0)
    const selectedAnswer = ref(null)
    const userAnswers = ref([])
    const showResult = ref(false)
    const showReview = ref(false)
    const elapsedTime = ref(0)
    const totalTime = ref(0)
    const timer = ref(null)

    const examTitle = computed(() => {
      if (route.name === 'WrongQuestionsExam' || route.params.id.startsWith('wrong-questions')) {
        return '错题练习'
      }
      const bank = examStore.getQuestionBankById(route.params.id)
      return bank ? bank.name : '未知题库'
    })

    const currentQuestion = computed(() => {
      return questions.value[currentQuestionIndex.value]
    })

    const progressPercentage = computed(() => {
      return Math.round(((currentQuestionIndex.value + 1) / questions.value.length) * 100)
    })

    const correctCount = computed(() => {
      return userAnswers.value.filter((answer, index) => 
        answer === questions.value[index].answer
      ).length
    })

    const wrongCount = computed(() => {
      return questions.value.length - correctCount.value
    })

    const score = computed(() => {
      return Math.round((correctCount.value / questions.value.length) * 100)
    })

    const resultIcon = computed(() => {
      return score.value >= 80 ? 'SuccessFilled' : score.value >= 60 ? 'WarningFilled' : 'CircleCloseFilled'
    })

    const resultIconClass = computed(() => {
      return score.value >= 80 ? 'success' : score.value >= 60 ? 'warning' : 'error'
    })

    const initExam = async () => {
      try {
        // 处理错题库重考的路由参数
        if (route.name === 'WrongQuestionsExam') {
          await examStore.loadWrongQuestions()
          // 使用bankId参数过滤错题
          const bankId = route.params.bankId
          questions.value = examStore.wrongQuestions.filter(q => q.bank_id === bankId)
        } else if (route.params.id && route.params.id.startsWith('wrong-questions')) {
          await examStore.loadWrongQuestions()
          
          // 如果是特定题库的错题练习，提取bank_id
          if (route.params.id.includes('/')) {
            const parts = route.params.id.split('/')
            const bankId = parts[1]
            // 只加载该题库的错题
            questions.value = examStore.wrongQuestions.filter(q => q.bank_id === bankId)
          } else {
            // 加载所有错题
            questions.value = [...examStore.wrongQuestions]
          }
        } else {
          const bank = await examStore.getQuestionBankDetails(route.params.id)
          if (bank && bank.questions) {
            questions.value = [...bank.questions]
          } else {
            ElMessage.error('题库不存在或为空')
            router.push('/practice')
            return
          }
        }

        if (questions.value.length === 0) {
          ElMessage.error('题库为空')
          router.push('/practice')
          return
        }

        // 打乱题目顺序
        questions.value = shuffleArray(questions.value)
        userAnswers.value = new Array(questions.value.length).fill(null)
        
        startTimer()
      } catch (error) {
        ElMessage.error('加载题库失败')
        router.push('/practice')
      }
    }

    const shuffleArray = (array) => {
      const newArray = [...array]
      for (let i = newArray.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1))
        ;[newArray[i], newArray[j]] = [newArray[j], newArray[i]]
      }
      return newArray
    }

    const startTimer = () => {
      timer.value = setInterval(() => {
        elapsedTime.value++
      }, 1000)
    }

    const stopTimer = () => {
      if (timer.value) {
        clearInterval(timer.value)
        timer.value = null
      }
    }

    const selectAnswer = (index) => {
      selectedAnswer.value = index
      userAnswers.value[currentQuestionIndex.value] = index
    }

    const nextQuestion = () => {
      if (selectedAnswer.value === null) {
        ElMessage.warning('请选择一个答案')
        return
      }

      if (currentQuestionIndex.value === questions.value.length - 1) {
        finishExam()
      } else {
        currentQuestionIndex.value++
        selectedAnswer.value = userAnswers.value[currentQuestionIndex.value]
      }
    }

    const previousQuestion = () => {
      if (currentQuestionIndex.value > 0) {
        currentQuestionIndex.value--
        selectedAnswer.value = userAnswers.value[currentQuestionIndex.value]
      }
    }

    const jumpToQuestion = (index) => {
      currentQuestionIndex.value = index
      selectedAnswer.value = userAnswers.value[index]
    }

    const finishExam = async () => {
      stopTimer()
      totalTime.value = elapsedTime.value
      showResult.value = true

      // 获取正确的bank_id - 错题练习时不保存考试结果
      let actualBankId = route.params.id
      let shouldSaveResult = true

      // 处理WrongQuestionsExam路由
      if (route.name === 'WrongQuestionsExam') {
        actualBankId = route.params.bankId
      } else if (route.params.id && route.params.id.startsWith('wrong-questions')) {
        // 如果是错题库重考，从路由参数中提取真实的bank_id
        if (route.params.id.includes('/')) {
          const parts = route.params.id.split('/')
          actualBankId = parts[1]
        } else {
          // 如果是所有错题练习，不保存考试结果
          shouldSaveResult = false
        }
      }

      // 保存错题
      if (route.params.id !== 'wrong-questions' && !route.params.id.startsWith('wrong-questions') && route.name !== 'WrongQuestionsExam') {
        for (let index = 0; index < questions.value.length; index++) {
          const question = questions.value[index]
          if (userAnswers.value[index] !== question.answer) {
            await examStore.addWrongQuestion(question, route.params.id)
          }
        }
      } else if ((route.params.id && route.params.id.startsWith('wrong-questions') && !route.params.id.includes('/')) || route.name === 'WrongQuestionsExam') {
        // 从错题库中移除答对的题目（错题练习时）
        for (let index = 0; index < questions.value.length; index++) {
          const question = questions.value[index]
          if (userAnswers.value[index] === question.answer) {
            await examStore.removeWrongQuestion(question.id)
          }
        }
      }

      // 保存考试结果（错题练习时不保存）
      if (shouldSaveResult) {
        await examStore.saveExamResult({
          bankId: actualBankId,
          score: score.value,
          correctCount: correctCount.value,
          wrongCount: wrongCount.value,
          totalTime: totalTime.value,
          totalQuestions: questions.value.length
        })
      }
    }

    const reviewAnswers = () => {
      showReview.value = !showReview.value
    }

    const restartExam = () => {
      currentQuestionIndex.value = 0
      selectedAnswer.value = null
      userAnswers.value = new Array(questions.value.length).fill(null)
      showResult.value = false
      showReview.value = false
      elapsedTime.value = 0
      
      // 重新打乱题目
      questions.value = shuffleArray(questions.value)
      startTimer()
    }

    const formatTime = (seconds) => {
      const minutes = Math.floor(seconds / 60)
      const remainingSeconds = seconds % 60
      return `${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`
    }

    onMounted(() => {
      initExam()
    })

    onUnmounted(() => {
      stopTimer()
    })

    return {
      questions,
      currentQuestionIndex,
      selectedAnswer,
      userAnswers,
      showResult,
      showReview,
      elapsedTime,
      totalTime,
      examTitle,
      currentQuestion,
      progressPercentage,
      correctCount,
      wrongCount,
      score,
      resultIcon,
      resultIconClass,
      selectAnswer,
      nextQuestion,
      previousQuestion,
      jumpToQuestion,
      reviewAnswers,
      restartExam,
      formatTime
    }
  }
}
</script>

<style scoped>
.exam {
  min-height: calc(100vh - 120px);
}

.exam-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding: 20px 30px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 15px;
  backdrop-filter: blur(10px);
}

.exam-info h2 {
  color: white;
  margin-bottom: 10px;
}

.exam-progress {
  color: rgba(255, 255, 255, 0.9);
}

.exam-progress .el-progress {
  margin-top: 8px;
}

.exam-timer {
  display: flex;
  align-items: center;
  gap: 8px;
  color: white;
  font-size: 18px;
  font-weight: bold;
}

.question-navigator {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 15px;
  padding: 20px;
  margin-bottom: 30px;
  backdrop-filter: blur(10px);
}

.navigator-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.navigator-header h3 {
  color: white;
  margin: 0;
  font-size: 18px;
}

.legend {
  display: flex;
  gap: 15px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 5px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 12px;
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
}

.legend-dot.completed {
  background: #67c23a;
}

.legend-dot.current {
  background: #409eff;
}

.navigator-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(40px, 1fr));
  gap: 8px;
  max-height: 120px;
  overflow-y: auto;
}

.question-nav-item {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.3s ease;
  font-weight: 500;
  border: 2px solid transparent;
}

.question-nav-item:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: translateY(-2px);
}

.question-nav-item.completed {
  background: #67c23a;
  color: white;
}

.question-nav-item.current {
  background: #409eff;
  color: white;
  border-color: white;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.3);
}

.question-nav-item.completed.current {
  background: #409eff;
  border-color: #67c23a;
}

.question-card {
  border-radius: 20px;
  border: none;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
}

.question-header {
  margin-bottom: 20px;
}

.question-number {
  background: #409eff;
  color: white;
  padding: 5px 15px;
  border-radius: 20px;
  font-size: 14px;
}

.question-content {
  margin-bottom: 30px;
}

.question-content h3 {
  font-size: 20px;
  line-height: 1.6;
  color: #303133;
}

.options-container {
  margin-bottom: 40px;
}

.option-item {
  display: flex;
  align-items: center;
  padding: 15px 20px;
  margin-bottom: 12px;
  border: 2px solid #e4e7ed;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.option-item:hover {
  border-color: #409eff;
  background: #f0f9ff;
}

.option-item.selected {
  border-color: #409eff;
  background: #e6f7ff;
}

.option-label {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  margin-right: 15px;
  flex-shrink: 0;
}

.option-item.selected .option-label {
  background: #409eff;
  color: white;
}

.option-text {
  flex: 1;
  font-size: 16px;
  line-height: 1.5;
}

.question-actions {
  text-align: center;
}

.result-card {
  border-radius: 20px;
  border: none;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
  text-align: center;
  margin-bottom: 30px;
}

.result-header {
  margin-bottom: 30px;
}

.result-icon {
  font-size: 60px;
  margin-bottom: 15px;
}

.result-icon.success {
  color: #67c23a;
}

.result-icon.warning {
  color: #e6a23c;
}

.result-icon.error {
  color: #f56c6c;
}

.result-header h2 {
  font-size: 28px;
  color: #303133;
}

.result-stats {
  display: flex;
  justify-content: space-around;
  margin-bottom: 40px;
  padding: 30px 0;
  border-top: 1px solid #ebeef5;
  border-bottom: 1px solid #ebeef5;
}

.stat-item {
  text-align: center;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #409eff;
  margin-bottom: 5px;
}

.stat-label {
  color: #909399;
  font-size: 14px;
}

.result-actions {
  display: flex;
  gap: 15px;
  justify-content: center;
}

.answer-review {
  margin-top: 40px;
}

.answer-review h3 {
  color: white;
  margin-bottom: 20px;
  text-align: center;
}

.review-item {
  background: white;
  border-radius: 15px;
  padding: 25px;
  margin-bottom: 20px;
  box-shadow: 0 5px 15px rgba(0, 0, 0, 0.1);
}

.review-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.review-number {
  font-weight: bold;
  color: #409eff;
}

.review-question {
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 15px;
  color: #303133;
}

.review-options {
  margin-bottom: 15px;
}

.review-option {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  margin-bottom: 8px;
  border-radius: 8px;
  font-size: 14px;
}

.review-option.correct {
  background: #f0f9ff;
  color: #67c23a;
  border: 1px solid #67c23a;
}

.review-option.wrong {
  background: #fef0f0;
  color: #f56c6c;
  border: 1px solid #f56c6c;
}

.review-option .option-label {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  margin-right: 10px;
}

.review-explanation {
  background: #f8f9fa;
  padding: 12px;
  border-radius: 8px;
  font-size: 14px;
  color: #606266;
  line-height: 1.5;
}
</style>
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
          <el-tag v-if="currentQuestion.is_multiple" type="warning" size="small" style="margin-left: 10px;">
            多选题
          </el-tag>
          <el-tag v-else type="primary" size="small" style="margin-left: 10px;">
            单选题
          </el-tag>
        </div>
        
        <div class="question-content">
          <h3>{{ currentQuestion.question }}</h3>
        </div>

        <div class="options-container">
          <!-- 多选题：使用checkbox -->
          <template v-if="currentQuestion.is_multiple">
            <div
              v-for="(option, index) in currentQuestion.options"
              :key="'multiple-' + index"
              class="option-item"
              :class="{ 'selected': isOptionSelected(index) }"
              @click="toggleAnswer(index)"
            >
              <el-checkbox
                :model-value="isOptionSelected(index)"
                @change="toggleAnswer(index)"
                class="option-checkbox"
              />
              <div class="option-label">{{ String.fromCharCode(65 + index) }}</div>
              <div class="option-text">{{ option }}</div>
            </div>
          </template>
          
          <!-- 单选题：使用radio -->
          <template v-else>
            <div
              v-for="(option, index) in currentQuestion.options"
              :key="'single-' + index"
              class="option-item"
              :class="{ 'selected': selectedAnswer === index }"
              @click="selectAnswer(index)"
            >
              <div class="option-label">{{ String.fromCharCode(65 + index) }}</div>
              <div class="option-text">{{ option }}</div>
            </div>
          </template>
        </div>

        <div class="question-actions">
          <el-button
            v-if="currentQuestionIndex > 0"
            @click="previousQuestion"
          >
            上一题
          </el-button>
          <el-button
            v-if="currentQuestionIndex < questions.length - 1"
            type="primary"
            @click="nextQuestion"
            :disabled="!isAnswerSelected"
          >
            下一题
          </el-button>
          <!-- 如果全部答完，显示"交卷"按钮；否则显示"提前交卷"按钮 -->
          <el-button
            v-if="isAllAnswered"
            type="success"
            @click="submitExam"
            :style="{ marginLeft: currentQuestionIndex < questions.length - 1 ? '10px' : '0' }"
          >
            交卷
          </el-button>
          <el-button
            v-else
            type="warning"
            @click="submitEarly"
            :style="{ marginLeft: currentQuestionIndex < questions.length - 1 ? '10px' : '0' }"
          >
            提前交卷
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
          <!-- 调试信息（开发时可以取消注释查看） -->
          <!-- <div style="font-size: 12px; color: #999; margin-bottom: 10px; background: #f5f5f5; padding: 5px; border-radius: 4px;">
            题目索引: {{ index }}, 
            用户答案: {{ JSON.stringify(userAnswers[index]) }}, 
            正确答案: {{ JSON.stringify(question.answer) }},
            是否正确: {{ isAnswerCorrect(index) }}
          </div> -->
          <div class="review-header">
            <span class="review-number">第 {{ index + 1 }} 题</span>
            <el-tag v-if="question.is_multiple" type="warning" size="small" style="margin-right: 10px;">
              多选题
            </el-tag>
            <el-tag :type="isAnswerCorrect(index) ? 'success' : 'danger'">
              {{ isAnswerCorrect(index) ? '正确' : '错误' }}
            </el-tag>
          </div>
          <div class="review-question">{{ question.question }}</div>
          <div class="review-options">
            <div
              v-for="(option, optionIndex) in question.options"
              :key="optionIndex"
              class="review-option"
              :class="{
                'correct': isCorrectAnswer(question, optionIndex),
                'wrong': isWrongAnswer(question, optionIndex, index),
                'user-selected': isUserSelectedAnswer(question, optionIndex, index) && isCorrectAnswer(question, optionIndex)
              }"
            >
              <span class="option-label">{{ String.fromCharCode(65 + optionIndex) }}</span>
              <span>{{ option }}</span>
              <span class="answer-tags">
                <el-tag 
                  v-if="isWrongAnswer(question, optionIndex, index)" 
                  type="danger" 
                  size="small" 
                  style="margin-left: 10px;"
                >
                  ✗ 你的错误答案
                </el-tag>
                <el-tag 
                  v-if="isCorrectAnswer(question, optionIndex)" 
                  type="success" 
                  size="small" 
                  style="margin-left: 10px;"
                >
                  ✓ {{ isUserSelectedAnswer(question, optionIndex, index) ? '你的答案（正确）' : '正确答案' }}
                </el-tag>
              </span>
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
import { ElMessage, ElMessageBox } from 'element-plus'

export default {
  name: 'Exam',
  setup() {
    const route = useRoute()
    const router = useRouter()
    const examStore = useExamStore()

    const questions = ref([])
    const originalQuestions = ref([])  // 保存原始题目顺序（不打乱）
    const questionIndexMap = ref(new Map())  // 保存打乱后的索引到原始索引的映射
    const currentQuestionIndex = ref(0)
    const selectedAnswer = ref(null) // 单选题答案（索引）
    const selectedAnswers = ref([])  // 多选题答案（索引数组）
    const userAnswers = ref([])      // 所有题目的答案（数组或索引，按照打乱后的顺序）
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
      if (!questions.value || questions.value.length === 0) {
        return 0
      }
      return Math.round(((currentQuestionIndex.value + 1) / questions.value.length) * 100)
    })

    // 判断答案是否正确（支持单选和多选）
    const isAnswerCorrect = (index) => {
      const question = questions.value[index]
      const userAnswer = userAnswers.value[index]
      
      if (!question || userAnswer === null || userAnswer === undefined) {
        return false
      }
      
      // 如果正确答案是数组格式（多选题或统一使用数组格式的单选题）
      if (Array.isArray(question.answer)) {
        // 如果用户答案也是数组格式
        if (Array.isArray(userAnswer)) {
          const correctAnswer = [...question.answer].sort((a, b) => a - b)
          const userAnswerArray = [...userAnswer].sort((a, b) => a - b)
          return correctAnswer.length === userAnswerArray.length &&
                 correctAnswer.every((val, idx) => val === userAnswerArray[idx])
        } else {
          // 单选题：用户答案是单个数字，正确答案是数组（如[0]）
          return question.answer.length === 1 && question.answer[0] === userAnswer
        }
      } else {
        // 如果正确答案是单个数字格式（兼容旧格式）
        if (Array.isArray(userAnswer)) {
          // 单选题：用户答案是数组，正确答案是单个数字
          return userAnswer.length === 1 && userAnswer[0] === question.answer
        } else {
          // 单选题：都是单个数字
          return userAnswer === question.answer
        }
      }
    }

    const correctCount = computed(() => {
      return userAnswers.value.filter((answer, index) => 
        isAnswerCorrect(index)
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
          // 获取题库详情（包含题目）
          const bank = await examStore.getQuestionBankDetails(route.params.id)
          console.log('获取到的题库详情:', bank)
          console.log('bank.questions:', bank?.questions)
          console.log('bank.Questions:', bank?.Questions)
          
          if (!bank) {
            ElMessage.error('题库不存在')
            router.push('/practice')
            return
          }
          
          // 检查返回的数据结构，后端返回的是小写questions（JSON tag）
          let questionList = null
          
          // 优先检查小写questions（JSON序列化后的字段名）
          if (bank.questions && Array.isArray(bank.questions)) {
            console.log('找到bank.questions，长度:', bank.questions.length)
            questionList = bank.questions
          } 
          // 兼容检查大写Questions（以防万一）
          else if (bank.Questions && Array.isArray(bank.Questions)) {
            console.log('找到bank.Questions，长度:', bank.Questions.length)
            questionList = bank.Questions
          }
          
          // 如果bank中没有questions或questions为空，尝试使用getQuestions API单独获取
          if (!questionList || questionList.length === 0) {
            console.log('bank中没有题目数据，尝试使用单独的API获取题目...')
            try {
              const questionsFromAPI = await examStore.getQuestions(route.params.id)
              console.log('单独API获取到的题目列表:', questionsFromAPI)
              console.log('题目数量:', questionsFromAPI?.length)
              
              if (questionsFromAPI && Array.isArray(questionsFromAPI) && questionsFromAPI.length > 0) {
                questionList = questionsFromAPI
              }
            } catch (apiError) {
              console.error('调用getQuestions API失败:', apiError)
            }
          }
          
          if (questionList && Array.isArray(questionList) && questionList.length > 0) {
            console.log('成功获取题目，数量:', questionList.length)
            questions.value = [...questionList]
          } else {
            console.error('无法获取题目列表')
            console.error('bank完整数据:', JSON.stringify(bank, null, 2))
            ElMessage.error('题库为空，请检查题库是否有题目')
            router.push('/practice')
            return
          }
        }

        if (!questions.value || questions.value.length === 0) {
          ElMessage.error('题库为空')
          router.push('/practice')
          return
        }

        // 保存原始题目顺序（不打乱）
        originalQuestions.value = [...questions.value]
        
        // 打乱题目顺序
        questions.value = shuffleArray(questions.value)
        
        // 创建索引映射：打乱后的索引 -> 原始索引
        questionIndexMap.value = new Map()
        questions.value.forEach((shuffledQ, shuffledIndex) => {
          const originalIndex = originalQuestions.value.findIndex(q => q.id === shuffledQ.id)
          if (originalIndex !== -1) {
            questionIndexMap.value.set(shuffledIndex, originalIndex)
          }
        })
        
        userAnswers.value = new Array(questions.value.length).fill(null)
        selectedAnswer.value = null
        selectedAnswers.value = []
        
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

    // 判断是否已选择答案
    const isAnswerSelected = computed(() => {
      const question = currentQuestion.value
      if (!question) return false
      
      if (question.is_multiple) {
        // 多选题：至少选择一个
        return selectedAnswers.value.length > 0
      } else {
        // 单选题：必须选择一个
        return selectedAnswer.value !== null
      }
    })

    // 判断是否所有题目都已作答
    const isAllAnswered = computed(() => {
      if (questions.value.length === 0) return false
      // 检查所有题目是否都有答案（不为null）
      return questions.value.every((_, index) => {
        const answer = userAnswers.value[index]
        if (answer === null || answer === undefined) {
          return false
        }
        // 多选题：至少有一个答案
        if (Array.isArray(answer)) {
          return answer.length > 0
        }
        // 单选题：答案不为null即可
        return true
      })
    })

    // 单选题：选择答案
    const selectAnswer = (index) => {
      selectedAnswer.value = index
      userAnswers.value[currentQuestionIndex.value] = index
    }

    // 多选题：切换答案选择
    const toggleAnswer = (index) => {
      const currentIndex = selectedAnswers.value.indexOf(index)
      if (currentIndex > -1) {
        // 取消选择
        selectedAnswers.value.splice(currentIndex, 1)
      } else {
        // 添加选择（最多10个选项）
        if (selectedAnswers.value.length < currentQuestion.value.options.length) {
          selectedAnswers.value.push(index)
        }
      }
      // 更新用户答案（保存数组的副本）
      userAnswers.value[currentQuestionIndex.value] = [...selectedAnswers.value]
    }

    // 判断选项是否被选中（多选题）
    const isOptionSelected = (index) => {
      if (!currentQuestion.value || !currentQuestion.value.is_multiple) {
        return false
      }
      return selectedAnswers.value.includes(index)
    }

    const nextQuestion = () => {
      const question = currentQuestion.value
      if (!question) return
      
      if (question.is_multiple) {
        if (selectedAnswers.value.length === 0) {
          ElMessage.warning('请至少选择一个答案')
          return
        }
        // 确保多选题的答案已保存
        userAnswers.value[currentQuestionIndex.value] = [...selectedAnswers.value]
      } else {
        if (selectedAnswer.value === null) {
          ElMessage.warning('请选择一个答案')
          return
        }
        // 确保单选题的答案已保存
        userAnswers.value[currentQuestionIndex.value] = selectedAnswer.value
      }

      if (currentQuestionIndex.value === questions.value.length - 1) {
        finishExam()
      } else {
        currentQuestionIndex.value++
        loadCurrentQuestionAnswer()
      }
    }

    const previousQuestion = () => {
      if (currentQuestionIndex.value > 0) {
        currentQuestionIndex.value--
        loadCurrentQuestionAnswer()
      }
    }

    const jumpToQuestion = (index) => {
      currentQuestionIndex.value = index
      loadCurrentQuestionAnswer()
    }

    // 加载当前题目的答案
    const loadCurrentQuestionAnswer = () => {
      const question = currentQuestion.value
      if (!question) return
      
      const answer = userAnswers.value[currentQuestionIndex.value]
      
      if (question.is_multiple) {
        // 多选题：加载数组
        if (Array.isArray(answer)) {
          selectedAnswers.value = [...answer]
        } else {
          selectedAnswers.value = []
        }
        selectedAnswer.value = null
      } else {
        // 单选题：加载单个值
        selectedAnswer.value = answer !== null && answer !== undefined ? answer : null
        selectedAnswers.value = []
      }
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

      // 保存错题（支持多选）
      if (route.params.id !== 'wrong-questions' && !route.params.id.startsWith('wrong-questions') && route.name !== 'WrongQuestionsExam') {
        for (let index = 0; index < questions.value.length; index++) {
          const question = questions.value[index]
          if (!isAnswerCorrect(index)) {
            await examStore.addWrongQuestion(question, route.params.id)
          }
        }
      } else if ((route.params.id && route.params.id.startsWith('wrong-questions') && !route.params.id.includes('/')) || route.name === 'WrongQuestionsExam') {
        // 从错题库中移除答对的题目（错题练习时）
        for (let index = 0; index < questions.value.length; index++) {
          const question = questions.value[index]
          if (isAnswerCorrect(index)) {
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
      if (showReview.value) {
        // 显示所有题目的答案数据，用于调试
        console.log('=== 答案回顾调试信息 ===')
        console.log('题目总数:', questions.value.length)
        console.log('用户答案数组:', userAnswers.value)
        questions.value.forEach((q, idx) => {
          console.log(`题目 ${idx + 1}:`, {
            题目: q.question.substring(0, 30) + '...',
            用户答案: userAnswers.value[idx],
            正确答案: q.answer,
            是否正确: isAnswerCorrect(idx),
            题目类型: q.is_multiple ? '多选' : '单选'
          })
        })
        console.log('========================')
      }
    }

    const submitEarly = () => {
      ElMessageBox.confirm(
        '确定要提前交卷吗？未作答的题目将按照未答处理。',
        '提前交卷确认',
        {
          confirmButtonText: '确定交卷',
          cancelButtonText: '取消',
          type: 'warning',
        }
      ).then(() => {
        finishExam()
      }).catch(() => {
        // 用户取消
      })
    }

    const submitExam = () => {
      ElMessageBox.confirm(
        '确定要交卷吗？交卷后将无法修改答案。',
        '交卷确认',
        {
          confirmButtonText: '确定交卷',
          cancelButtonText: '取消',
          type: 'success',
        }
      ).then(() => {
        finishExam()
      }).catch(() => {
        // 用户取消
      })
    }

    const restartExam = () => {
      currentQuestionIndex.value = 0
      selectedAnswer.value = null
      selectedAnswers.value = []
      userAnswers.value = new Array(questions.value.length).fill(null)
      showResult.value = false
      showReview.value = false
      elapsedTime.value = 0
      
      // 重新打乱题目
      questions.value = shuffleArray(questions.value)
      loadCurrentQuestionAnswer()
      startTimer()
    }

    // 判断是否为正确答案（用于答案回顾）
    const isCorrectAnswer = (question, optionIndex) => {
      if (!question || question.answer === null || question.answer === undefined) {
        return false
      }
      
      // 如果答案是数组格式（多选题或统一使用数组格式的单选题）
      if (Array.isArray(question.answer)) {
        return question.answer.includes(optionIndex)
      } else {
        // 如果答案是单个数字格式（兼容旧格式的单选题）
        return question.answer === optionIndex
      }
    }

    // 判断是否为错误答案（用于答案回顾）
    const isWrongAnswer = (question, optionIndex, questionIndex) => {
      // questionIndex 是打乱后的索引，userAnswers 也是按照打乱后的顺序保存的，所以直接使用
      const userAnswer = userAnswers.value[questionIndex]
      
      // 添加调试日志
      if (questionIndex === 0) {
        console.log('isWrongAnswer 调试 - questionIndex:', questionIndex)
        console.log('isWrongAnswer 调试 - userAnswer:', userAnswer)
        console.log('isWrongAnswer 调试 - optionIndex:', optionIndex)
        console.log('isWrongAnswer 调试 - isCorrectAnswer:', isCorrectAnswer(question, optionIndex))
      }
      
      if (!question || userAnswer === null || userAnswer === undefined) {
        return false
      }
      
      // 如果用户答案是数组格式
      if (Array.isArray(userAnswer)) {
        // 用户选择了这个选项，但这不是正确答案
        return userAnswer.includes(optionIndex) && !isCorrectAnswer(question, optionIndex)
      } else {
        // 用户答案是单个数字格式
        // 用户选择了这个选项，但这不是正确答案
        return userAnswer === optionIndex && !isCorrectAnswer(question, optionIndex)
      }
    }

    // 判断是否为用户选择的答案（用于答案回顾）
    const isUserSelectedAnswer = (question, optionIndex, questionIndex) => {
      // questionIndex 是打乱后的索引，userAnswers 也是按照打乱后的顺序保存的，所以直接使用
      const userAnswer = userAnswers.value[questionIndex]
      
      if (!question || userAnswer === null || userAnswer === undefined) {
        return false
      }
      
      // 如果用户答案是数组格式（多选题或统一使用数组格式的单选题）
      if (Array.isArray(userAnswer)) {
        return userAnswer.includes(optionIndex)
      } else {
        // 用户答案是单个数字格式
        return userAnswer === optionIndex
      }
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
      toggleAnswer,
      isOptionSelected,
      isAnswerSelected,
      isAllAnswered,
      nextQuestion,
      previousQuestion,
      jumpToQuestion,
      loadCurrentQuestionAnswer,
      reviewAnswers,
      submitEarly,
      submitExam,
      restartExam,
      formatTime,
      isAnswerCorrect,
      isCorrectAnswer,
      isWrongAnswer,
      isUserSelectedAnswer
    }
  }
}
</script>

<style scoped>
.exam {
  min-height: calc(100vh - 120px);
  display: flex;
  flex-direction: column;
  position: relative;
  contain: layout style;
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
  height: 100px !important;
  max-height: 100px !important;
  min-height: 100px !important;
  width: 100% !important;
  min-width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box;
  overflow: hidden;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  position: relative;
  z-index: 2;
}

.exam-info {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.exam-info h2 {
  color: white;
  margin-bottom: 10px;
  font-size: 24px;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  flex-shrink: 0;
}

.exam-progress {
  color: rgba(255, 255, 255, 0.9);
  flex-shrink: 0;
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
  height: 200px !important;
  max-height: 200px !important;
  min-height: 200px !important;
  width: 100% !important;
  min-width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box;
  display: flex !important;
  flex-direction: column !important;
  overflow: hidden;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  position: relative;
  z-index: 1;
  contain: layout size style;
  will-change: auto;
  margin-top: 0 !important;
  clear: both;
}

.navigator-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  height: 40px !important;
  max-height: 40px !important;
  min-height: 40px !important;
  width: 100% !important;
  min-width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box;
  overflow: hidden;
  position: relative;
}

.navigator-header h3 {
  color: white;
  margin: 0;
  font-size: 18px;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  height: auto;
  width: auto;
  min-width: 0;
  max-width: none;
}

.legend {
  display: flex;
  gap: 15px;
  flex-shrink: 0;
  white-space: nowrap;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 5px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 12px;
  flex-shrink: 0;
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
  flex-shrink: 0;
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
  flex: 0 0 auto !important;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  width: 100% !important;
  min-width: 100% !important;
  max-width: 100% !important;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
  max-height: calc(200px - 40px - 15px - 40px) !important; /* 总高度200px - header高度40px - header margin-bottom 15px - padding 40px(上下各20px) */
  height: calc(200px - 40px - 15px - 40px) !important;
  box-sizing: border-box;
  position: relative;
  contain: layout size style;
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

.exam-content {
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
  position: relative;
  z-index: 0;
  overflow-y: auto;
  contain: layout style;
  margin-top: 0 !important;
  clear: both;
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

.option-checkbox {
  margin-right: 10px;
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
  padding: 12px 15px;
  margin-bottom: 10px;
  border-radius: 8px;
  font-size: 14px;
  border: 2px solid transparent;
  transition: all 0.3s ease;
}

.review-option.correct {
  background: #f0f9ff;
  color: #303133;
  border: 2px solid #67c23a;
  font-weight: 500;
}

.review-option.correct .option-label {
  background: #67c23a;
  color: white;
  font-weight: bold;
}

.review-option.wrong {
  background: #fef0f0;
  color: #303133;
  border: 2px solid #f56c6c;
  font-weight: 500;
}

.review-option.wrong .option-label {
  background: #f56c6c;
  color: white;
  font-weight: bold;
}

.review-option.user-selected {
  background: #e8f5e9;
  border-color: #67c23a;
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.2);
}

.review-option.user-selected .option-label {
  background: #67c23a;
  color: white;
  font-weight: bold;
}

/* 确保错误答案样式优先级更高 */
.review-option.wrong.correct {
  background: #fef0f0;
  border-color: #f56c6c;
}

.review-option.wrong.correct .option-label {
  background: #f56c6c;
  color: white;
}

.review-option .option-label {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  margin-right: 12px;
  font-weight: 500;
  flex-shrink: 0;
}

.answer-tags {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
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
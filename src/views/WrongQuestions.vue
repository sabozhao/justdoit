<template>
  <div class="wrong-questions">
    <div class="wrong-questions-header">
      <h1>错题库</h1>
      <div class="header-actions">
        <el-button
          v-if="wrongQuestions.length > 0"
          type="primary"
          @click="practiceAllWrongQuestions"
        >
          <el-icon><Edit /></el-icon>
          练习全部错题
        </el-button>
        <el-button
          v-if="wrongQuestions.length > 0"
          type="danger"
          @click="clearAllWrongQuestions"
        >
          <el-icon><Delete /></el-icon>
          清空错题库
        </el-button>
      </div>
    </div>

    <!-- 分类筛选 -->
    <div class="wrong-questions-filters">
      <el-select 
        v-model="selectedCategoryId" 
        placeholder="选择分类" 
        clearable 
        @change="handleCategoryChange"
        style="width: 200px;"
      >
        <el-option label="全部分类" value="" />
        <el-option label="未分类" value="uncategorized" />
        <el-option 
          v-for="cat in flatCategories" 
          :key="cat.id" 
          :label="cat.name" 
          :value="cat.id"
        >
          <span>{{ getCategoryPath(cat) }}</span>
        </el-option>
      </el-select>
    </div>

    <div class="wrong-questions-content">
      <!-- 按分类分组显示 -->
      <div v-if="groupedByCategory.length > 0">
        <div
          v-for="categoryGroup in groupedByCategory"
          :key="categoryGroup.categoryId || 'uncategorized'"
          class="category-group"
        >
          <div class="category-header">
            <h2>
              <el-icon><FolderOpened /></el-icon>
              {{ categoryGroup.categoryName }}
            </h2>
            <span class="category-count">{{ categoryGroup.totalCount }} 道错题</span>
          </div>

          <!-- 该分类下的题库分组 -->
          <div
            v-for="bankGroup in categoryGroup.banks"
            :key="bankGroup.bankId"
            class="question-group"
          >
            <div class="group-header">
              <div class="group-title">
                <h3>{{ bankGroup.bankName }}</h3>
                <el-tag v-if="bankGroup.isPublic" type="success" size="small">共享题库</el-tag>
                <el-tag v-else type="info" size="small">我的题库</el-tag>
              </div>
              <div class="group-actions">
                <el-button size="small" @click="practiceGroupQuestions(bankGroup.bankId)">
                  练习该题库错题 ({{ bankGroup.questions.length }})
                </el-button>
              </div>
            </div>

            <div class="questions-list">
              <div
                v-for="(question, index) in bankGroup.questions"
                :key="question.id"
                class="question-item"
              >
                <div class="question-header">
                  <span class="question-number">第 {{ index + 1 }} 题</span>
                  <div style="display: flex; gap: 10px; align-items: center;">
                    <el-tag v-if="question.type === 'judgment'" type="info" size="small">
                      判断题
                    </el-tag>
                    <el-tag v-else-if="question.is_multiple" type="warning" size="small">
                      多选题
                    </el-tag>
                    <el-tag v-else type="primary" size="small">
                      单选题
                    </el-tag>
                    <el-button
                      type="text"
                      size="small"
                      @click="removeFromWrongQuestions(question.id)"
                    >
                      <el-icon><Close /></el-icon>
                      移除
                    </el-button>
                  </div>
                </div>

                <div class="question-content">
                  <div class="question-text">{{ question.question }}</div>
                  
                  <div class="question-options">
                    <div
                      v-for="(option, optionIndex) in question.options"
                      :key="optionIndex"
                      class="option-item"
                      :class="{ 'correct': isCorrectAnswer(question, optionIndex) }"
                    >
                      <span class="option-label">{{ String.fromCharCode(65 + optionIndex) }}</span>
                      <span class="option-text">{{ option }}</span>
                      <el-tag v-if="isCorrectAnswer(question, optionIndex)" type="success" size="small">
                        正确答案
                      </el-tag>
                    </div>
                  </div>

                  <div v-if="question.explanation" class="question-explanation">
                    <strong>解析：</strong>{{ question.explanation }}
                  </div>

                  <div class="question-meta">
                    <span class="meta-item">
                      <el-icon><Calendar /></el-icon>
                      添加时间：{{ formatDate(question.added_at) }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <el-empty description="暂无错题">
          <template #image>
            <el-icon class="empty-icon"><SuccessFilled /></el-icon>
          </template>
          <template #description>
            <p>太棒了！你还没有错题</p>
            <p>继续保持，争取全部答对</p>
          </template>
          <el-button type="primary" @click="$router.push('/my-library')">
            去刷题
          </el-button>
        </el-empty>
      </div>
    </div>
  </div>
</template>

<script>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useExamStore } from '../stores/exam'
import { categoryAPI } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FolderOpened, Close, Calendar, Edit, Delete, SuccessFilled } from '@element-plus/icons-vue'

export default {
  name: 'WrongQuestions',
  components: {
    FolderOpened,
    Close,
    Calendar,
    Edit,
    Delete,
    SuccessFilled
  },
  setup() {
    const router = useRouter()
    const examStore = useExamStore()

    const wrongQuestions = computed(() => examStore.wrongQuestions || [])
    const categories = ref([])
    const selectedCategoryId = ref('')

    // 扁平化分类列表（用于下拉选择）
    const flatCategories = computed(() => {
      const flatten = (cats, parentPath = '') => {
        let result = []
        for (const cat of cats) {
          const path = parentPath ? `${parentPath} / ${cat.name}` : cat.name
          result.push({ ...cat, path })
          if (cat.children && cat.children.length > 0) {
            result = result.concat(flatten(cat.children, path))
          }
        }
        return result
      }
      return flatten(categories.value)
    })
    
    // 获取分类路径
    const getCategoryPath = (cat) => {
      return cat.path || cat.name
    }

    // 按分类分组错题
    const groupedByCategory = computed(() => {
      const categoryGroups = {}
      
      wrongQuestions.value.forEach(question => {
        // 获取分类ID（如果题库有分类信息）
        const categoryId = question.category_id || null
        const categoryName = question.category_name || '未分类'
        const categoryKey = categoryId || 'uncategorized'
        
        // 筛选：如果选择了分类，只显示该分类
        if (selectedCategoryId.value) {
          if (selectedCategoryId.value === 'uncategorized' && categoryId !== null) {
            return // 选择了"未分类"，但当前题目有分类，跳过
          }
          if (selectedCategoryId.value !== 'uncategorized' && categoryId !== selectedCategoryId.value) {
            return // 选择了特定分类，但当前题目不是该分类，跳过
          }
        }
        
        if (!categoryGroups[categoryKey]) {
          categoryGroups[categoryKey] = {
            categoryId: categoryId,
            categoryName: categoryName,
            totalCount: 0,
            banks: {}
          }
        }
        
        // 按题库分组
        const bankId = question.bank_id
        if (!categoryGroups[categoryKey].banks[bankId]) {
          categoryGroups[categoryKey].banks[bankId] = {
            bankId: bankId,
            bankName: question.bank_name || '未知题库',
            isPublic: question.is_public || false,
            questions: []
          }
        }
        
        categoryGroups[categoryKey].banks[bankId].questions.push(question)
        categoryGroups[categoryKey].totalCount++
      })
      
      // 转换为数组并排序（未分类放在最后）
      const result = Object.values(categoryGroups).map(group => ({
        ...group,
        banks: Object.values(group.banks)
      }))
      
      // 排序：有分类的在前，未分类在后
      result.sort((a, b) => {
        if (a.categoryId === null && b.categoryId !== null) return 1
        if (a.categoryId !== null && b.categoryId === null) return -1
        return (a.categoryName || '').localeCompare(b.categoryName || '')
      })
      
      return result
    })

    // 加载分类
    const loadCategories = async () => {
      try {
        const result = await categoryAPI.getAll()
        categories.value = result || []
      } catch (error) {
        console.error('加载分类失败:', error)
        // 不显示错误，因为分类是可选的
      }
    }

    // 处理分类切换
    const handleCategoryChange = () => {
      // 分类切换时，groupedByCategory 会自动更新
    }

    const practiceAllWrongQuestions = () => {
      router.push('/exam/wrong-questions')
    }

    const practiceGroupQuestions = (bankId) => {
      // 使用新的路由名称导航到特定题库的错题练习
      router.push({ name: 'WrongQuestionsExam', params: { bankId } })
    }

    const removeFromWrongQuestions = async (questionId) => {
      try {
        await ElMessageBox.confirm('确定要从错题库中移除这道题吗？', '确认移除', {
          type: 'warning'
        })
        await examStore.removeWrongQuestion(questionId)
      } catch {
        // 用户取消
      }
    }

    const clearAllWrongQuestions = async () => {
      try {
        await ElMessageBox.confirm(
          '确定要清空所有错题吗？此操作不可恢复！',
          '确认清空',
          {
            type: 'warning',
            confirmButtonText: '确定清空',
            cancelButtonText: '取消'
          }
        )
        
        await examStore.clearAllWrongQuestions()
      } catch {
        // 用户取消
      }
    }

    // 判断是否为正确答案（支持多选）
    const isCorrectAnswer = (question, optionIndex) => {
      if (!question || question.answer === null || question.answer === undefined) {
        return false
      }
      
      if (question.is_multiple && Array.isArray(question.answer)) {
        return question.answer.includes(optionIndex)
      } else if (!question.is_multiple && !Array.isArray(question.answer)) {
        return question.answer === optionIndex
      }
      
      // 兼容旧数据：如果answer是数组但is_multiple为false，或反之
      if (Array.isArray(question.answer)) {
        return question.answer.includes(optionIndex)
      } else {
        return question.answer === optionIndex
      }
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleString('zh-CN')
    }

    // 页面加载时获取数据
    onMounted(async () => {
      await loadCategories()
      await examStore.loadWrongQuestions()
    })

    return {
      wrongQuestions,
      groupedByCategory,
      categories,
      flatCategories,
      getCategoryPath,
      selectedCategoryId,
      handleCategoryChange,
      practiceAllWrongQuestions,
      practiceGroupQuestions,
      removeFromWrongQuestions,
      clearAllWrongQuestions,
      formatDate,
      isCorrectAnswer
    }
  }
}
</script>

<style scoped>
.wrong-questions {
  min-height: calc(100vh - 120px);
}

.wrong-questions-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding: 20px 30px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 15px;
  backdrop-filter: blur(10px);
}

.wrong-questions-header h1 {
  color: white;
  font-size: 28px;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 15px;
}

.question-group {
  margin-bottom: 40px;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 15px;
  overflow: hidden;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
}

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 30px;
  background: #f8f9fa;
  border-bottom: 1px solid #e9ecef;
}

.group-header h3 {
  color: #303133;
  margin: 0;
  font-size: 20px;
}

.questions-list {
  padding: 0;
}

.question-item {
  border-bottom: 1px solid #f0f0f0;
  padding: 25px 30px;
}

.question-item:last-child {
  border-bottom: none;
}

.question-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.question-number {
  background: #409eff;
  color: white;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: bold;
}

.question-content {
  padding-left: 0;
}

.question-text {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 15px;
  line-height: 1.6;
}

.question-options {
  margin-bottom: 15px;
}

.option-item {
  display: flex;
  align-items: center;
  padding: 10px 15px;
  margin-bottom: 8px;
  border-radius: 8px;
  background: #f8f9fa;
  transition: all 0.3s ease;
}

.option-item.correct {
  background: #f0f9ff;
  border: 1px solid #67c23a;
}

.option-label {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: bold;
  margin-right: 12px;
  flex-shrink: 0;
}

.option-item.correct .option-label {
  background: #67c23a;
  color: white;
}

.option-text {
  flex: 1;
  margin-right: 10px;
}

.question-explanation {
  background: #f8f9fa;
  padding: 15px;
  border-radius: 8px;
  margin-bottom: 15px;
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
}

.question-meta {
  display: flex;
  gap: 20px;
  color: #909399;
  font-size: 12px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.empty-state {
  text-align: center;
  padding: 80px 0;
}

.empty-icon {
  font-size: 80px;
  color: #67c23a;
  margin-bottom: 20px;
}

.empty-state p {
  color: rgba(255, 255, 255, 0.8);
  font-size: 16px;
  margin: 5px 0;
}

.wrong-questions-filters {
  margin-bottom: 20px;
  padding: 15px 30px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  backdrop-filter: blur(10px);
}

.category-group {
  margin-bottom: 40px;
}

.category-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 30px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 10px;
  margin-bottom: 20px;
  backdrop-filter: blur(10px);
}

.category-header h2 {
  color: white;
  font-size: 22px;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.category-count {
  color: rgba(255, 255, 255, 0.9);
  font-size: 14px;
  background: rgba(255, 255, 255, 0.2);
  padding: 5px 15px;
  border-radius: 15px;
}

.group-title {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>
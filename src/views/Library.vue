<template>
  <div class="library">
    <div class="library-header">
      <h1>题库管理</h1>
      <div class="header-buttons">
        <el-button type="success" size="large" @click="showUploadDialog = true">
          <el-icon><Upload /></el-icon>
          上传题库文件
        </el-button>
        <el-button type="primary" size="large" @click="showCreateDialog = true">
          <el-icon><Plus /></el-icon>
          手动创建题库
        </el-button>
      </div>
    </div>

    <div class="library-content">
      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="3" animated />
      </div>
      
      <div v-else-if="displayedBanks.length > 0" class="banks-grid">
        <div class="bank-card-wrapper" v-for="bank in displayedBanks" :key="bank.id">
          <div class="bank-card-header">
            <h3>{{ bank.name }}</h3>
            <el-dropdown @command="handleCommand" trigger="click">
              <el-button type="text" class="more-btn">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item :command="`practice-${bank.id}`">
                    <el-icon><VideoPlay /></el-icon>
                    开始刷题
                  </el-dropdown-item>
                  <el-dropdown-item :command="`manage-${bank.id}`">
                    <el-icon><Edit /></el-icon>
                    管理题目
                  </el-dropdown-item>
                  <el-dropdown-item :command="`delete-${bank.id}`" class="danger-item">
                    <el-icon><Delete /></el-icon>
                    删除题库
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          
          <div class="bank-card-body">
            <p class="bank-description">{{ bank.description || '暂无描述' }}</p>
            <div class="bank-stats">
              <div class="stat-item">
                <el-icon><Document /></el-icon>
                <span>{{ bank.question_count || 0 }} 道题</span>
              </div>
              <div class="stat-item">
                <el-icon><Calendar /></el-icon>
                <span>{{ formatDate(bank.created_at) }}</span>
              </div>
            </div>
          </div>
          
          <div class="bank-card-footer">
            <el-button type="primary" @click="startPractice(bank.id)" class="practice-btn">
              开始练习
            </el-button>
          </div>
        </div>
      </div>

      <div v-else class="empty-state">
        <el-empty description="暂无题库">
          <el-button type="primary" @click="showUploadDialog = true">
            上传第一个题库
          </el-button>
        </el-empty>
      </div>
    </div>

    <!-- 上传对话框 -->
    <el-dialog v-model="showUploadDialog" title="上传题库" width="600px">
      <el-form :model="uploadForm" label-width="80px">
        <el-form-item label="题库名称" required>
          <el-input v-model="uploadForm.name" placeholder="请输入题库名称" />
        </el-form-item>
        <el-form-item label="题库描述">
          <el-input v-model="uploadForm.description" type="textarea" placeholder="请输入题库描述（可选）" />
        </el-form-item>
        <el-form-item label="选择文件" required>
          <el-upload
            :auto-upload="false"
            :show-file-list="true"
            :limit="1"
            accept=".json,.xlsx,.xls,.csv"
            :on-change="handleFileChange"
          >
            <el-button type="primary">选择文件</el-button>
            <template #tip>
              <div class="el-upload__tip">
                支持 JSON、Excel (.xlsx, .xls) 和 CSV 格式的题库文件
              </div>
            </template>
          </el-upload>
        </el-form-item>
      </el-form>
      
      <el-collapse>
        <el-collapse-item title="JSON格式示例" name="example">
          <pre class="json-example">{{ jsonExample }}</pre>
        </el-collapse-item>
        <el-collapse-item title="Excel/CSV格式示例" name="excel-example">
          <div class="format-example">
            <p><strong>Excel/CSV文件应包含以下列：</strong></p>
            <table class="format-table">
              <thead>
                <tr>
                  <th>题目</th>
                  <th>选项A</th>
                  <th>选项B</th>
                  <th>选项C</th>
                  <th>选项D</th>
                  <th>正确答案</th>
                  <th>解析</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>这是一道示例题目？</td>
                  <td>选项A内容</td>
                  <td>选项B内容</td>
                  <td>选项C内容</td>
                  <td>选项D内容</td>
                  <td>A</td>
                  <td>答案解析（可选）</td>
                </tr>
              </tbody>
            </table>
            <p class="format-note">
              <strong>注意：</strong><br>
              • 正确答案可以填写 A/B/C/D 或 1/2/3/4<br>
              • 选项C和D是可选的，至少需要A和B两个选项<br>
              • 解析列是可选的<br>
              • 支持中英文列名
            </p>
          </div>
        </el-collapse-item>
      </el-collapse>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showUploadDialog = false">取消</el-button>
          <el-button type="primary" @click="uploadQuestionBank">确定上传</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 手动创建题库对话框 -->
    <el-dialog v-model="showCreateDialog" title="手动创建题库" width="800px">
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="题库名称" required>
          <el-input v-model="createForm.name" placeholder="请输入题库名称" />
        </el-form-item>
        <el-form-item label="题库描述">
          <el-input v-model="createForm.description" type="textarea" placeholder="请输入题库描述（可选）" />
        </el-form-item>
        <el-form-item label="题目列表" required>
          <div class="questions-container">
            <div v-for="(question, index) in createForm.questions" :key="index" class="question-item">
              <div class="question-header">
                <span>第 {{ index + 1 }} 题</span>
                <el-button type="danger" size="small" @click="removeQuestion(index)" v-if="createForm.questions.length > 1">
                  删除
                </el-button>
              </div>
              <el-input v-model="question.question" placeholder="请输入题目内容" class="question-input" />
              <div class="options-container">
                <div v-for="(option, optionIndex) in question.options" :key="optionIndex" class="option-item">
                  <span class="option-label">{{ String.fromCharCode(65 + optionIndex) }}:</span>
                  <el-input v-model="question.options[optionIndex]" placeholder="请输入选项内容" />
                  <el-button type="danger" size="small" @click="removeOption(index, optionIndex)" v-if="question.options.length > 2">
                    删除
                  </el-button>
                </div>
                <el-button type="text" @click="addOption(index)" v-if="question.options.length < 4">
                  + 添加选项
                </el-button>
              </div>
              <div class="answer-container">
                <span>正确答案：</span>
                <el-radio-group v-model="question.answer">
                  <el-radio v-for="(option, optionIndex) in question.options" :key="optionIndex" :label="optionIndex">
                    {{ String.fromCharCode(65 + optionIndex) }}
                  </el-radio>
                </el-radio-group>
              </div>
              <el-input v-model="question.explanation" placeholder="答案解析（可选）" class="explanation-input" />
            </div>
            <el-button type="primary" @click="addQuestion" class="add-question-btn">
              + 添加题目
            </el-button>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button type="primary" @click="createQuestionBank">确定创建</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 题目管理对话框 -->
    <el-dialog v-model="showManageDialog" :title="currentBank ? `${currentBank.name} - 题目管理` : '题目管理'" width="900px">
      <div v-if="currentBank">
        <div style="margin-bottom: 20px;">
          <h3>题库: {{ currentBank.name }}</h3>
          <p>{{ currentBank.description || '暂无描述' }}</p>
        </div>
        
        <div style="margin-bottom: 20px;">
          <el-button type="primary" @click="addNewQuestion">
            <el-icon><Plus /></el-icon>
            添加题目
          </el-button>
        </div>
        
        <el-table :data="bankQuestions" style="width: 100%">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="question" label="题目内容" min-width="200">
            <template #default="scope">
              <div style="max-height: 60px; overflow: hidden; text-overflow: ellipsis;">
                {{ scope.row.question }}
              </div>
            </template>
          </el-table-column>
          <el-table-column label="选项" min-width="300">
            <template #default="scope">
              <div v-for="(option, index) in scope.row.options" :key="index">
                {{ String.fromCharCode(65 + index) }}. {{ option }}
              </div>
            </template>
          </el-table-column>
          <el-table-column label="正确答案" width="100">
            <template #default="scope">
              {{ String.fromCharCode(65 + scope.row.answer) }}
            </template>
          </el-table-column>
          <el-table-column prop="explanation" label="解析" min-width="200" />
          <el-table-column label="操作" width="150">
            <template #default="scope">
              <el-button size="small" @click="editQuestion(scope.row)">编辑</el-button>
              <el-button size="small" type="danger" @click="deleteQuestion(scope.row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 题目编辑对话框 -->
    <el-dialog v-model="showQuestionEditDialog" :title="currentQuestion.id ? '编辑题目' : '添加题目'" width="600px">
      <el-form :model="currentQuestion" label-width="80px">
        <el-form-item label="题目内容">
          <el-input v-model="currentQuestion.question" type="textarea" :rows="3" placeholder="请输入题目内容" />
        </el-form-item>
        <el-form-item label="选项">
          <div v-for="(option, index) in currentQuestion.options" :key="index" style="margin-bottom: 10px;">
            <el-input v-model="currentQuestion.options[index]" :placeholder="`选项 ${String.fromCharCode(65 + index)}`">
              <template #prepend>{{ String.fromCharCode(65 + index) }}</template>
            </el-input>
          </div>
        </el-form-item>
        <el-form-item label="正确答案">
          <el-select v-model="currentQuestion.answer">
            <el-option v-for="(option, index) in currentQuestion.options" 
                       :key="index" 
                       :label="String.fromCharCode(65 + index)" 
                       :value="index"
                       :disabled="!option.trim()">
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="答案解析">
          <el-input v-model="currentQuestion.explanation" type="textarea" :rows="2" placeholder="请输入题目解析（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showQuestionEditDialog = false">取消</el-button>
        <el-button type="primary" @click="saveQuestion">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useExamStore } from '@/stores/exam'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Upload, MoreFilled, VideoPlay, Delete, Document, Calendar, Edit } from '@element-plus/icons-vue'

export default {
  name: 'Library',
  components: {
    Plus,
    Upload,
    MoreFilled,
    VideoPlay,
    Delete,
    Document,
    Calendar
  },
  setup() {
    const router = useRouter()
    const examStore = useExamStore()
    
    const showUploadDialog = ref(false)
    const showCreateDialog = ref(false)
    const showManageDialog = ref(false)
    const selectedFile = ref(null)
    const uploadForm = reactive({
      name: '',
      description: ''
    })
    const createForm = reactive({
      name: '',
      description: '',
      questions: [
        {
          question: '',
          options: ['', ''],
          answer: 0,
          explanation: ''
        }
      ]
    })
    
    // 题目管理相关状态
    const currentBank = ref(null)
    const bankQuestions = ref([])
    const showQuestionEditDialog = ref(false)
    const currentQuestion = ref({
      id: '',
      question: '',
      options: ['', '', '', ''],
      answer: 0,
      explanation: ''
    })

    const questionBanks = computed(() => {
      const banks = examStore.questionBanks
      if (!Array.isArray(banks)) {
        console.warn('questionBanks is not an array:', banks)
        return []
      }
      return banks
    })
    const loading = computed(() => examStore.loading)
    
    // 显示所有题库，不限制总数，过滤掉undefined对象
    const displayedBanks = computed(() => {
      const banks = questionBanks.value || []
      return banks.filter(bank => bank && bank.id)
    })

    const jsonExample = `{
  "questions": [
    {
      "question": "题目内容",
      "options": ["选项A", "选项B", "选项C", "选项D"],
      "answer": 0,
      "explanation": "答案解析（可选）"
    }
  ]
}`

    const handleFileChange = (file) => {
      selectedFile.value = file
    }

    const uploadQuestionBank = async () => {
      if (!uploadForm.name.trim()) {
        ElMessage.error('请输入题库名称')
        return
      }

      if (!selectedFile.value) {
        ElMessage.error('请选择题库文件')
        return
      }

      try {
        const fileExt = selectedFile.value.name.split('.').pop().toLowerCase()
        
        if (fileExt === 'json') {
          // 处理JSON文件
          const fileContent = await readFileContent(selectedFile.value.raw)
          const questionData = JSON.parse(fileContent)
          
          if (!questionData.questions || !Array.isArray(questionData.questions)) {
            throw new Error('JSON格式不正确，缺少questions数组')
          }

          // 验证题目格式
          for (let i = 0; i < questionData.questions.length; i++) {
            const q = questionData.questions[i]
            if (!q.question || !q.options || !Array.isArray(q.options) || typeof q.answer !== 'number') {
              throw new Error(`第${i + 1}题格式不正确`)
            }
          }

          await examStore.addQuestionBank({
            name: uploadForm.name,
            description: uploadForm.description,
            questions: questionData.questions
          })
        } else {
          // 处理Excel/CSV文件 - 先创建题库，然后上传文件
          const newBank = await examStore.addQuestionBank({
            name: uploadForm.name,
            description: uploadForm.description,
            questions: [] // 先创建空题库
          })
          
          // 然后上传文件到这个题库
          const formData = new FormData()
          formData.append('file', selectedFile.value.raw)
          
          await examStore.uploadQuestionBankFile(newBank.id, formData)
        }

        showUploadDialog.value = false
        resetUploadForm()
        ElMessage.success('题库上传成功！')
      } catch (error) {
        ElMessage.error('文件上传失败: ' + error.message)
      }
    }

    const readFileContent = (file) => {
      return new Promise((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = (e) => resolve(e.target.result)
        reader.onerror = reject
        reader.readAsText(file)
      })
    }

    const resetUploadForm = () => {
      uploadForm.name = ''
      uploadForm.description = ''
      selectedFile.value = null
    }

    const resetCreateForm = () => {
      createForm.name = ''
      createForm.description = ''
      createForm.questions = [
        {
          question: '',
          options: ['', ''],
          answer: 0,
          explanation: ''
        }
      ]
    }

    const addQuestion = () => {
      createForm.questions.push({
        question: '',
        options: ['', ''],
        answer: 0,
        explanation: ''
      })
    }

    const removeQuestion = (index) => {
      createForm.questions.splice(index, 1)
    }

    const addOption = (questionIndex) => {
      createForm.questions[questionIndex].options.push('')
    }

    const removeOption = (questionIndex, optionIndex) => {
      const question = createForm.questions[questionIndex]
      question.options.splice(optionIndex, 1)
      // 如果删除的选项是正确答案，重置为第一个选项
      if (question.answer >= question.options.length) {
        question.answer = 0
      }
    }

    const createQuestionBank = async () => {
      if (!createForm.name.trim()) {
        ElMessage.error('请输入题库名称')
        return
      }

      // 验证题目
      for (let i = 0; i < createForm.questions.length; i++) {
        const q = createForm.questions[i]
        if (!q.question.trim()) {
          ElMessage.error(`第${i + 1}题的题目不能为空`)
          return
        }
        if (q.options.some(opt => !opt.trim())) {
          ElMessage.error(`第${i + 1}题的选项不能为空`)
          return
        }
      }

      try {
        await examStore.addQuestionBank({
          name: createForm.name,
          description: createForm.description,
          questions: createForm.questions
        })

        showCreateDialog.value = false
        resetCreateForm()
        ElMessage.success('题库创建成功！')
      } catch (error) {
        ElMessage.error('创建题库失败: ' + error.message)
      }
    }

    const startPractice = (bankId) => {
      console.log('startPractice 被调用，题库ID:', bankId)
      console.log('即将跳转到路由:', `/exam/${bankId}`)
      router.push(`/exam/${bankId}`)
    }

    const handleCommand = async (command) => {
      console.log('handleCommand 接收到命令:', command)
      const [action, ...bankIdParts] = command.split('-')
      const bankId = bankIdParts.join('-')
      console.log('解析结果 - action:', action, 'bankId:', bankId)
      
      if (action === 'practice') {
        console.log('执行开始刷题，题库ID:', bankId)
        startPractice(bankId)
      } else if (action === 'manage') {
        console.log('执行管理题目，题库ID:', bankId)
        manageQuestions(bankId)
      } else if (action === 'delete') {
        try {
          await ElMessageBox.confirm('确定要删除这个题库吗？', '确认删除', {
            type: 'warning',
            confirmButtonText: '确定删除',
            cancelButtonText: '取消'
          })
          
          console.log('开始删除题库:', bankId)
          await examStore.deleteQuestionBank(bankId)
          console.log('题库删除成功')
        } catch (error) {
          if (error !== 'cancel') {
            console.error('删除题库失败:', error)
            ElMessage.error('删除题库失败: ' + (error.message || error))
          }
        }
      }
    }

    // 管理题目
    const manageQuestions = async (bankId) => {
      try {
        const bank = questionBanks.value.find(b => b && b.id === bankId)
        if (!bank || !bank.id) {
          ElMessage.error('题库不存在或数据不完整')
          return
        }
        
        currentBank.value = bank
        // 获取题库题目
        const questions = await examStore.getQuestions(bankId)
        bankQuestions.value = Array.isArray(questions) ? questions : []
        showManageDialog.value = true
      } catch (error) {
        console.error('获取题目失败:', error)
        ElMessage.error('获取题目失败: ' + (error.message || error))
      }
    }

    // 添加新题目
    const addNewQuestion = () => {
      currentQuestion.value = {
        id: '',
        question: '',
        options: ['', '', '', ''],
        answer: 0,
        explanation: ''
      }
      showQuestionEditDialog.value = true
    }

    // 编辑题目
    const editQuestion = (question) => {
      currentQuestion.value = {
        id: question.id,
        question: question.question,
        options: [...question.options],
        answer: question.answer,
        explanation: question.explanation || ''
      }
      showQuestionEditDialog.value = true
    }

    // 保存题目
    const saveQuestion = async () => {
      if (!currentBank.value || !currentBank.value.id) {
        ElMessage.error('当前题库信息不完整')
        return
      }
      
      try {
        const questionData = {
          bank_id: currentBank.value.id,
          question: currentQuestion.value.question,
          options: currentQuestion.value.options,
          answer: currentQuestion.value.answer,
          explanation: currentQuestion.value.explanation
        }
        
        if (currentQuestion.value.id) {
          // 更新题目
          await examStore.updateQuestion(currentQuestion.value.id, questionData)
          ElMessage.success('题目更新成功')
        } else {
          // 添加新题目
          await examStore.addQuestion(questionData)
          ElMessage.success('题目添加成功')
        }
        
        showQuestionEditDialog.value = false
        // 刷新题目列表
        await manageQuestions(currentBank.value.id)
      } catch (error) {
        console.error('保存题目失败:', error)
        ElMessage.error('保存题目失败: ' + (error.message || error))
      }
    }

    // 删除题目
    const deleteQuestion = async (question) => {
      if (!currentBank.value || !currentBank.value.id) {
        ElMessage.error('当前题库信息不完整')
        return
      }
      
      try {
        await ElMessageBox.confirm(
          '确定要删除这个题目吗？此操作不可恢复。',
          '确认删除'
        )
        
        await examStore.deleteQuestion(question.id)
        ElMessage.success('题目删除成功')
        // 刷新题目列表
        await manageQuestions(currentBank.value.id)
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除题目失败')
        }
      }
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleDateString('zh-CN')
    }

    // 页面加载时获取数据
    onMounted(async () => {
      try {
        await examStore.loadQuestionBanks()
      } catch (error) {
        console.error('加载题库失败:', error)
        ElMessage.error('加载题库失败，请刷新页面重试')
      }
    })

    return {
      showUploadDialog,
      showCreateDialog,
      showManageDialog,
      selectedFile,
      uploadForm,
      createForm,
      questionBanks,
      displayedBanks,
      loading,
      jsonExample,
      handleFileChange,
      uploadQuestionBank,
      resetUploadForm,
      resetCreateForm,
      addQuestion,
      removeQuestion,
      addOption,
      removeOption,
      createQuestionBank,
      startPractice,
      handleCommand,
      formatDate,
      // 题目管理相关变量
      currentBank,
      bankQuestions,
      showQuestionEditDialog,
      currentQuestion,
      manageQuestions,
      addNewQuestion,
      editQuestion,
      saveQuestion,
      deleteQuestion
    }
  }
}
</script>

<style scoped>
.library {
  min-height: calc(100vh - 120px);
}

.library-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 15px;
  backdrop-filter: blur(10px);
}

.library-header h1 {
  color: white;
  font-size: 28px;
  margin: 0;
}

.library-content {
  min-height: 400px;
}

.banks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 24px;
  width: 100%;
  max-width: 100%;
}

/* 限制每行最多5个 */
@media (min-width: 1400px) {
  .banks-grid {
    grid-template-columns: repeat(5, 1fr);
  }
}

@media (max-width: 768px) {
  .banks-grid {
    grid-template-columns: 1fr;
  }
}

.bank-card-wrapper {
  background: white;
  margin-bottom: 24px;
  border-radius: 16px;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
  height: 240px;
  overflow: hidden;
}

.bank-card-wrapper:hover {
  transform: translateY(-5px);
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.15);
}

.bank-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 20px 10px;
  border-bottom: 1px solid #f0f0f0;
}

.bank-card-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.more-btn {
  color: #666;
  font-size: 18px;
  padding: 4px;
}

.more-btn:hover {
  color: #409eff;
}

.bank-card-body {
  padding: 15px 20px;
  flex: 1;
}

.bank-description {
  color: #666;
  font-size: 14px;
  line-height: 1.5;
  margin: 0 0 15px 0;
  height: 42px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.bank-stats {
  display: flex;
  gap: 15px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #888;
  font-size: 13px;
}

.stat-item .el-icon {
  font-size: 14px;
}

.bank-card-footer {
  padding: 15px 20px 20px;
  border-top: 1px solid #f0f0f0;
}

.practice-btn {
  width: 100%;
  border-radius: 8px;
  font-weight: 500;
}

.loading-container {
  padding: 20px;
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 300px;
}

.json-example {
  background: #f5f5f5;
  padding: 15px;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
}

.danger-item {
  color: #f56c6c;
}

.danger-item:hover {
  background-color: #fef0f0;
}

.header-buttons {
  display: flex;
  gap: 12px;
}

.format-example {
  font-size: 14px;
}

.format-table {
  width: 100%;
  border-collapse: collapse;
  margin: 10px 0;
}

.format-table th,
.format-table td {
  border: 1px solid #ddd;
  padding: 8px;
  text-align: left;
}

.format-table th {
  background-color: #f5f5f5;
  font-weight: 600;
}

.format-note {
  background-color: #f0f9ff;
  padding: 10px;
  border-radius: 6px;
  border-left: 4px solid #409eff;
  margin-top: 10px;
}

.questions-container {
  max-height: 400px;
  overflow-y: auto;
}

.question-item {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  background-color: #fafafa;
}

.question-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 600;
  color: #333;
}

.question-input {
  margin-bottom: 12px;
}

.options-container {
  margin-bottom: 12px;
}

.option-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.option-label {
  font-weight: 600;
  min-width: 20px;
  color: #666;
}

.answer-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  font-weight: 600;
  color: #333;
}

.explanation-input {
  margin-top: 8px;
}

.add-question-btn {
  width: 100%;
  margin-top: 16px;
  border-style: dashed;
}
</style>
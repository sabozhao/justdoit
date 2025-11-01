<template>
  <div class="admin-container">
    <div class="admin-header">
      <h1>管理员控制台</h1>
      <p>系统管理和数据分析</p>
    </div>

    <div class="admin-stats">
      <div class="stat-card">
        <h3>用户统计</h3>
        <div class="stat-number">{{ userStats.totalUsers }}</div>
        <p>总用户数</p>
      </div>
      <div class="stat-card">
        <h3>题库统计</h3>
        <div class="stat-number">{{ userStats.totalBanks }}</div>
        <p>总题库数</p>
      </div>
      <div class="stat-card">
        <h3>题目统计</h3>
        <div class="stat-number">{{ userStats.totalQuestions }}</div>
        <p>总题目数</p>
      </div>
    </div>

    <div class="admin-tabs">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="用户管理" name="users">
          <div class="user-management">
            <el-table :data="users" style="width: 100%">
              <el-table-column prop="username" label="用户名"></el-table-column>
              <el-table-column prop="email" label="邮箱"></el-table-column>
              <el-table-column prop="created_at" label="注册时间"></el-table-column>
              <el-table-column label="管理员">
                <template #default="scope">
                  <el-tag v-if="scope.row.is_admin" type="success">是</el-tag>
                  <el-tag v-else type="info">否</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作">
                <template #default="scope">
                  <el-button size="small" @click="toggleAdmin(scope.row)">
                    {{ scope.row.is_admin ? '取消管理员' : '设为管理员' }}
                  </el-button>
                  <el-button size="small" type="danger" @click="deleteUser(scope.row)">
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="题库管理" name="banks">
          <div class="bank-management">
            <el-table :data="questionBanks" style="width: 100%">
              <el-table-column prop="name" label="题库名称"></el-table-column>
              <el-table-column prop="description" label="描述"></el-table-column>
              <el-table-column prop="question_count" label="题目数量"></el-table-column>
              <el-table-column label="创建者">
                <template #default="scope">
                  <span>{{ getUserName(scope.row.user_id) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="280">
                <template #default="scope">
                  <el-button size="small" @click="viewBankQuestions(scope.row)">查看题目</el-button>
                  <el-button size="small" @click="editBank(scope.row)">编辑题库</el-button>
                  <el-button size="small" type="primary" @click="addQuestionToBank(scope.row)">添加题目</el-button>
                  <el-button size="small" type="danger" @click="deleteBank(scope.row)">
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="系统设置" name="settings">
          <div class="system-settings">
            <el-form label-width="120px">
              <el-form-item label="平台名称">
                <el-input v-model="settings.platformName"></el-input>
              </el-form-item>
              <el-form-item label="最大用户数">
                <el-input-number v-model="settings.maxUsers" :min="0"></el-input-number>
              </el-form-item>
              <el-form-item label="启用注册">
                <el-switch v-model="settings.allowRegistration"></el-switch>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="saveSettings">保存设置</el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 题库编辑对话框 -->
    <el-dialog v-model="bankDialogVisible" title="编辑题库" width="500px">
      <el-form :model="bankForm" label-width="80px">
        <el-form-item label="题库名称">
          <el-input v-model="bankForm.name"></el-input>
        </el-form-item>
        <el-form-item label="题库描述">
          <el-input v-model="bankForm.description" type="textarea" :rows="3"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="bankDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveBankInfo">保存</el-button>
      </template>
    </el-dialog>

    <!-- 题目管理对话框 -->
    <el-dialog v-model="questionListDialogVisible" :title="currentBank ? `${currentBank.name} - 题目管理` : '题目管理'" width="800px" :before-close="handleClose">
      <div v-if="currentBank">
        <div style="margin-bottom: 20px;">
          <h3>题库: {{ currentBank.name }}</h3>
          <p>{{ currentBank.description }}</p>
        </div>
        
        <div style="margin-bottom: 20px;">
          <el-button type="primary" @click="addQuestionToBank(currentBank)">
            添加题目
          </el-button>
        </div>
        
        <el-table :data="bankQuestions" style="width: 100%">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="question" label="题目内容" min-width="200" />
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
    <el-dialog v-model="questionEditDialogVisible" :title="currentQuestion.id ? '编辑题目' : '添加题目'" width="600px" v-if="currentQuestion">
      <el-form :model="currentQuestion" label-width="80px">
        <el-form-item label="题目内容">
          <el-input v-model="currentQuestion.question" type="textarea" :rows="3"></el-input>
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
          <el-input v-model="currentQuestion.explanation" type="textarea" :rows="2"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="questionEditDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveQuestion">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import { adminAPI } from '../api'

const authStore = useAuthStore()
const activeTab = ref('users')
const users = ref([])
const questionBanks = ref([])
const userStats = ref({
  totalUsers: 0,
  totalBanks: 0,
  totalQuestions: 0
})

const settings = ref({
  platformName: '智能刷题平台',
  maxUsers: 1000,
  allowRegistration: true
})

// 题库编辑相关状态
const currentBank = ref(null)
const bankQuestions = ref([])
const questionListDialogVisible = ref(false)
const questionEditDialogVisible = ref(false)
const bankDialogVisible = ref(false)
const currentQuestion = ref({
  id: '',
  question: '',
  options: ['', '', '', ''],
  answer: 0,
  explanation: ''
})
const bankForm = ref({
  name: '',
  description: ''
})

// 检查管理员权限
const checkAdminPermission = () => {
  if (!authStore.user?.is_admin) {
    ElMessage.error('您没有管理员权限')
    return false
  }
  return true
}

// 加载用户数据
const loadUsers = async () => {
  try {
    const response = await adminAPI.getUsers()
    console.log('Users API response:', response)
    users.value = response || []
    console.log('Users data updated:', users.value)
  } catch (error) {
    console.error('Users API error:', error)
    ElMessage.error('加载用户数据失败')
  }
}

// 加载题库数据
const loadQuestionBanks = async () => {
  try {
    const response = await adminAPI.getQuestionBanks()
    console.log('Question banks API response:', response)
    questionBanks.value = response || []
    console.log('Question banks data updated:', questionBanks.value)
  } catch (error) {
    console.error('Question banks API error:', error)
    ElMessage.error('加载题库数据失败')
  }
}

// 加载统计数据
const loadStats = async () => {
  try {
    const response = await adminAPI.getStats()
    console.log('Stats API response:', response)
    if (response) {
      userStats.value = {
        totalUsers: response.total_users,
        totalBanks: response.total_question_banks,
        totalQuestions: response.total_questions
      }
      console.log('User stats updated:', userStats.value)
    }
  } catch (error) {
    console.error('Stats API error:', error)
    ElMessage.error('加载统计数据失败')
  }
}

// 切换管理员权限
const toggleAdmin = async (user) => {
  if (!checkAdminPermission()) return
  
  try {
    await ElMessageBox.confirm(
      `确定要${user.is_admin ? '取消' : '设置'} ${user.username} 的管理员权限吗？`,
      '确认操作'
    )
    
    await adminAPI.updateUser(user.id, { is_admin: !user.is_admin })
    ElMessage.success('操作成功')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

// 删除用户
const deleteUser = async (user) => {
  if (!checkAdminPermission()) return
  
  try {
    await ElMessageBox.confirm(
      `确定要删除用户 ${user.username} 吗？此操作不可恢复。`,
      '确认删除'
    )
    
    await adminAPI.deleteUser(user.id)
    ElMessage.success('用户删除成功')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 删除题库
const deleteBank = async (bank) => {
  if (!checkAdminPermission()) return
  
  try {
    await ElMessageBox.confirm(
      `确定要删除题库 ${bank.name} 吗？此操作不可恢复。`,
      '确认删除'
    )
    
    await adminAPI.deleteQuestionBank(bank.id)
    ElMessage.success('题库删除成功')
    loadQuestionBanks()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 根据用户ID获取用户名
const getUserName = (userId) => {
  const user = users.value.find(u => u.id === userId)
  return user ? user.username : userId
}

// 查看题库题目
const viewBankQuestions = async (bank) => {
  currentBank.value = bank
  try {
    const response = await adminAPI.getQuestions(bank.id)
    bankQuestions.value = response || []
    questionListDialogVisible.value = true
  } catch (error) {
    console.error('获取题目失败:', error)
    ElMessage.error('获取题目失败')
  }
}

// 编辑题库信息
const editBank = (bank) => {
  currentBank.value = bank
  bankForm.value = {
    name: bank.name,
    description: bank.description
  }
  bankDialogVisible.value = true
}

// 保存题库信息
const saveBankInfo = async () => {
  if (!currentBank.value) return
  
  try {
    await adminAPI.updateQuestionBank(currentBank.value.id, bankForm.value)
    ElMessage.success('题库信息更新成功')
    bankDialogVisible.value = false
    loadQuestionBanks()
  } catch (error) {
    console.error('更新题库失败:', error)
    ElMessage.error('更新题库失败')
  }
}

// 添加题目到题库
const addQuestionToBank = (bank) => {
  currentBank.value = bank
  currentQuestion.value = {
    id: '',
    question: '',
    options: ['', '', '', ''],
    answer: 0,
    explanation: ''
  }
  questionEditDialogVisible.value = true
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
  questionEditDialogVisible.value = true
}

// 保存题目
const saveQuestion = async () => {
  if (!currentBank.value) return
  
  try {
    const questionData = {
      bank_id: currentBank.value.id,
      question: currentQuestion.value.question,
      options: currentQuestion.value.options,
      answer: currentQuestion.value.answer,
      explanation: currentQuestion.value.explanation
    }
    
    if (currentQuestion.value.id) {
      await adminAPI.updateQuestion(currentQuestion.value.id, questionData)
      ElMessage.success('题目更新成功')
    } else {
      await adminAPI.createQuestion(questionData)
      ElMessage.success('题目添加成功')
    }
    
    questionEditDialogVisible.value = false
    viewBankQuestions(currentBank.value) // 刷新题目列表
  } catch (error) {
    console.error('保存题目失败:', error)
    ElMessage.error('保存题目失败')
  }
}

// 删除题目
const deleteQuestion = async (question) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除这个题目吗？此操作不可恢复。',
      '确认删除'
    )
    
    await adminAPI.deleteQuestion(question.id)
    ElMessage.success('题目删除成功')
    viewBankQuestions(currentBank.value) // 刷新题目列表
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除题目失败')
    }
  }
}

// 处理对话框关闭
const handleClose = () => {
  questionListDialogVisible.value = false
  questionEditDialogVisible.value = false
  bankDialogVisible.value = false
}

// 保存设置
const saveSettings = async () => {
  try {
    await adminAPI.updateSettings(settings.value)
    ElMessage.success('设置保存成功')
  } catch (error) {
    ElMessage.error('保存设置失败')
  }
}

onMounted(async () => {
  // 等待用户信息加载完成
  if (!authStore.user) {
    // 如果用户信息为空，等待一下再检查
    setTimeout(() => {
      if (authStore.user?.is_admin) {
        Promise.all([loadUsers(), loadQuestionBanks(), loadStats()])
      } else {
        ElMessage.error('您没有访问此页面的权限')
      }
    }, 500)
  } else if (authStore.user.is_admin) {
    await Promise.all([loadUsers(), loadQuestionBanks(), loadStats()])
  } else {
    ElMessage.error('您没有访问此页面的权限')
  }
})
</script>

<style scoped>
.admin-container {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.admin-header {
  text-align: center;
  margin-bottom: 30px;
}

.admin-header h1 {
  color: #409eff;
  margin-bottom: 10px;
}

.admin-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.stat-card {
  background: #f5f7fa;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
}

.stat-card h3 {
  margin: 0 0 10px 0;
  color: #606266;
}

.stat-number {
  font-size: 2em;
  font-weight: bold;
  color: #409eff;
  margin-bottom: 5px;
}

.admin-tabs {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}
</style>
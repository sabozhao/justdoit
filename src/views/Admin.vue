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
            <el-table :data="users" style="width: 100%" :default-sort="{prop: 'created_at', order: 'descending'}">
              <el-table-column prop="username" label="用户名" width="150" min-width="120"></el-table-column>
              <el-table-column prop="email" label="邮箱" width="200" min-width="180" show-overflow-tooltip></el-table-column>
              <el-table-column prop="created_at" label="注册时间" width="180" min-width="160">
                <template #default="scope">
                  {{ formatDateTime(scope.row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column label="管理员" width="100" align="center">
                <template #default="scope">
                  <el-tag v-if="scope.row.is_admin" type="success">是</el-tag>
                  <el-tag v-else type="info">否</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="200" fixed="right">
                <template #default="scope">
                  <div class="action-buttons">
                    <el-button size="small" @click="toggleAdmin(scope.row)">
                      {{ scope.row.is_admin ? '取消管理员' : '设为管理员' }}
                    </el-button>
                    <el-button size="small" type="danger" @click="deleteUser(scope.row)">
                      删除
                    </el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="题库管理" name="banks">
          <div class="bank-management">
            <el-table :data="questionBanks" style="width: 100%" :default-sort="{prop: 'question_count', order: 'descending'}">
              <el-table-column prop="name" label="题库名称" width="180" min-width="150" show-overflow-tooltip></el-table-column>
              <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip></el-table-column>
              <el-table-column prop="question_count" label="题目数量" width="120" align="center" sortable></el-table-column>
              <el-table-column label="创建者" width="120" align="center">
                <template #default="scope">
                  <span>{{ getUserName(scope.row.user_id) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="380" min-width="380" fixed="right">
                <template #default="scope">
                  <div class="action-buttons">
                    <el-button size="small" @click="viewBankQuestions(scope.row)">查看题目</el-button>
                    <el-button size="small" @click="editBank(scope.row)">编辑题库</el-button>
                    <el-button size="small" type="primary" @click="addQuestionToBank(scope.row)">添加题目</el-button>
                    <el-button size="small" type="danger" @click="deleteBank(scope.row)">删除</el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="系统设置" name="settings">
          <div class="system-settings">
            <el-card class="settings-card">
              <template #header>
                <div class="card-header">
                  <span>腾讯云AI配置</span>
                  <el-button type="primary" size="small" @click="saveSettings">保存配置</el-button>
                </div>
              </template>
              
              <el-form label-width="140px" :model="aiSettings">
                <el-form-item label="SecretId" required>
                  <el-input 
                    v-model="aiSettings.tencent_secret_id" 
                    placeholder="请输入腾讯云SecretId"
                    show-password
                    type="password"
                    :disabled="isLoadingSettings"
                  />
                  <div class="form-tip">腾讯云API密钥ID，可在<a href="https://console.cloud.tencent.com/cam/capi" target="_blank">腾讯云控制台</a>获取</div>
                </el-form-item>
                
                <el-form-item label="SecretKey" required>
                  <el-input 
                    v-model="aiSettings.tencent_secret_key" 
                    placeholder="请输入腾讯云SecretKey（如果已配置则显示为***）"
                    show-password
                    type="password"
                    :disabled="isLoadingSettings"
                  />
                  <div class="form-tip">腾讯云API密钥，安全敏感信息</div>
                </el-form-item>
                
                <el-form-item label="区域" required>
                  <el-select v-model="aiSettings.tencent_region" placeholder="请选择区域" :disabled="isLoadingSettings">
                    <el-option label="北京 (ap-beijing)" value="ap-beijing" />
                    <el-option label="广州 (ap-guangzhou)" value="ap-guangzhou" />
                    <el-option label="上海 (ap-shanghai)" value="ap-shanghai" />
                    <el-option label="成都 (ap-chengdu)" value="ap-chengdu" />
                  </el-select>
                  <div class="form-tip">腾讯云服务区域</div>
                </el-form-item>
                
                <el-form-item label="模型名称" required>
                  <el-select v-model="aiSettings.tencent_model" placeholder="请选择模型" :disabled="isLoadingSettings">
                    <el-option label="混元精简版 (hunyuan-lite) - 免费" value="hunyuan-lite" />
                    <el-option label="混元专业版 (hunyuan-pro) - 付费" value="hunyuan-pro" />
                    <el-option label="混元标准版 (hunyuan-standard) - 付费" value="hunyuan-standard" />
                  </el-select>
                  <div class="form-tip">AI模型类型，推荐使用免费的hunyuan-lite</div>
                </el-form-item>
                
                <el-form-item label="API端点">
                  <el-input 
                    v-model="aiSettings.tencent_endpoint" 
                    placeholder="hunyuan.tencentcloudapi.com"
                    :disabled="isLoadingSettings"
                  />
                  <div class="form-tip">API端点地址，一般无需修改</div>
                </el-form-item>
              </el-form>
            </el-card>
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
          <el-table-column label="类型" width="100">
            <template #default="scope">
              <el-tag v-if="scope.row.type === 'judgment'" type="info" size="small">判断题</el-tag>
              <el-tag v-else-if="scope.row.is_multiple" type="warning" size="small">多选</el-tag>
              <el-tag v-else type="primary" size="small">单选</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="question" label="题目内容" min-width="200" />
          <el-table-column label="选项" min-width="300">
            <template #default="scope">
              <div v-for="(option, index) in scope.row.options" :key="index">
                {{ String.fromCharCode(65 + index) }}. {{ option }}
              </div>
            </template>
          </el-table-column>
          <el-table-column label="正确答案" width="150">
            <template #default="scope">
              <div v-if="scope.row.type === 'judgment'">
                <el-tag type="success" size="small">
                  {{ scope.row.answer && scope.row.answer[0] === 1 ? '正确' : '错误' }}
                </el-tag>
              </div>
              <div v-else-if="scope.row.is_multiple && Array.isArray(scope.row.answer)">
                <el-tag 
                  v-for="(ansIdx, idx) in scope.row.answer" 
                  :key="idx" 
                  type="success" 
                  size="small"
                  style="margin-right: 5px;"
                >
                  {{ String.fromCharCode(65 + ansIdx) }}
                </el-tag>
              </div>
              <span v-else>
                {{ String.fromCharCode(65 + (Array.isArray(scope.row.answer) ? scope.row.answer[0] : scope.row.answer)) }}
              </span>
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
    <el-dialog v-model="questionEditDialogVisible" :title="currentQuestion.id ? '编辑题目' : '添加题目'" width="700px" v-if="currentQuestion">
      <el-form :model="currentQuestion" label-width="100px">
        <el-form-item label="题目内容">
          <el-input v-model="currentQuestion.question" type="textarea" :rows="3"></el-input>
        </el-form-item>
        <el-form-item label="题目类型">
          <el-radio-group v-model="currentQuestion.type" @change="handleQuestionTypeChange">
            <el-radio label="choice">选择题</el-radio>
            <el-radio label="judgment">判断题</el-radio>
          </el-radio-group>
          <div v-if="currentQuestion.type === 'choice'" style="margin-top: 8px;">
            <el-radio-group v-model="currentQuestion.is_multiple" size="small" @change="handleQuestionTypeChange">
              <el-radio :label="false">单选题</el-radio>
              <el-radio :label="true">多选题</el-radio>
            </el-radio-group>
          </div>
        </el-form-item>
        <el-form-item label="选项" v-if="currentQuestion.type === 'choice'">
          <div v-for="(option, index) in currentQuestion.options" :key="index" style="margin-bottom: 10px;">
            <el-input 
              v-model="currentQuestion.options[index]" 
              :placeholder="`选项 ${String.fromCharCode(65 + index)}`"
              style="margin-bottom: 8px;"
            >
              <template #prepend>{{ String.fromCharCode(65 + index) }}</template>
              <template #append>
                <el-button 
                  v-if="currentQuestion.options.length > 2 && index >= 2"
                  type="danger" 
                  size="small"
                  @click="removeOption(index)"
                  :icon="Delete"
                >
                  删除
                </el-button>
              </template>
            </el-input>
          </div>
          <el-button 
            v-if="currentQuestion.options.length < 10"
            type="primary" 
            plain
            size="small"
            @click="addOption"
            style="width: 100%;"
          >
            <el-icon><Plus /></el-icon>
            添加选项（最多10个）
          </el-button>
          <div v-else style="color: #909399; font-size: 12px; text-align: center; margin-top: 8px;">
            已达到最大选项数（10个）
          </div>
        </el-form-item>
        <el-form-item v-if="currentQuestion.type === 'judgment'" label="正确答案">
          <el-radio-group v-model="currentQuestion.answer">
            <el-radio :label="0">错误</el-radio>
            <el-radio :label="1">正确</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-else :label="currentQuestion.is_multiple ? '正确答案（多选）' : '正确答案'">
          <!-- 多选题：使用checkbox -->
          <template v-if="currentQuestion.is_multiple === true">
            <el-checkbox-group v-model="currentQuestion.answer" style="display: flex; flex-direction: column; gap: 8px;">
              <el-checkbox
                v-for="(option, index) in currentQuestion.options"
                :key="'multi-' + index"
                :label="index"
                :disabled="!option.trim()"
              >
                {{ String.fromCharCode(65 + index) }}. {{ option || `选项 ${String.fromCharCode(65 + index)}` }}
              </el-checkbox>
            </el-checkbox-group>
            <div v-if="!Array.isArray(currentQuestion.answer) || currentQuestion.answer.length === 0" style="color: #f56c6c; font-size: 12px; margin-top: 8px;">
              请至少选择一个正确答案
            </div>
          </template>
          <!-- 单选题：使用select -->
          <template v-else>
            <el-select v-model="currentQuestion.answer" placeholder="请选择正确答案" style="width: 100%;">
              <el-option v-for="(option, index) in currentQuestion.options" 
                         :key="'single-' + index" 
                         :label="`${String.fromCharCode(65 + index)}. ${option || '选项 ' + String.fromCharCode(65 + index)}`" 
                         :value="index"
                         :disabled="!option.trim()">
              </el-option>
            </el-select>
          </template>
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
import { ref, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
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
  platformName: '刷个题',
  maxUsers: 1000,
  allowRegistration: true
})

const aiSettings = ref({
  tencent_secret_id: '',
  tencent_secret_key: '',
  tencent_region: 'ap-beijing',
  tencent_model: 'hunyuan-lite',
  tencent_endpoint: 'hunyuan.tencentcloudapi.com'
})

// 保存初始设置值，用于比较哪些字段被修改了
const initialSettings = ref({
  tencent_secret_id: '',
  tencent_secret_key: '',
  tencent_region: 'ap-beijing',
  tencent_model: 'hunyuan-lite',
  tencent_endpoint: 'hunyuan.tencentcloudapi.com'
})

const isLoadingSettings = ref(false)

// 题库编辑相关状态
const currentBank = ref(null)
const bankQuestions = ref([])
const questionListDialogVisible = ref(false)
const questionEditDialogVisible = ref(false)
const bankDialogVisible = ref(false)
const currentQuestion = ref({
  id: '',
  question: '',
  options: ['', ''], // 初始只有2个选项
  answer: 0, // 单选题默认选择第一个选项（数字），多选题会变为数组，判断题：0=错误，1=正确
  is_multiple: false, // 是否为多选题，默认为单选题
  type: 'choice', // 题目类型：choice（选择题）或judgment（判断题）
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

// 格式化日期时间
const formatDateTime = (dateString) => {
  if (!dateString) return ''
  try {
    const date = new Date(dateString)
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    return `${year}-${month}-${day} ${hours}:${minutes}`
  } catch (e) {
    return dateString
  }
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
  // 重置题目为初始状态
  currentQuestion.value = {
    id: '',
    question: '',
    options: ['', ''], // 初始只有2个选项，可以添加更多
    answer: 0, // 单选题默认选择第一个选项（数字），判断题：0=错误，1=正确
    is_multiple: false, // 默认单选题
    type: 'choice', // 默认选择题
    explanation: ''
  }
  console.log('添加题目 - currentQuestion初始化:', currentQuestion.value)
  questionEditDialogVisible.value = true
}

// 编辑题目
const editQuestion = (question) => {
  // 处理答案：保持原有格式（多选题是数组，单选题是数字，判断题是0或1）
  let answer = question.answer
  const questionType = question.type || 'choice'
  
  if (questionType === 'judgment') {
    // 判断题：答案必须是0或1
    if (Array.isArray(answer)) {
      answer = answer.length > 0 ? answer[0] : 0
    } else if (answer === null || answer === undefined) {
      answer = 0
    }
    // 确保答案在0-1范围内
    answer = answer === 1 ? 1 : 0
  } else if (question.is_multiple) {
    // 多选题：确保是数组
    if (!Array.isArray(answer)) {
      answer = answer !== null && answer !== undefined ? [answer] : []
    }
  } else {
    // 单选题：确保是数字
    if (Array.isArray(answer)) {
      answer = answer.length > 0 ? answer[0] : 0
    } else if (answer === null || answer === undefined) {
      answer = 0
    }
  }
  
  currentQuestion.value = {
    id: question.id,
    question: question.question,
    options: questionType === 'judgment' ? ['错误', '正确'] : [...question.options],
    answer: answer,
    is_multiple: question.is_multiple || false,
    type: questionType,
    explanation: question.explanation || ''
  }
  questionEditDialogVisible.value = true
}

// 添加选项
const addOption = () => {
  if (currentQuestion.value.options.length < 10) {
    currentQuestion.value.options.push('')
    // 如果当前是单选题且没有选择答案，自动选择第一个
    if (!currentQuestion.value.is_multiple && 
        (currentQuestion.value.answer === null || currentQuestion.value.answer === undefined)) {
      currentQuestion.value.answer = 0
    }
  } else {
    ElMessage.warning('最多只能添加10个选项')
  }
}

// 删除选项
const removeOption = (index) => {
  if (currentQuestion.value.options.length > 2) {
    currentQuestion.value.options.splice(index, 1)
    
    // 调整答案索引
    if (currentQuestion.value.is_multiple) {
      // 多选题：从数组中移除该索引，并调整其他索引
      if (Array.isArray(currentQuestion.value.answer)) {
        currentQuestion.value.answer = currentQuestion.value.answer
          .filter(ans => ans !== index) // 移除被删除的选项
          .map(ans => ans > index ? ans - 1 : ans) // 调整大于被删除索引的选项
      }
      // 如果答案数组为空，至少保留第一个选项（如果存在）
      if (currentQuestion.value.answer.length === 0 && currentQuestion.value.options.length > 0) {
        // 不自动选择，让用户自己选择
      }
    } else {
      // 单选题：调整单个答案索引
      if (currentQuestion.value.answer === index) {
        // 如果删除的就是当前答案，选择第一个选项
        currentQuestion.value.answer = 0
      } else if (currentQuestion.value.answer > index) {
        // 如果答案索引大于被删除的索引，需要减1
        currentQuestion.value.answer = currentQuestion.value.answer - 1
      }
    }
  } else {
    ElMessage.warning('至少需要2个选项')
  }
}

// 处理题目类型变化
const handleQuestionTypeChange = () => {
  const questionType = currentQuestion.value.type
  
  if (questionType === 'judgment') {
    // 判断题：选项固定为["错误", "正确"]，答案默认为0（错误）
    currentQuestion.value.options = ['错误', '正确']
    currentQuestion.value.answer = 0
    currentQuestion.value.is_multiple = false
  } else {
    // 选择题：恢复选项编辑功能
    if (currentQuestion.value.options.length < 2) {
      currentQuestion.value.options = ['', '']
    }
    // 如果答案不在有效范围内，重置为0
    if (currentQuestion.value.is_multiple) {
      if (!Array.isArray(currentQuestion.value.answer)) {
        const currentAnswer = currentQuestion.value.answer
        if (currentAnswer !== null && currentAnswer !== undefined && 
            typeof currentAnswer === 'number' &&
            currentAnswer >= 0 && currentAnswer < currentQuestion.value.options.length) {
          currentQuestion.value.answer = [currentAnswer]
        } else {
          currentQuestion.value.answer = []
        }
      }
    } else {
      if (Array.isArray(currentQuestion.value.answer)) {
        currentQuestion.value.answer = currentQuestion.value.answer.length > 0 ? 
                                        currentQuestion.value.answer[0] : 0
      } else if (currentQuestion.value.answer === null || currentQuestion.value.answer === undefined) {
        currentQuestion.value.answer = 0
      } else if (currentQuestion.value.answer < 0 || currentQuestion.value.answer >= currentQuestion.value.options.length) {
        currentQuestion.value.answer = 0
      }
    }
  }
}

// 保存题目
const saveQuestion = async () => {
  if (!currentBank.value) return
  
  // 验证题目数据
  if (!currentQuestion.value.question.trim()) {
    ElMessage.warning('请输入题目内容')
    return
  }
  
  const questionType = currentQuestion.value.type || 'choice'
  let validOptions = []
  let finalAnswer = []
  
  if (questionType === 'judgment') {
    // 判断题：选项固定为["错误", "正确"]，答案：0=错误，1=正确
    validOptions = ['错误', '正确']
    const answer = currentQuestion.value.answer
    if (answer !== 0 && answer !== 1) {
      ElMessage.warning('判断题答案必须是0（错误）或1（正确）')
      return
    }
    finalAnswer = [answer]
  } else {
    // 选择题：验证选项数量（最多10个）
    if (currentQuestion.value.options.length > 10) {
      ElMessage.warning('选项数量不能超过10个')
      return
    }
    
    // 过滤空选项，但保留原始索引映射
    const indexMap = [] // 原始索引到新索引的映射
    
    for (let i = 0; i < currentQuestion.value.options.length; i++) {
      const opt = currentQuestion.value.options[i]
      if (opt.trim()) {
        indexMap[i] = validOptions.length // 原始索引i对应的新索引
        validOptions.push(opt.trim())
      }
    }
    
    if (validOptions.length < 2) {
      ElMessage.warning('至少需要2个有效选项')
      return
    }
    
    // 验证答案并转换索引（从原始索引转换为有效选项的索引）
    if (currentQuestion.value.is_multiple) {
      if (!Array.isArray(currentQuestion.value.answer) || currentQuestion.value.answer.length === 0) {
        ElMessage.warning('多选题请至少选择一个正确答案')
        return
      }
      // 转换多选题答案索引
      for (const originalIdx of currentQuestion.value.answer) {
        if (originalIdx < 0 || originalIdx >= indexMap.length || indexMap[originalIdx] === undefined) {
          ElMessage.warning('答案索引无效（可能对应空选项）')
          return
        }
        const newIdx = indexMap[originalIdx]
        if (newIdx < 0 || newIdx >= validOptions.length) {
          ElMessage.warning('答案索引超出选项范围')
          return
        }
        // 去重
        if (!finalAnswer.includes(newIdx)) {
          finalAnswer.push(newIdx)
        }
      }
    } else {
      // 单选题
      if (currentQuestion.value.answer === null || currentQuestion.value.answer === undefined) {
        ElMessage.warning('请选择正确答案')
        return
      }
      const originalIdx = currentQuestion.value.answer
      if (originalIdx < 0 || originalIdx >= indexMap.length || indexMap[originalIdx] === undefined) {
        ElMessage.warning('答案索引无效（可能对应空选项）')
        return
      }
      const newIdx = indexMap[originalIdx]
      if (newIdx < 0 || newIdx >= validOptions.length) {
        ElMessage.warning('答案索引超出选项范围')
        return
      }
      finalAnswer = [newIdx] // 转换为数组格式
    }
  }
  
  try {
    const questionData = {
      bank_id: currentBank.value.id,
      question: currentQuestion.value.question.trim(),
      options: validOptions, // 使用过滤后的有效选项
      answer: finalAnswer, // 已经是数组格式，索引已经转换为有效选项的索引
      type: questionType, // 题目类型：choice（选择题）或judgment（判断题）
      explanation: currentQuestion.value.explanation || ''
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
    ElMessage.error(error.response?.data?.error || '保存题目失败')
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

// 加载系统设置
const loadSettings = async () => {
  try {
    isLoadingSettings.value = true
    const response = await adminAPI.getSettings()
    if (response && response.settings) {
      // 更新AI配置
      if (response.settings.tencent_secret_id) {
        aiSettings.value.tencent_secret_id = response.settings.tencent_secret_id
      }
      if (response.settings.tencent_secret_key) {
        // 如果返回的是掩码格式（包含***），保持原样，不更新
        if (response.settings.tencent_secret_key.includes('***')) {
          aiSettings.value.tencent_secret_key = response.settings.tencent_secret_key
        } else {
          aiSettings.value.tencent_secret_key = response.settings.tencent_secret_key
        }
      }
      if (response.settings.tencent_region) {
        aiSettings.value.tencent_region = response.settings.tencent_region
      }
      if (response.settings.tencent_model) {
        aiSettings.value.tencent_model = response.settings.tencent_model
      }
      if (response.settings.tencent_endpoint) {
        aiSettings.value.tencent_endpoint = response.settings.tencent_endpoint
      }
      
      // 保存初始值，用于比较哪些字段被修改了
      initialSettings.value = {
        tencent_secret_id: aiSettings.value.tencent_secret_id || '',
        tencent_secret_key: aiSettings.value.tencent_secret_key || '',
        tencent_region: aiSettings.value.tencent_region || 'ap-beijing',
        tencent_model: aiSettings.value.tencent_model || 'hunyuan-lite',
        tencent_endpoint: aiSettings.value.tencent_endpoint || 'hunyuan.tencentcloudapi.com'
      }
    }
  } catch (error) {
    console.error('加载设置失败:', error)
  } finally {
    isLoadingSettings.value = false
  }
}

// 保存设置（只保存用户修改的字段，未修改的字段保持不变）
const saveSettings = async () => {
  try {
    isLoadingSettings.value = true
    
    // 构建要保存的设置对象，只包含用户实际修改的字段
    // 比较当前值与初始值（加载时的值），只有不同的字段才保存
    const settingsToSave = {}
    
    // SecretId：如果当前值有内容，且与初始值不同，则更新
    if (aiSettings.value.tencent_secret_id && aiSettings.value.tencent_secret_id.trim() !== '') {
      if (aiSettings.value.tencent_secret_id !== initialSettings.value.tencent_secret_id) {
        settingsToSave.tencent_secret_id = aiSettings.value.tencent_secret_id
      }
    }
    
    // SecretKey：如果用户输入了新密钥（不是掩码格式），则更新
    // 注意：即使初始值是掩码，只要用户输入了新密钥，就应该保存
    if (aiSettings.value.tencent_secret_key && !aiSettings.value.tencent_secret_key.includes('***')) {
      // 用户输入了新密钥（不是掩码格式），无论初始值是什么，都保存
      settingsToSave.tencent_secret_key = aiSettings.value.tencent_secret_key
    }
    // 如果当前值是掩码格式（***），说明用户没有修改密钥，不更新
    
    // Region：如果当前值与初始值不同，则更新
    if (aiSettings.value.tencent_region !== initialSettings.value.tencent_region) {
      if (aiSettings.value.tencent_region) {
        settingsToSave.tencent_region = aiSettings.value.tencent_region
      }
    }
    
    // Model：如果当前值与初始值不同，则更新
    if (aiSettings.value.tencent_model !== initialSettings.value.tencent_model) {
      if (aiSettings.value.tencent_model) {
        settingsToSave.tencent_model = aiSettings.value.tencent_model
      }
    }
    
    // Endpoint：如果当前值与初始值不同，则更新
    if (aiSettings.value.tencent_endpoint !== initialSettings.value.tencent_endpoint) {
      if (aiSettings.value.tencent_endpoint) {
        settingsToSave.tencent_endpoint = aiSettings.value.tencent_endpoint
      }
    }
    
    // 验证必填项
    // SecretId: 如果当前值有，就用当前值；否则用初始值；都没有就报错
    const finalSecretId = aiSettings.value.tencent_secret_id || initialSettings.value.tencent_secret_id
    
    // SecretKey: 
    // 1. 如果当前值不是掩码格式，说明用户输入了新密钥，使用当前值
    // 2. 如果当前值是掩码格式（***），说明用户没有修改密钥，应该检查：
    //    - 如果初始值不是掩码格式，说明数据库中已经有密钥，使用初始值（但初始值是掩码，无法使用）
    //    - 实际上：如果当前值是掩码，而初始值也是掩码，说明都没有填写
    //    - 如果当前值是掩码，但初始值不是掩码，说明用户没有修改，应该用初始值（但需要确保初始值不是掩码）
    // 但问题是：如果数据库中已经有密钥，返回给前端的是掩码格式，所以初始值也是掩码
    // 所以如果当前值是掩码，我们应该认为数据库中有密钥（只是被掩码了）
    let finalSecretKey = null
    if (aiSettings.value.tencent_secret_key && !aiSettings.value.tencent_secret_key.includes('***')) {
      // 当前值是有效密钥（不是掩码），说明用户输入了新密钥
      finalSecretKey = aiSettings.value.tencent_secret_key
    } else if (aiSettings.value.tencent_secret_key && aiSettings.value.tencent_secret_key.includes('***')) {
      // 当前值是掩码格式（***），说明用户没有修改密钥
      // 如果初始值也是掩码，说明数据库中没有密钥，需要用户填写
      // 如果初始值不是掩码（这种情况不应该发生，因为loadSettings时会保存掩码），但为了保险还是检查
      if (initialSettings.value.tencent_secret_key && !initialSettings.value.tencent_secret_key.includes('***')) {
        // 初始值不是掩码，使用初始值
        finalSecretKey = initialSettings.value.tencent_secret_key
      } else {
        // 初始值也是掩码或为空，说明数据库中没有密钥，需要用户填写
        // 但这里应该允许保存（因为掩码说明数据库中有密钥，只是被隐藏了）
        // 实际上，如果数据库中已经有密钥，返回的是掩码，我们应该认为有效
        // 所以这里设置为非null，表示数据库中已经有密钥
        finalSecretKey = '***MASKED***' // 标记为已配置（掩码）
      }
    } else if (initialSettings.value.tencent_secret_key && !initialSettings.value.tencent_secret_key.includes('***')) {
      // 当前值为空或未设置，但初始值有有效密钥
      finalSecretKey = initialSettings.value.tencent_secret_key
    } else if (initialSettings.value.tencent_secret_key && initialSettings.value.tencent_secret_key.includes('***')) {
      // 当前值为空，初始值也是掩码，说明数据库中已有密钥（被掩码了）
      finalSecretKey = '***MASKED***' // 标记为已配置（掩码）
    }
    
    console.log('验证必填项 - SecretId:', finalSecretId ? `已填写(${finalSecretId.substring(0, 3)}...)` : '未填写')
    console.log('验证必填项 - SecretKey:', finalSecretKey ? (finalSecretKey === '***MASKED***' ? '已配置（掩码）' : '已填写') : '未填写')
    console.log('当前SecretKey值:', aiSettings.value.tencent_secret_key)
    console.log('初始SecretKey值:', initialSettings.value.tencent_secret_key)
    
    // 检查SecretId和SecretKey是否有效
    // SecretKey如果是'***MASKED***'，说明数据库中已有密钥（被掩码），认为有效
    const hasValidSecretId = finalSecretId && finalSecretId.trim() !== ''
    // SecretKey有效的情况：
    // 1. 用户输入了新密钥（不是掩码，不是空）
    // 2. 数据库中已有密钥（返回掩码，标记为'***MASKED***'）
    const hasValidSecretKey = finalSecretKey && finalSecretKey !== '' && (
      finalSecretKey === '***MASKED***' || // 掩码表示数据库中已有密钥
      !finalSecretKey.includes('***') // 不是掩码，说明是用户输入的新密钥
    )
    
    if (!hasValidSecretId || !hasValidSecretKey) {
      console.log('验证失败 - SecretId:', hasValidSecretId, finalSecretId)
      console.log('验证失败 - SecretKey:', hasValidSecretKey, finalSecretKey)
      ElMessage.warning('请填写完整的SecretId和SecretKey')
      return
    }
    
    // 如果没有任何字段需要更新，提示用户
    if (Object.keys(settingsToSave).length === 0) {
      ElMessage.info('没有检测到任何修改')
      return
    }
    
    console.log('保存的设置（仅修改的字段）:', settingsToSave)
    await adminAPI.updateSettings(settingsToSave)
    ElMessage.success('配置保存成功，AI服务配置已更新')
    
    // 重新加载设置以更新初始值快照
    await loadSettings()
  } catch (error) {
    console.error('保存设置失败:', error)
    ElMessage.error('保存配置失败: ' + (error.message || error))
  } finally {
    isLoadingSettings.value = false
  }
}

// 监听is_multiple的变化，自动转换答案格式
watch(() => currentQuestion.value.is_multiple, (newValue, oldValue) => {
  // 只在值真正改变时处理（避免初始化时触发）
  if (oldValue !== undefined && newValue !== undefined && newValue !== oldValue) {
    console.log('🔔 watch检测到is_multiple变化:', oldValue, '->', newValue, '类型:', typeof newValue)
    // 延迟执行，确保radio-group的值已经更新
    setTimeout(() => {
      handleQuestionTypeChange(newValue)
    }, 10)
  }
}, { immediate: false })

onMounted(async () => {
  // 等待用户信息加载完成
  if (!authStore.user) {
    // 如果用户信息为空，等待一下再检查
    setTimeout(() => {
      if (authStore.user?.is_admin) {
        Promise.all([loadUsers(), loadQuestionBanks(), loadStats(), loadSettings()])
      } else {
        ElMessage.error('您没有访问此页面的权限')
      }
    }, 500)
  } else if (authStore.user.is_admin) {
    await Promise.all([loadUsers(), loadQuestionBanks(), loadStats(), loadSettings()])
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
  display: flex;
  flex-direction: column;
  min-height: 0;
  position: relative;
}

.admin-container > * {
  position: relative;
}

.system-settings {
  padding: 20px;
}

.settings-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 5px;
  line-height: 1.5;
}

.form-tip a {
  color: #409eff;
  text-decoration: none;
}

.form-tip a:hover {
  text-decoration: underline;
}

.admin-header {
  text-align: center;
  margin-bottom: 30px !important;
  flex-shrink: 0;
  height: auto;
  min-height: 60px;
  position: relative;
  z-index: 2;
}

.admin-header h1 {
  color: #409eff;
  margin-bottom: 10px;
}

.admin-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 30px !important;
  margin-top: 0 !important;
  height: 150px !important;
  min-height: 150px !important;
  max-height: 150px !important;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  contain: layout size style;
  position: relative;
  z-index: 1;
  clear: both;
}

.stat-card {
  background: #f5f7fa;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
  height: 110px !important;
  min-height: 110px !important;
  max-height: 110px !important;
  box-sizing: border-box;
  display: flex !important;
  flex-direction: column !important;
  justify-content: center;
  overflow: hidden;
  contain: layout size style;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  position: relative;
}

.stat-card h3 {
  margin: 0 0 10px 0;
  color: #606266;
  font-size: 14px;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
  height: 20px;
}

.stat-card p {
  margin: 0;
  color: #909399;
  font-size: 12px;
  line-height: 1.2;
  flex-shrink: 0;
  height: 16px;
}

.stat-number {
  font-size: 2em;
  font-weight: bold;
  color: #409eff;
  margin-bottom: 5px;
  line-height: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.admin-tabs {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  position: relative;
  z-index: 0;
  margin-top: 0 !important;
  clear: both;
}

/* 强制覆盖 Element Plus tabs 的默认样式，防止定位问题 */
.admin-tabs :deep(.el-tabs__header) {
  margin: 0 0 15px 0 !important;
  position: relative !important;
  z-index: 0 !important;
}

/* 统一tab标签宽度，避免切换时长度变化 */
.admin-tabs :deep(.el-tabs__nav) {
  display: flex;
  width: 100%;
}

.admin-tabs :deep(.el-tabs__item) {
  flex: 1 !important;
  text-align: center !important;
  width: auto !important;
  min-width: 0 !important;
  max-width: none !important;
  padding: 0 20px !important;
}

.admin-tabs :deep(.el-tabs__content) {
  position: relative !important;
  z-index: 0 !important;
  overflow: visible !important;
}

.admin-tabs :deep(.el-tab-pane) {
  position: relative !important;
  z-index: 0 !important;
}

/* 用户管理表格样式 */
.user-management {
  width: 100%;
}

.user-management :deep(.el-table) {
  width: 100%;
}

.user-management :deep(.el-table__body-wrapper) {
  overflow-x: auto;
  overflow-y: auto;
}

/* 确保表格可以横向滚动，不会截断操作列 */
.user-management :deep(.el-table__header-wrapper),
.user-management :deep(.el-table__body-wrapper) {
  min-width: 100%;
}

.user-management :deep(.el-table) {
  min-width: 750px; /* 确保所有列都能完整显示 */
}

.action-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: flex-start;
}

.action-buttons .el-button {
  flex-shrink: 0;
  white-space: nowrap;
}

/* 题库管理表格样式 */
.bank-management {
  width: 100%;
}

.bank-management :deep(.el-table) {
  width: 100%;
}

.bank-management :deep(.el-table__body-wrapper) {
  overflow-x: auto;
  overflow-y: auto;
}

/* 确保表格可以横向滚动，不会截断操作列 */
.bank-management :deep(.el-table__header-wrapper),
.bank-management :deep(.el-table__body-wrapper) {
  min-width: 100%;
}

.bank-management :deep(.el-table) {
  min-width: 900px; /* 确保所有列都能完整显示 */
}
</style>
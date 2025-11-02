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
    <el-dialog 
      v-model="showUploadDialog" 
      title="上传题库" 
      width="600px" 
      :close-on-click-modal="!isUploading" 
      :close-on-press-escape="!isUploading" 
      :show-close="!isUploading"
      @closed="resetUploadForm"
    >
      <el-form :model="uploadForm" label-width="80px">
        <el-form-item label="题库名称" required>
          <el-input v-model="uploadForm.name" placeholder="请输入题库名称" />
        </el-form-item>
        <el-form-item label="题库描述">
          <el-input v-model="uploadForm.description" type="textarea" placeholder="请输入题库描述（可选）" />
        </el-form-item>
        <el-form-item label="选择文件" required>
          <el-upload
            ref="uploadRef"
            :auto-upload="false"
            :show-file-list="true"
            :limit="1"
            accept=".xlsx,.xls,.csv,.pdf,.doc,.docx"
            :on-change="handleFileChange"
            :on-remove="handleFileRemove"
          >
            <el-button type="primary">选择文件</el-button>
            <template #tip>
              <div class="el-upload__tip">
                支持 Excel (.xlsx, .xls)、CSV、PDF 和 Word (.doc, .docx) 格式的题库文件<br/>
                <span style="color: #409eff; font-weight: bold;">✨ AI 智能识别：</span>上传 PDF 或 Word 文件后，系统会自动使用 AI 识别题目、选项和答案，轻松导入题库
              </div>
            </template>
          </el-upload>
        </el-form-item>
        
        <el-form-item label="解析方式" required>
          <el-radio-group v-model="uploadForm.parseMode">
            <el-radio label="format">固定格式解析</el-radio>
            <el-radio label="ai">AI 自动分析</el-radio>
          </el-radio-group>
          <div class="parse-mode-tip">
            <div v-if="uploadForm.parseMode === 'format'">
              <strong>固定格式解析：</strong>按照文件的标准格式解析（Excel/CSV 按列格式，Word 按文本格式）
            </div>
            <div v-else>
              <strong>AI 自动分析：</strong>使用 AI 智能识别文件中的题目（需要配置腾讯云 AI 服务）
            </div>
          </div>
        </el-form-item>
      </el-form>
      
      <el-collapse v-if="uploadForm.parseMode === 'format'">
        <el-collapse-item title="Excel格式示例" name="excel-example">
          <div class="format-example">
            <p><strong>Excel文件格式（列顺序）：</strong></p>
            <p style="color: #409eff; font-weight: bold; margin: 10px 0;">
              题目、正确答案、选项A、选项B、选项C、...、选项J、解析
            </p>
            <table class="format-table">
              <thead>
                <tr>
                  <th>题目</th>
                  <th>正确答案</th>
                  <th>选项A</th>
                  <th>选项B</th>
                  <th>选项C</th>
                  <th>选项D</th>
                  <th>...</th>
                  <th>选项J</th>
                  <th>解析</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>这是一道示例题目？</td>
                  <td>A</td>
                  <td>选项A内容</td>
                  <td>选项B内容</td>
                  <td>选项C内容</td>
                  <td>选项D内容</td>
                  <td>...</td>
                  <td></td>
                  <td>答案解析（可选）</td>
                </tr>
                <tr>
                  <td>多选题示例？</td>
                  <td>A,B,C</td>
                  <td>选项A</td>
                  <td>选项B</td>
                  <td>选项C</td>
                  <td>选项D</td>
                  <td>...</td>
                  <td></td>
                  <td>多选题解析</td>
                </tr>
                <tr>
                  <td>最多选项示例？</td>
                  <td>J</td>
                  <td>选项A</td>
                  <td>选项B</td>
                  <td>选项C</td>
                  <td>选项D</td>
                  <td>...</td>
                  <td>选项J</td>
                  <td>支持最多10个选项</td>
                </tr>
              </tbody>
            </table>
            <p class="format-note">
              <strong>格式说明：</strong><br>
              • <strong>列顺序固定：</strong>题目（第1列）、正确答案（第2列，固定位置）、选项A-J（第3-12列）、解析（最后一列）<br>
              • <strong>正确答案：</strong>可以填写 A/B/C/D/E/F/G/H/I/J 或 1/2/3/4/5/6/7/8/9/10，多选题使用逗号分隔（如 A,B,C）<br>
              • <strong>选项：</strong>至少需要选项A和B，最多支持10个选项（A-J）<br>
              • <strong>解析列：</strong>可选，放在最后<br>
              • <strong>列名：</strong>支持中英文列名，但列顺序需保持一致
            </p>
            <div class="demo-download">
              <el-button type="primary" size="small" @click="downloadDemo('excel')">
                <el-icon><Download /></el-icon>
                下载Excel示例文件
              </el-button>
            </div>
          </div>
        </el-collapse-item>
        
        <el-collapse-item title="CSV格式示例" name="csv-example">
          <div class="format-example">
            <p><strong>CSV文件格式（列顺序）：</strong></p>
            <p style="color: #409eff; font-weight: bold; margin: 10px 0;">
              题目、正确答案、选项A、选项B、选项C、...、选项J、解析
            </p>
            <table class="format-table">
              <thead>
                <tr>
                  <th>题目</th>
                  <th>正确答案</th>
                  <th>选项A</th>
                  <th>选项B</th>
                  <th>选项C</th>
                  <th>选项D</th>
                  <th>...</th>
                  <th>选项J</th>
                  <th>解析</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>这是一道示例题目？</td>
                  <td>A</td>
                  <td>选项A内容</td>
                  <td>选项B内容</td>
                  <td>选项C内容</td>
                  <td>选项D内容</td>
                  <td>...</td>
                  <td></td>
                  <td>答案解析（可选）</td>
                </tr>
                <tr>
                  <td>多选题示例？</td>
                  <td>"A,B,C"</td>
                  <td>选项A</td>
                  <td>选项B</td>
                  <td>选项C</td>
                  <td>选项D</td>
                  <td>...</td>
                  <td></td>
                  <td>多选题解析</td>
                </tr>
                <tr>
                  <td>最多选项示例？</td>
                  <td>J</td>
                  <td>选项A</td>
                  <td>选项B</td>
                  <td>选项C</td>
                  <td>选项D</td>
                  <td>...</td>
                  <td>选项J</td>
                  <td>支持最多10个选项</td>
                </tr>
              </tbody>
            </table>
            <p class="format-note">
              <strong>格式说明：</strong><br>
              • <strong>列顺序固定：</strong>题目（第1列）、正确答案（第2列，固定位置）、选项A-J（第3-12列）、解析（最后一列）<br>
              • <strong>CSV规则：</strong>使用逗号分隔，包含逗号的字段需要用双引号包围（如多选题答案 "A,B,C"）<br>
              • <strong>正确答案：</strong>可以填写 A/B/C/D/E/F/G/H/I/J 或 1/2/3/4/5/6/7/8/9/10，多选题使用逗号分隔（如 "A,B,C"）<br>
              • <strong>选项：</strong>至少需要选项A和B，最多支持10个选项（A-J）<br>
              • <strong>解析列：</strong>可选，放在最后<br>
              • <strong>列名：</strong>支持中英文列名，但列顺序需保持一致
            </p>
            <div class="demo-download">
              <el-button type="primary" size="small" @click="downloadDemo('csv')">
                <el-icon><Download /></el-icon>
                下载CSV示例文件
              </el-button>
            </div>
          </div>
        </el-collapse-item>
        
        <el-collapse-item title="Word格式示例" name="docx-example">
          <div class="format-example">
            <p><strong>Word（DOCX）文件固定格式说明：</strong></p>
            <div class="format-text-example">
              <pre>这是一道单选题？
A. 选项A的内容
B. 选项B的内容
C. 选项C的内容（可选）
D. 选项D的内容（可选）
答案：A
解析：这是单选题的解析（可选）

这是另一道单选题？
A. 第一个选项
B. 第二个选项
C. 第三个选项
D. 第四个选项
答案：B

多选题示例？
A. 选项A
B. 选项B
C. 选项C
D. 选项D
答案：A,B,C
解析：这是多选题的解析</pre>
            </div>
            <p class="format-note">
              <strong>格式要求：</strong><br>
              • 每道题目之间用空行分隔<br>
              • 题目内容可以跨多行<br>
              • 选项以 A. B. C. D. 等开头（至少需要 A 和 B，最多支持到 J）<br>
              • 答案格式：答案：A 或 答案：A,B,C（多选题使用逗号分隔）<br>
              • 答案也可以写成：正确答案：A 或 Answer：A<br>
              • 解析可选，格式：解析：解析内容 或 Explanation：解析内容<br>
              • 支持单选和多选题（根据答案个数自动判断）
            </p>
            <div class="demo-download">
              <el-button type="primary" size="small" @click="downloadDemo('docx')">
                <el-icon><Download /></el-icon>
                下载Word示例文件
              </el-button>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>

      <!-- 上传状态提示 -->
      <div v-if="isUploading" class="upload-status">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>{{ uploadStatus || '正在处理中...' }}</span>
      </div>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showUploadDialog = false" :disabled="isUploading">取消</el-button>
          <el-button type="primary" @click="uploadQuestionBank" :loading="isUploading" :disabled="isUploading">
            {{ isUploading ? '上传中...' : '确定上传' }}
          </el-button>
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
              <div style="margin: 10px 0;">
                <span style="margin-right: 10px; color: #909399; font-size: 12px;">
                  提示：选择单个答案为单选题，选择多个答案为多选题
                </span>
              </div>
              <div class="options-container">
                <div v-for="(option, optionIndex) in question.options" :key="optionIndex" class="option-item">
                  <span class="option-label">{{ String.fromCharCode(65 + optionIndex) }}:</span>
                  <el-input v-model="question.options[optionIndex]" placeholder="请输入选项内容" />
                  <el-button type="danger" size="small" @click="removeOption(index, optionIndex)" v-if="question.options.length > 2">
                    删除
                  </el-button>
                </div>
                <el-button type="text" @click="addOption(index)" v-if="question.options.length < 10">
                  + 添加选项（最多10个）
                </el-button>
                <div v-else style="color: #909399; font-size: 12px; margin-top: 8px;">
                  已达到最大选项数（10个）
                </div>
              </div>
              <div class="answer-container">
                <span>正确答案：</span>
                <!-- 使用checkbox组，支持单选和多选 -->
                <el-checkbox-group v-model="question.answer" style="display: flex; flex-direction: column; gap: 8px; margin-top: 8px;" @change="handleAnswerChange(index)">
                  <el-checkbox
                    v-for="(option, optionIndex) in question.options"
                    :key="optionIndex"
                    :label="optionIndex"
                    :disabled="!option.trim()"
                  >
                    {{ String.fromCharCode(65 + optionIndex) }}. {{ option || `选项 ${String.fromCharCode(65 + optionIndex)}` }}
                  </el-checkbox>
                </el-checkbox-group>
                <div v-if="!Array.isArray(question.answer) || question.answer.length === 0" style="color: #f56c6c; font-size: 12px; margin-top: 8px;">
                  请至少选择一个正确答案
                </div>
                <div v-else style="color: #409eff; font-size: 12px; margin-top: 8px;">
                  {{ question.answer.length === 1 ? '单选题（1个答案）' : `多选题（${question.answer.length}个答案）` }}
                </div>
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
import { Plus, Upload, MoreFilled, VideoPlay, Delete, Document, Calendar, Edit, Loading, Download } from '@element-plus/icons-vue'
import { API_BASE_URL } from '@/api'

export default {
  name: 'Library',
  components: {
    Plus,
    Upload,
    MoreFilled,
    VideoPlay,
    Delete,
    Document,
    Calendar,
    Loading,
    Download
  },
  setup() {
    const router = useRouter()
    const examStore = useExamStore()
    
    const showUploadDialog = ref(false)
    const showCreateDialog = ref(false)
    const showManageDialog = ref(false)
    const selectedFile = ref(null)
    const uploadRef = ref(null)  // el-upload 组件引用
    const isUploading = ref(false)
    const uploadStatus = ref('')
    const uploadForm = reactive({
      name: '',
      description: '',
      parseMode: 'format' // 'format' 或 'ai'
    })
    const createForm = reactive({
      name: '',
      description: '',
      questions: [
        {
          question: '',
          options: ['', ''],
          answer: [], // 答案数组，空数组表示未选择，根据数组长度自动判断单选/多选
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

    // JSON格式已移除，保留此变量以防其他地方引用
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

    const handleFileRemove = () => {
      selectedFile.value = null
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

      if (!uploadForm.parseMode) {
        ElMessage.error('请选择解析方式')
        return
      }

      const fileExt = selectedFile.value.name.split('.').pop().toLowerCase()
      
      // 验证文件格式和解析方式的兼容性
      if (uploadForm.parseMode === 'format') {
        if (fileExt === 'pdf' || fileExt === 'doc') {
          if (fileExt === 'pdf') {
            ElMessage.warning('PDF 文件不支持固定格式解析，请使用 AI 自动分析')
          } else {
            ElMessage.warning('旧版 DOC 格式不支持固定格式解析，请转换为 DOCX 格式或使用 AI 自动分析')
          }
          return
        }
      }

      // 开始上传，设置loading状态
      isUploading.value = true
      uploadStatus.value = '正在准备上传...'

      try {
        // 处理所有文件类型（Excel/CSV/PDF/DOC/DOCX）
        uploadStatus.value = '正在创建题库...'
        const newBank = await examStore.addQuestionBank({
          name: uploadForm.name,
          description: uploadForm.description,
          questions: [] // 先创建空题库
        })
        
        // 然后上传文件到这个题库
        uploadStatus.value = '正在上传文件...'
        const formData = new FormData()
        formData.append('file', selectedFile.value.raw)
        formData.append('parseMode', uploadForm.parseMode) // 传递解析模式
        
        // 根据解析模式设置不同的提示
        if (uploadForm.parseMode === 'ai') {
          uploadStatus.value = '正在使用AI解析试题，请稍候...'
        } else {
          if (fileExt === 'xlsx' || fileExt === 'xls') {
            uploadStatus.value = '正在按固定格式解析Excel文件...'
          } else if (fileExt === 'csv') {
            uploadStatus.value = '正在按固定格式解析CSV文件...'
          } else if (fileExt === 'docx') {
            uploadStatus.value = '正在按固定格式解析Word文件...'
          } else {
            uploadStatus.value = '正在按固定格式解析文件...'
          }
        }
        
        await examStore.uploadQuestionBankFile(newBank.id, formData)

        uploadStatus.value = '上传成功！'
        // 延迟一下让用户看到成功提示
        await new Promise(resolve => setTimeout(resolve, 500))
        
        showUploadDialog.value = false
        resetUploadForm()
        ElMessage.success('题库上传成功！')
      } catch (error) {
        ElMessage.error('文件上传失败: ' + error.message)
      } finally {
        // 无论成功或失败，都要重置loading状态
        isUploading.value = false
        uploadStatus.value = ''
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
      uploadForm.parseMode = 'format'
      selectedFile.value = null
      // 清空 el-upload 组件的文件列表
      if (uploadRef.value) {
        uploadRef.value.clearFiles()
      }
    }

    // 下载示例文件
    const downloadDemo = (type) => {
      // 使用统一的 API_BASE_URL，确保与打包脚本的配置一致
      const url = `${API_BASE_URL}/demo/${type}`
      
      // 获取token
      const token = localStorage.getItem('token')
      const headers = {}
      if (token) {
        headers.Authorization = `Bearer ${token}`
      }
      
      // 使用fetch下载文件
      fetch(url, {
        method: 'GET',
        headers: headers
      })
        .then(response => {
          if (!response.ok) {
            throw new Error('下载失败')
          }
          return response.blob()
        })
        .then(blob => {
          // 创建下载链接
          const downloadUrl = window.URL.createObjectURL(blob)
          const link = document.createElement('a')
          link.href = downloadUrl
          
          // 根据类型设置文件名
          let filename = '题库格式示例.csv'
          if (type === 'excel') {
            filename = '题库格式示例.xlsx'
          } else if (type === 'docx') {
            filename = '题库格式示例.docx'
          }
          link.download = filename
          
          // 触发下载
          document.body.appendChild(link)
          link.click()
          document.body.removeChild(link)
          
          // 清理URL对象
          window.URL.revokeObjectURL(downloadUrl)
          
          ElMessage.success('示例文件下载成功')
        })
        .catch(error => {
          console.error('下载失败:', error)
          ElMessage.error('下载失败: ' + error.message)
        })
    }

    const resetCreateForm = () => {
      createForm.name = ''
      createForm.description = ''
      createForm.questions = [
        {
          question: '',
          options: ['', ''],
          answer: [], // 答案数组，空数组表示未选择
          explanation: ''
        }
      ]
    }

    const addQuestion = () => {
      createForm.questions.push({
        question: '',
        options: ['', ''],
        answer: [], // 答案数组，空数组表示未选择
        explanation: ''
      })
    }

    // 处理答案变化（单选/多选自动判断）
    const handleAnswerChange = (questionIndex) => {
      const question = createForm.questions[questionIndex]
      if (!question || !Array.isArray(question.answer)) return
      
      // 确保answer始终是数组格式
      // 根据数组长度自动判断：1个答案=单选，多个答案=多选
      // 后端会根据答案数组长度自动设置is_multiple
    }

    const removeQuestion = (index) => {
      createForm.questions.splice(index, 1)
    }

    const addOption = (questionIndex) => {
      if (createForm.questions[questionIndex].options.length < 10) {
        createForm.questions[questionIndex].options.push('')
      } else {
        ElMessage.warning('最多只能添加10个选项')
      }
    }

    const removeOption = (questionIndex, optionIndex) => {
      const question = createForm.questions[questionIndex]
      if (question.options.length <= 2) {
        ElMessage.warning('至少需要2个选项')
        return
      }
      
      question.options.splice(optionIndex, 1)
      
      // 调整答案索引（答案始终是数组格式）
      if (Array.isArray(question.answer)) {
        question.answer = question.answer
          .filter(ans => ans !== optionIndex) // 移除被删除的选项
          .map(ans => ans > optionIndex ? ans - 1 : ans) // 调整大于被删除索引的选项
        // 如果答案数组为空且还有选项，不自动选择，让用户自己选择
      }
    }

    const createQuestionBank = async () => {
      if (!createForm.name.trim()) {
        ElMessage.error('请输入题库名称')
        return
      }

      // 检查题目列表是否为空
      if (!createForm.questions || createForm.questions.length === 0) {
        ElMessage.error('至少需要添加一道题目')
        return
      }

      let questions = []
      
      try {
        // 验证题目并转换答案格式为数组
        questions = createForm.questions.map((q, index) => {
          // 验证题目内容
          if (!q || !q.question || !q.question.trim()) {
            throw new Error(`第${index + 1}题的题目不能为空`)
          }
          
          // 验证选项
          if (!q.options || !Array.isArray(q.options) || q.options.length === 0) {
            throw new Error(`第${index + 1}题至少需要2个选项`)
          }
          
          // 过滤空选项，保留有效选项和索引映射
          const validOptions = []
          const indexMap = []
          for (let i = 0; i < q.options.length; i++) {
            if (q.options[i] && q.options[i].trim()) {
              indexMap[i] = validOptions.length
              validOptions.push(q.options[i].trim())
            }
          }
          
          if (validOptions.length < 2) {
            throw new Error(`第${index + 1}题至少需要2个有效选项`)
          }
          
          // 验证和转换答案：从原始索引转换为有效选项的索引（答案始终是数组格式）
          let answer = []
          if (!Array.isArray(q.answer)) {
            // 兼容旧数据：如果是单个数字，转换为数组
            if (q.answer !== null && q.answer !== undefined) {
              const originalIdx = typeof q.answer === 'number' ? q.answer : parseInt(q.answer)
              if (isNaN(originalIdx)) {
                throw new Error(`第${index + 1}题的答案格式无效`)
              }
              if (originalIdx >= 0 && originalIdx < indexMap.length && indexMap[originalIdx] !== undefined) {
                answer = [indexMap[originalIdx]]
              } else {
                throw new Error(`第${index + 1}题的答案索引无效（可能对应空选项）`)
              }
            } else {
              throw new Error(`第${index + 1}题必须选择正确答案`)
            }
          } else if (q.answer.length > 0) {
            // 答案已经是数组，转换索引
            for (const originalIdx of q.answer) {
              if (typeof originalIdx !== 'number' || isNaN(originalIdx)) {
                continue // 跳过无效的索引
              }
              if (originalIdx >= 0 && originalIdx < indexMap.length && indexMap[originalIdx] !== undefined) {
                const newIdx = indexMap[originalIdx]
                if (!answer.includes(newIdx)) {
                  answer.push(newIdx)
                }
              }
            }
            if (answer.length === 0) {
              throw new Error(`第${index + 1}题的答案索引无效（可能对应空选项）`)
            }
          } else {
            throw new Error(`第${index + 1}题必须至少选择一个正确答案`)
          }
          
          // 注意：后端会根据answer数组长度自动判断单选/多选（len(answer) > 1 = 多选）
          
          return {
            question: q.question.trim(),
            options: validOptions,
            answer: answer, // 确保是数组格式
            explanation: q.explanation ? q.explanation.trim() : ''
          }
        })

        // 如果所有题目验证通过，尝试创建题库
        await examStore.addQuestionBank({
          name: createForm.name,
          description: createForm.description,
          questions: questions
        })

        showCreateDialog.value = false
        resetCreateForm()
        ElMessage.success('题库创建成功！')
      } catch (error) {
        console.error('创建题库失败:', error)
        // 显示详细的错误信息
        const errorMessage = error.message || error.toString() || '创建题库失败'
        ElMessage.error(errorMessage)
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
      isUploading,
      uploadStatus,
      downloadDemo,
      showCreateDialog,
      showManageDialog,
      selectedFile,
      uploadRef,
      uploadForm,
      createForm,
      questionBanks,
      displayedBanks,
      loading,
      jsonExample,
      handleFileChange,
      handleFileRemove,
      uploadQuestionBank,
      resetUploadForm,
      resetCreateForm,
      addQuestion,
      removeQuestion,
      addOption,
      removeOption,
      handleAnswerChange,
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

.upload-status {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 15px;
  margin-bottom: 15px;
  background: #f0f9ff;
  border-radius: 8px;
  border: 1px solid #b3d8ff;
  color: #409eff;
  font-size: 14px;
}

.upload-status .el-icon {
  font-size: 18px;
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
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

.parse-mode-tip {
  margin-top: 10px;
  padding: 10px;
  background: #f0f9ff;
  border-radius: 4px;
  border-left: 3px solid #409eff;
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
}

.parse-mode-tip strong {
  color: #409eff;
}

.demo-download {
  margin-top: 15px;
  padding-top: 15px;
  border-top: 1px solid #e4e7ed;
  text-align: center;
}

.demo-download .el-button {
  margin: 0 5px;
}

.format-text-example {
  background: #f5f5f5;
  padding: 15px;
  border-radius: 8px;
  margin: 10px 0;
  overflow-x: auto;
}

.format-text-example pre {
  margin: 0;
  font-family: 'Courier New', Consolas, monospace;
  font-size: 13px;
  line-height: 1.8;
  color: #333;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.add-question-btn {
  width: 100%;
  margin-top: 16px;
  border-style: dashed;
}
</style>
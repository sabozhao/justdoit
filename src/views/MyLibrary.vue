<template>
  <div class="library">
    <!-- 面包屑导航 -->
    <div class="breadcrumb-container">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item>
          <span @click="goToRoot" class="breadcrumb-link">我的题库</span>
        </el-breadcrumb-item>
        <el-breadcrumb-item v-for="(path, index) in breadcrumbPaths" :key="index">
          <span v-if="index < breadcrumbPaths.length - 1" @click="goToCategory(path.id)" class="breadcrumb-link">
            {{ path.name }}
          </span>
          <span v-else>{{ path.name }}</span>
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <!-- 操作栏 -->
    <div class="library-toolbar">
      <div class="toolbar-left">
        <el-button type="success" @click="openUploadDialog">
          <el-icon><Upload /></el-icon>
          上传题库
        </el-button>
        <el-button type="warning" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>
          人工导题
        </el-button>
        <el-button type="primary" @click="showCategoryDialog = true">
          <el-icon><FolderAdd /></el-icon>
          新建分类
        </el-button>
        <el-dropdown @command="handleMoreCommand">
          <el-button>
            更多操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
              <el-dropdown-item command="batch-delete">批量删除</el-dropdown-item>
              <el-dropdown-item command="export">导出题库</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
      <div class="toolbar-right">
        <el-input
          v-model="searchKeyword"
          placeholder="输入您想搜索的题库名称"
          clearable
          style="width: 300px;"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
              </div>
              </div>

    <!-- 表格内容 -->
    <div class="library-content">
      <el-table
        v-if="!loading"
        :data="tableData"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        :default-expand-all="false"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" label="编号" width="80" sortable>
          <template #default="{ row, $index }">
            <span v-if="row.type === 'category'">{{ $index + 1 }}</span>
            <span v-else>{{ row.displayIndex || $index + 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="300" sortable>
          <template #default="{ row }">
            <div 
              class="name-cell" 
              :class="{ 'clickable': row.type === 'category' }"
              @click="row.type === 'category' ? enterCategory(row) : null"
            >
              <el-icon v-if="row.type === 'category'" class="folder-icon">
                <Folder />
              </el-icon>
              <el-icon v-else class="document-icon">
                <Document />
              </el-icon>
              <span class="name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="添加时间" width="180" sortable>
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="question_count" label="题目数量" width="120">
          <template #default="{ row }">
            <span v-if="row.type === 'category'">--</span>
            <span v-else>{{ row.question_count || 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button
                v-if="row.type === 'bank'"
                type="primary"
                link
                size="small"
                @click="startPractice(row)"
              >
                练习
              </el-button>
              <el-button
                v-if="row.type === 'category'"
                type="primary"
                link
                size="small"
                @click="addChildCategory(row)"
              >
                新建子分类
              </el-button>
              <el-button
                v-if="row.type === 'bank'"
                type="primary"
                link
                size="small"
                @click="manageQuestions(row)"
              >
                试题管理
              </el-button>
              <el-button
                v-if="row.type === 'bank'"
                type="primary"
                link
                size="small"
                @click="editBank(row)"
              >
                编辑
              </el-button>
              <el-button
                type="danger"
                link
                size="small"
                @click="handleDelete(row)"
              >
                删除
              </el-button>
          </div>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>

      <div v-if="!loading && tableData.length === 0" class="empty-state">
        <el-empty description="暂无数据">
          <el-button type="primary" @click="openUploadDialog">
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
        <el-form-item label="分类">
          <el-select v-model="uploadForm.categoryId" placeholder="选择分类（可选，不选则为未分类）" clearable style="width: 100%">
            <el-option label="未分类" value="" />
            <el-option 
              v-for="cat in flatCategories" 
              :key="cat.id" 
              :label="getCategoryPath(cat)" 
              :value="cat.id"
            />
          </el-select>
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
            :on-exceed="handleFileExceed"
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
        <el-form-item label="分类">
          <el-select v-model="createForm.categoryId" placeholder="选择分类（可选，不选则为未分类）" clearable style="width: 100%">
            <el-option label="未分类" value="" />
            <el-option 
              v-for="cat in flatCategories" 
              :key="cat.id" 
              :label="getCategoryPath(cat)" 
              :value="cat.id"
            />
          </el-select>
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
                <el-radio-group v-model="question.type" size="small" @change="handleCreateFormQuestionTypeChange(index)">
                  <el-radio label="choice">选择题</el-radio>
                  <el-radio label="judgment">判断题</el-radio>
                </el-radio-group>
                <span v-if="question.type === 'choice'" style="margin-left: 10px; color: #909399; font-size: 12px;">
                  提示：选择单个答案为单选题，选择多个答案为多选题
                </span>
              </div>
              <div v-if="question.type === 'choice'" class="options-container">
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
                <!-- 判断题答案选择 -->
                <div v-if="question.type === 'judgment'">
                  <el-radio-group v-model="question.answer" style="margin-top: 8px;">
                    <el-radio :label="0">错误</el-radio>
                    <el-radio :label="1">正确</el-radio>
                  </el-radio-group>
                </div>
                <!-- 选择题答案选择：使用checkbox组，支持单选和多选 -->
                <div v-else>
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
        <el-form-item label="题目类型">
          <el-radio-group v-model="currentQuestion.type" @change="handleQuestionTypeChange">
            <el-radio label="choice">选择题</el-radio>
            <el-radio label="judgment">判断题</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="选项" v-if="currentQuestion.type === 'choice'">
          <div v-for="(option, index) in currentQuestion.options" :key="index" style="margin-bottom: 10px;">
            <el-input v-model="currentQuestion.options[index]" :placeholder="`选项 ${String.fromCharCode(65 + index)}`">
              <template #prepend>{{ String.fromCharCode(65 + index) }}</template>
            </el-input>
          </div>
        </el-form-item>
        <el-form-item v-if="currentQuestion.type === 'judgment'" label="正确答案">
          <el-radio-group v-model="currentQuestion.answer">
            <el-radio :label="0">错误</el-radio>
            <el-radio :label="1">正确</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-else label="正确答案">
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

    <!-- 分类管理对话框 -->
    <el-dialog v-model="showCategoryDialog" title="管理分类" width="800px">
      <div style="margin-bottom: 20px;">
        <el-button type="primary" @click="showCreateCategoryDialog">
          <el-icon><Plus /></el-icon>
          创建分类
        </el-button>
      </div>
      
      <div v-if="categoryTree.length === 0" style="text-align: center; padding: 40px; color: #909399;">
        <el-empty description="暂无分类，请创建第一个分类" />
      </div>
      <div v-else class="category-tree-container">
        <el-tree
          :data="categoryTree"
          :props="{ children: 'children', label: 'name' }"
          default-expand-all
          node-key="id"
          :expand-on-click-node="false"
        >
          <template #default="{ node, data }">
            <div class="category-tree-node">
              <div class="category-info">
                <span class="category-name">{{ data.name }}</span>
                <span class="category-description" v-if="data.description">{{ data.description }}</span>
              </div>
              <div class="category-actions">
                <el-button size="small" type="primary" @click="editCategory(data)">编辑</el-button>
                <el-button size="small" type="success" @click="addChildCategory(data)">添加子分类</el-button>
                <el-button size="small" type="danger" @click="deleteCategory(data)">删除</el-button>
              </div>
            </div>
          </template>
        </el-tree>
      </div>
    </el-dialog>

    <!-- 编辑题库对话框 -->
    <el-dialog 
      v-model="showEditDialog" 
      title="编辑题库" 
      width="600px"
      @closed="() => { editForm.id = ''; editForm.name = ''; editForm.description = ''; editForm.categoryId = '' }"
    >
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="题库名称" required>
          <el-input v-model="editForm.name" placeholder="请输入题库名称" />
        </el-form-item>
        <el-form-item label="题库描述">
          <el-input v-model="editForm.description" type="textarea" :rows="4" placeholder="请输入题库描述（可选）" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="editForm.categoryId" placeholder="选择分类（可选，不选则为未分类）" clearable style="width: 100%">
            <el-option label="未分类" value="" />
            <el-option 
              v-for="cat in flatCategories" 
              :key="cat.id" 
              :label="getCategoryPath(cat)" 
              :value="cat.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="saveEditBank">保存</el-button>
      </template>
    </el-dialog>

    <!-- 分类编辑对话框 -->
    <el-dialog 
      v-model="categoryDialogVisible" 
      :title="isEditingCategory ? '编辑分类' : '创建分类'" 
      width="500px"
    >
      <el-form :model="categoryForm" label-width="100px">
        <el-form-item label="分类名称" required>
          <el-input v-model="categoryForm.name" placeholder="请输入分类名称" />
        </el-form-item>
        <el-form-item label="分类描述">
          <el-input v-model="categoryForm.description" type="textarea" placeholder="请输入分类描述（可选）" />
        </el-form-item>
        <el-form-item label="父分类">
          <el-select v-model="categoryForm.parent_id" placeholder="选择父分类（可选）" clearable style="width: 100%">
            <el-option label="无（顶级分类）" :value="null" />
            <el-option 
              v-for="cat in flatCategories" 
              :key="cat.id" 
              :label="getCategoryPath(cat)" 
              :value="cat.id"
              :disabled="cat.id === categoryForm.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="categoryForm.sort_order" :min="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="categoryDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveCategory">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, inject, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useExamStore } from '@/stores/exam'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Upload, MoreFilled, VideoPlay, Delete, Document, Calendar, Edit, Loading, Download, Folder, FolderAdd, Search, ArrowDown, Setting } from '@element-plus/icons-vue'
import { API_BASE_URL, categoryAPI } from '@/api'

export default {
  name: 'MyLibrary',
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
    
    // 我的题库只显示个人题库
    const showUploadDialog = ref(false)
    const showCreateDialog = ref(false)
    const showManageDialog = ref(false)
    const showEditDialog = ref(false)
    const selectedFile = ref(null)
    const uploadRef = ref(null)  // el-upload 组件引用
    const isUploading = ref(false)
    const uploadStatus = ref('')
    // 筛选相关状态
    const selectedCategoryId = ref('')
    const categories = ref([])
    const showCategoryDialog = ref(false)
    const categoryTree = ref([])
    const categoryDialogVisible = ref(false)
    const categoryForm = ref({
      id: '',
      name: '',
      description: '',
      parent_id: null,
      sort_order: 0
    })
    const isEditingCategory = ref(false)
    
    // 表格和搜索相关状态
    const searchKeyword = ref('')
    const currentPath = ref('全部题库')
    const selectedRows = ref([])
    
    // 当前分类路径（用于面包屑和筛选）
    const currentCategoryId = ref(null) // null 表示根目录
    const breadcrumbPaths = ref([]) // 面包屑路径数组
    
    // 编辑题库表单
    const editForm = reactive({
      id: '',
      name: '',
      description: '',
      categoryId: ''
    })
    
    const uploadForm = reactive({
      name: '',
      description: '',
      parseMode: 'format', // 'format' 或 'ai'
      categoryId: '' // 分类ID（空字符串表示未分类）
    })
    const createForm = reactive({
      name: '',
      description: '',
      categoryId: '', // 分类ID（空字符串表示未分类）
      questions: [
        {
          question: '',
          options: ['', ''],
          answer: [], // 答案数组，空数组表示未选择，根据数组长度自动判断单选/多选
          type: 'choice', // 题目类型：choice（选择题）或judgment（判断题）
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
      type: 'choice', // 题目类型：choice（选择题）或judgment（判断题）
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
    // 加载分类
    const loadCategories = async () => {
      try {
        const result = await categoryAPI.getAll()
        console.log('获取到的分类数据（原始）:', JSON.parse(JSON.stringify(result)))
        
        // 后端返回的是树形结构（rootCategories），直接使用
        // 但需要确保数据结构正确
        if (result && Array.isArray(result)) {
          // 后端已经返回树形结构，直接使用
          categoryTree.value = result
          
          // 扁平化用于其他用途（如下拉选择）
          const flattenTree = (nodes) => {
            let flat = []
            for (const node of nodes) {
              // 移除children字段，创建扁平对象
              const flatNode = { ...node }
              delete flatNode.children
              flat.push(flatNode)
              if (node.children && Array.isArray(node.children) && node.children.length > 0) {
                flat.push(...flattenTree(node.children))
              }
            }
            return flat
          }
          categories.value = flattenTree(result)
        } else {
          // 如果后端返回的不是数组，使用空数组
          categoryTree.value = []
          categories.value = []
        }
        
        // 调试：检查分类树的层级深度
        const checkDepth = (nodes, depth = 0) => {
          let maxDepth = depth
          for (const node of nodes) {
            if (node.children && Array.isArray(node.children) && node.children.length > 0) {
              const childDepth = checkDepth(node.children, depth + 1)
              maxDepth = Math.max(maxDepth, childDepth)
            }
          }
          return maxDepth
        }
        const maxDepth = checkDepth(categoryTree.value)
        console.log('分类树最大深度:', maxDepth)
        console.log('构建后的分类树:', JSON.parse(JSON.stringify(categoryTree.value)))
      } catch (error) {
        console.error('加载分类失败:', error)
        // 不显示错误，因为分类是可选的
      }
    }
    
    // 构建表格数据（树形结构：分类作为父节点，题库作为子节点）
    const tableData = computed(() => {
      const result = []
      const categoryMap = new Map()
      
      // 将后端返回的树形分类转换为表格节点（支持多层级）
      const convertCategoryTreeToNodes = (categoryTree, depth = 0) => {
        if (!categoryTree || !Array.isArray(categoryTree)) {
          return []
        }
        
        return categoryTree.map(cat => {
          const categoryNode = {
            id: `category-${cat.id}`,
            type: 'category',
            name: cat.name,
            description: cat.description || '',
            created_at: cat.created_at,
            question_count: 0,
            categoryId: cat.id,
            children: [],
            hasChildren: false
          }
          
          // 递归处理子分类（支持多层级）
          if (cat.children && Array.isArray(cat.children) && cat.children.length > 0) {
            const childNodes = convertCategoryTreeToNodes(cat.children, depth + 1)
            categoryNode.children = childNodes
            categoryNode.hasChildren = childNodes.length > 0
            // 调试：打印每个层级的分类
            console.log(`层级 ${depth + 1} 分类 "${cat.name}" 有 ${childNodes.length} 个子分类`)
          }
          
          categoryMap.set(cat.id, categoryNode)
          return categoryNode
        })
      }
      
      // 使用已构建好的分类树（支持多层级）
      const categoryNodes = convertCategoryTreeToNodes(categoryTree.value)
      result.push(...categoryNodes)
      
      // 将题库添加到对应的分类下
      displayedBanks.value.forEach(bank => {
        const bankNode = {
          id: `bank-${bank.id}`,
          type: 'bank',
          name: bank.name,
          description: bank.description || '',
          created_at: bank.created_at,
          question_count: bank.question_count || 0,
          bankId: bank.id,
          categoryId: bank.category_id,
          children: [] // 题库没有子节点
        }
        
        if (bank.category_id) {
          // 递归查找对应的分类节点（支持多层级查找）
          const findCategoryNode = (nodes, categoryId) => {
            for (const node of nodes) {
              if (node.categoryId === categoryId) {
                return node
              }
              // 递归查找子分类（支持多层级）
              if (node.children && node.children.length > 0) {
                const found = findCategoryNode(node.children, categoryId)
                if (found) return found
              }
            }
            return null
          }
          
          const categoryNode = findCategoryNode(result, bank.category_id)
          if (categoryNode) {
            // 将题库添加到分类的 children 中（与子分类一起）
            categoryNode.children.push(bankNode)
            // 更新 hasChildren（因为现在有子节点了）
            if (!categoryNode.hasChildren) {
              categoryNode.hasChildren = true
            }
            categoryNode.question_count += bank.question_count || 0
          } else {
            // 分类不存在，放到未分类
            result.push(bankNode)
          }
        } else {
          // 未分类的题库，直接添加到根节点
          result.push(bankNode)
        }
      })
      
      // 调试：打印最终的数据结构
      console.log('tableData 最终结构:', JSON.parse(JSON.stringify(result)))
      
      // 如果当前在某个分类内，只显示该分类下的内容
      if (currentCategoryId.value !== null) {
        const findCategoryInTree = (nodes, categoryId) => {
          for (const node of nodes) {
            if (node.categoryId === categoryId) {
              return node
            }
            if (node.children && node.children.length > 0) {
              const found = findCategoryInTree(node.children, categoryId)
              if (found) return found
            }
          }
          return null
        }
        
        const targetCategory = findCategoryInTree(result, currentCategoryId.value)
        if (targetCategory) {
          // 返回该分类下的子分类和题库
          const children = targetCategory.children || []
          // 按创建时间排序
          const sortByDate = (items) => {
            items.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
            items.forEach(item => {
              if (item.children && item.children.length > 0) {
                sortByDate(item.children)
              }
            })
          }
          sortByDate(children)
          return children
        } else {
          // 分类不存在，返回空数组
          return []
        }
      }
      
      // 按创建时间排序
      const sortByDate = (items) => {
        items.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
        items.forEach(item => {
          if (item.children && item.children.length > 0) {
            sortByDate(item.children)
          }
        })
      }
      sortByDate(result)
      
      return result
    })
    
    // 进入分类
    const enterCategory = (categoryRow) => {
      const categoryId = categoryRow.categoryId
      if (!categoryId) return
      
      // 更新当前分类ID
      currentCategoryId.value = categoryId
      
      // 更新面包屑
      updateBreadcrumb(categoryId)
    }
    
    // 更新面包屑导航
    const updateBreadcrumb = (categoryId) => {
      // 递归查找分类路径
      const findCategoryPath = (cats, targetId, currentPath = []) => {
        for (const cat of cats) {
          const newPath = [...currentPath, { id: cat.id, name: cat.name }]
          
          if (cat.id === targetId) {
            return newPath
          }
          
          if (cat.children && cat.children.length > 0) {
            const found = findCategoryPath(cat.children, targetId, newPath)
            if (found) return found
          }
        }
        return null
      }
      
      // 构建扁平化的分类列表用于查找（从原始categories构建树）
      const buildCategoryTree = (cats, parentId = null) => {
        return cats
          .filter(cat => {
            if (parentId === null) {
              return !cat.parent_id || cat.parent_id === null
            }
            return cat.parent_id === parentId
          })
          .map(cat => ({
            ...cat,
            children: buildCategoryTree(cats, cat.id)
          }))
      }
      
      const categoryTree = buildCategoryTree(categories.value)
      const path = findCategoryPath(categoryTree, categoryId)
      
      if (path) {
        breadcrumbPaths.value = path
      } else {
        // 如果找不到，至少显示当前分类名称
        const category = categories.value.find(c => c.id === categoryId)
        if (category) {
          breadcrumbPaths.value = [{ id: categoryId, name: category.name }]
        } else {
          breadcrumbPaths.value = []
        }
      }
    }
    
    // 返回根目录
    const goToRoot = () => {
      currentCategoryId.value = null
      breadcrumbPaths.value = []
      currentPath.value = '全部题库'
    }
    
    // 跳转到指定分类
    const goToCategory = (categoryId) => {
      currentCategoryId.value = categoryId
      updateBreadcrumb(categoryId)
    }
    
    // 扁平化分类列表（用于下拉选择，支持多层级）
    const flatCategories = computed(() => {
      const flatten = (cats, parentPath = '') => {
        let result = []
        for (const cat of cats) {
          const path = parentPath ? `${parentPath} / ${cat.name}` : cat.name
          result.push({ ...cat, path })
          if (cat.children && Array.isArray(cat.children) && cat.children.length > 0) {
            result = result.concat(flatten(cat.children, path))
          }
        }
        return result
      }
      // 使用树形结构（支持多层级）
      return flatten(categoryTree.value)
    })
    
    // 获取分类路径
    const getCategoryPath = (cat) => {
      return cat.path || cat.name
    }
    
    // 显示创建分类对话框
    const showCreateCategoryDialog = (parentCategory = null) => {
      // 获取父分类ID：如果是表格行数据，使用 categoryId；否则使用 id
      let parentId = null
      if (parentCategory) {
        if (parentCategory.categoryId) {
          // 表格行数据，使用 categoryId
          parentId = parentCategory.categoryId
        } else if (parentCategory.id) {
          // 完整的分类对象，使用 id
          parentId = parentCategory.id
        }
      }
      
      categoryForm.value = {
        id: '',
        name: '',
        description: '',
        parent_id: parentId,
        sort_order: 0
      }
      isEditingCategory.value = false
      categoryDialogVisible.value = true
    }
    
    // 添加子分类
    const addChildCategory = (parentCategory) => {
      // showCreateCategoryDialog 已经能正确处理表格行数据和分类对象
      showCreateCategoryDialog(parentCategory)
    }
    
    // 编辑分类
    const editCategory = (category) => {
      categoryForm.value = {
        id: category.id,
        name: category.name,
        description: category.description || '',
        parent_id: category.parent_id || null,
        sort_order: category.sort_order || 0
      }
      isEditingCategory.value = true
      categoryDialogVisible.value = true
    }
    
    // 保存分类
    const saveCategory = async () => {
      if (!categoryForm.value.name.trim()) {
        ElMessage.error('请输入分类名称')
        return
      }
      
      try {
        if (isEditingCategory.value) {
          // 更新分类
          await categoryAPI.update(categoryForm.value.id, {
            name: categoryForm.value.name,
            description: categoryForm.value.description,
            parent_id: categoryForm.value.parent_id,
            sort_order: categoryForm.value.sort_order
          })
          ElMessage.success('分类更新成功')
        } else {
          // 创建分类
          await categoryAPI.create({
            name: categoryForm.value.name,
            description: categoryForm.value.description,
            parent_id: categoryForm.value.parent_id,
            sort_order: categoryForm.value.sort_order
          })
          ElMessage.success('分类创建成功')
        }
        
        categoryDialogVisible.value = false
        // 重新加载分类树，确保子分类立即显示
        await loadCategories()
        await loadBanks() // 刷新题库列表
        console.log('保存分类后，分类树已更新:', categoryTree.value)
      } catch (error) {
        console.error('保存分类失败:', error)
        ElMessage.error('保存分类失败: ' + (error.message || error))
      }
    }
    
    // 删除分类
    const deleteCategory = async (category) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除分类"${category.name}"吗？删除后，该分类下的题库将自动归为"未分类"。此操作不可恢复。`,
          '确认删除',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        await categoryAPI.delete(category.id)
        ElMessage.success('分类删除成功')
        await loadCategories()
        await loadBanks() // 刷新题库列表
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除分类失败:', error)
          ElMessage.error('删除分类失败: ' + (error.message || error))
        }
      }
    }
    
    // 处理分类切换
    const handleCategoryChange = () => {
      loadBanks()
    }
    
    // 加载题库（只加载个人题库）
    const loadBanks = async () => {
      const params = { type: 'personal' }
      if (selectedCategoryId.value && selectedCategoryId.value !== 'uncategorized') {
        params.category_id = selectedCategoryId.value
      }
      // 如果选择了"未分类"，不传 category_id，让后端返回所有，然后前端过滤
      await examStore.loadQuestionBanks(params)
    }
    
    // 过滤显示的题库（包括未分类筛选）
    const displayedBanks = computed(() => {
      const banks = questionBanks.value || []
      if (selectedCategoryId.value === 'uncategorized') {
        // 只显示未分类的（category_id 为 null 或空）
        return banks.filter(bank => bank && bank.id && !bank.category_id)
      }
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

    const handleFileChange = (file, fileList) => {
      // 处理文件变化，支持文件替换
      // fileList 是当前文件列表，file 是当前变化的文件
      if (fileList && fileList.length > 0) {
        // 获取文件列表中的最后一个文件（最新选择的文件）
        const latestFile = fileList[fileList.length - 1]
        selectedFile.value = latestFile
      } else if (file) {
        // 如果文件列表为空但有文件对象，使用该文件
        selectedFile.value = file
      } else {
        // 如果都没有，清空
        selectedFile.value = null
      }
    }

    const handleFileRemove = (file, fileList) => {
      // 如果文件列表为空，清空selectedFile
      if (!fileList || fileList.length === 0) {
        selectedFile.value = null
      } else {
        // 如果还有文件，使用最新的文件
        selectedFile.value = fileList[fileList.length - 1]
      }
    }

    // 处理文件超出限制的情况（当limit=1时，选择新文件会触发此事件）
    const handleFileExceed = (files, fileList) => {
      // 当选择新文件时，自动替换旧文件
      // Element Plus的upload组件在limit=1时会自动替换，我们只需要更新selectedFile
      if (files && files.length > 0) {
        const newFile = files[0]
        // 先清空文件列表
        if (uploadRef.value) {
          uploadRef.value.clearFiles()
        }
        // 更新selectedFile为新文件
        selectedFile.value = newFile
        // 延迟添加新文件，确保清空操作完成
        setTimeout(() => {
          if (uploadRef.value) {
            // Element Plus的upload组件支持handleStart方法添加文件
            try {
              if (uploadRef.value.handleStart) {
                uploadRef.value.handleStart(newFile)
              }
            } catch (e) {
              // 如果handleStart不存在，通过on-change事件会自动处理
              console.log('文件替换:', newFile.name)
            }
          }
        }, 50)
      }
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
        // 直接上传文件并创建新题库（使用 'new' 作为 bankId）
        uploadStatus.value = '正在上传文件...'
        const formData = new FormData()
        formData.append('file', selectedFile.value.raw)
        formData.append('parseMode', uploadForm.parseMode) // 传递解析模式
        formData.append('bankName', uploadForm.name) // 题库名称
        formData.append('is_public', 'false') // 我的题库始终是个人题库，即使管理员创建
        if (uploadForm.categoryId) {
          formData.append('category_id', uploadForm.categoryId) // 分类ID
        }
        // 不传 category_id 或传空字符串表示未分类
        
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
        
        // 使用 'new' 作为 bankId，后端会自动创建新题库
        const result = await examStore.uploadQuestionBankFile('new', formData)

        uploadStatus.value = '上传成功！'
        // 延迟一下让用户看到成功提示
        await new Promise(resolve => setTimeout(resolve, 500))
        
        // 重新加载题库列表（保持当前筛选条件）
        await loadBanks()
        
        showUploadDialog.value = false
        resetUploadForm()
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

    // 打开上传对话框
    const openUploadDialog = () => {
      // 设置分类为当前分类（如果有）
      uploadForm.categoryId = currentCategoryId.value || ''
      showUploadDialog.value = true
    }
    
    const resetUploadForm = () => {
      uploadForm.name = ''
      uploadForm.description = ''
      uploadForm.parseMode = 'format'
      uploadForm.isPublic = false
      uploadForm.categoryId = ''
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

    // 打开人工导题对话框
    const openCreateDialog = () => {
      // 设置分类为当前分类（如果有）
      createForm.categoryId = currentCategoryId.value || ''
      showCreateDialog.value = true
    }
    
    const resetCreateForm = () => {
      createForm.name = ''
      createForm.description = ''
      createForm.categoryId = '' // 重置分类
      createForm.questions = [
        {
          question: '',
          options: ['', ''],
          answer: [], // 答案数组，空数组表示未选择
          type: 'choice', // 题目类型：choice（选择题）或judgment（判断题）
          explanation: ''
        }
      ]
    }

    const addQuestion = () => {
      createForm.questions.push({
        question: '',
        options: ['', ''],
        answer: [], // 答案数组，空数组表示未选择
        type: 'choice', // 题目类型：choice（选择题）或judgment（判断题）
        explanation: ''
      })
    }

    // 处理手动创建题库中的题目类型变化
    const handleCreateFormQuestionTypeChange = (questionIndex) => {
      const question = createForm.questions[questionIndex]
      if (!question) return
      
      if (question.type === 'judgment') {
        // 判断题：选项固定为["错误", "正确"]，答案默认为0（错误）
        question.options = ['错误', '正确']
        question.answer = 0
      } else {
        // 选择题：恢复选项编辑功能，至少2个选项
        if (question.options.length < 2) {
          question.options = ['', '']
        }
        // 重置答案
        question.answer = []
      }
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
          
          const questionType = q.type || 'choice'
          let validOptions = []
          let answer = []
          
          if (questionType === 'judgment') {
            // 判断题：选项固定为["错误", "正确"]，答案：0=错误，1=正确
            validOptions = ['错误', '正确']
            const answerValue = q.answer
            if (answerValue !== 0 && answerValue !== 1) {
              throw new Error(`第${index + 1}题（判断题）答案必须是0（错误）或1（正确）`)
            }
            answer = [answerValue]
          } else {
            // 选择题：验证选项
            if (!q.options || !Array.isArray(q.options) || q.options.length === 0) {
              throw new Error(`第${index + 1}题至少需要2个选项`)
            }
            
            // 过滤空选项，保留有效选项和索引映射
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
          }
          
          // 注意：后端会根据answer数组长度自动判断单选/多选（len(answer) > 1 = 多选）
          
          return {
            question: q.question.trim(),
            options: validOptions,
            answer: answer, // 确保是数组格式
            type: questionType, // 题目类型：choice（选择题）或judgment（判断题）
            explanation: q.explanation ? q.explanation.trim() : ''
          }
        })

        // 如果所有题目验证通过，尝试创建题库
        await examStore.addQuestionBank({
          name: createForm.name,
          description: createForm.description,
          category_id: createForm.categoryId || null, // 分类ID（null表示未分类）
          is_public: false, // 我的题库始终是个人题库，即使管理员创建
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

    const startPractice = (rowOrId) => {
      // 支持传入 row 对象或 bankId 字符串
      let bankId
      if (typeof rowOrId === 'string') {
        bankId = rowOrId
      } else if (rowOrId.bankId) {
        bankId = rowOrId.bankId
      } else if (rowOrId.id) {
        // 处理 bank-xxx 格式的ID
        bankId = rowOrId.id.replace(/^bank-/, '')
      } else {
        ElMessage.error('无法获取题库ID')
        return
      }
      console.log('startPractice 被调用，题库ID:', bankId)
      console.log('即将跳转到路由:', `/exam/${bankId}`)
      router.push(`/exam/${bankId}`)
    }

    // 编辑题库
    const editBank = async (row) => {
      // 从 row 中提取 bankId
      let bankId
      if (row.bankId) {
        bankId = row.bankId
      } else if (row.id) {
        // 处理 bank-xxx 格式的ID
        bankId = row.id.replace(/^bank-/, '')
      } else {
        ElMessage.error('无法获取题库ID')
        return
      }
      
      try {
        // 获取题库详情
        const bank = displayedBanks.value.find(b => b && b.id === bankId)
        if (!bank) {
          // 如果本地找不到，尝试从API获取
          const bankDetails = await examStore.getQuestionBankDetails(bankId)
          editForm.id = bankDetails.id
          editForm.name = bankDetails.name || ''
          editForm.description = bankDetails.description || ''
          editForm.categoryId = bankDetails.category_id || ''
        } else {
          editForm.id = bank.id
          editForm.name = bank.name || ''
          editForm.description = bank.description || ''
          editForm.categoryId = bank.category_id || ''
        }
        showEditDialog.value = true
      } catch (error) {
        console.error('获取题库详情失败:', error)
        ElMessage.error('获取题库详情失败: ' + (error.message || error))
      }
    }
    
    // 保存编辑的题库
    const saveEditBank = async () => {
      if (!editForm.name || editForm.name.trim() === '') {
        ElMessage.warning('请输入题库名称')
        return
      }
      
      try {
        const updateData = {
          name: editForm.name.trim(),
          description: editForm.description.trim(),
          category_id: editForm.categoryId || null
        }
        
        await examStore.updateQuestionBank(editForm.id, updateData)
        await loadBanks()
        showEditDialog.value = false
        // 重置表单
        editForm.id = ''
        editForm.name = ''
        editForm.description = ''
        editForm.categoryId = ''
      } catch (error) {
        console.error('更新题库失败:', error)
        ElMessage.error('更新题库失败: ' + (error.message || error))
      }
    }
    
    // 处理删除（分类或题库）
    // skipConfirm: 是否跳过确认对话框和成功提示（用于批量删除）
    const handleDelete = async (row, skipConfirm = false) => {
      if (row.type === 'category') {
        // 删除分类
        const category = categories.value.find(c => c.id === row.categoryId)
        if (category) {
          if (skipConfirm) {
            // 批量删除时，直接删除分类（不弹出确认，不显示成功提示）
            try {
              await categoryAPI.delete(category.id)
              // 批量删除时不显示单个成功提示，统一在批量删除完成后显示
            } catch (error) {
              console.error('删除分类失败:', error)
              // 批量删除时也不显示单个错误提示，统一在批量删除完成后显示
              throw error // 抛出错误以便统一处理
            }
          } else {
            // 单个删除时，使用原有的确认逻辑（会显示成功提示）
            await deleteCategory(category)
          }
        }
      } else {
        // 删除题库
        const bankId = row.bankId || row.id?.replace('bank-', '')
        try {
          if (!skipConfirm) {
            // 单个删除时，弹出确认对话框
            await ElMessageBox.confirm('确定要删除这个题库吗？', '确认删除', {
              type: 'warning',
              confirmButtonText: '确定删除',
              cancelButtonText: '取消'
            })
          }
          // 执行删除操作
          // skipConfirm为true时，不显示成功提示（批量删除）
          await examStore.deleteQuestionBank(bankId, skipConfirm)
          if (!skipConfirm) {
            // 单个删除时，刷新列表（批量删除时在批量删除完成后统一刷新）
            await loadBanks()
          }
          // 批量删除时不显示单个成功提示，统一在批量删除完成后显示
        } catch (error) {
          if (error !== 'cancel') {
            console.error('删除题库失败:', error)
            if (!skipConfirm) {
              // 单个删除时显示错误提示
              ElMessage.error('删除题库失败: ' + (error.message || error))
            }
            // 批量删除时抛出错误以便统一处理
            throw error
          }
        }
      }
    }
    
    // 处理搜索
    const handleSearch = () => {
      // 搜索逻辑已在 displayedBanks computed 中实现
    }
    
    // 处理更多操作
    const handleMoreCommand = (command) => {
      if (command === 'batch-delete') {
        if (selectedRows.value.length === 0) {
          ElMessage.warning('请先选择要删除的项目')
          return
        }
        ElMessageBox.confirm(
          `确定要删除选中的 ${selectedRows.value.length} 项吗？`,
          '确认批量删除',
          {
            type: 'warning',
            confirmButtonText: '确定删除',
            cancelButtonText: '取消'
          }
        ).then(async () => {
          // 批量删除逻辑：跳过每个项目的确认对话框
          const deletePromises = []
          for (const row of selectedRows.value) {
            deletePromises.push(handleDelete(row, true).catch(err => {
              // 记录错误但不中断其他删除操作
              console.error('删除失败:', row, err)
              return { success: false, row, error: err }
            }))
          }
          
          // 等待所有删除操作完成
          const results = await Promise.all(deletePromises)
          const successCount = results.filter(r => !r || r.success !== false).length
          const failCount = results.filter(r => r && r.success === false).length
          
          // 清空选择
          selectedRows.value = []
          
          // 刷新列表
          await loadCategories()
          await loadBanks()
          
          // 显示结果提示
          if (failCount === 0) {
            ElMessage.success(`成功删除 ${successCount} 项`)
          } else {
            ElMessage.warning(`成功删除 ${successCount} 项，失败 ${failCount} 项`)
          }
        }).catch(() => {
          // 用户取消批量删除
        })
      } else if (command === 'export') {
        ElMessage.info('导出功能开发中...')
      }
    }
    
    // 处理表格选择变化
    const handleSelectionChange = (selection) => {
      selectedRows.value = selection
    }

    const handleCommand = async (command) => {
      console.log('handleCommand 接收到命令:', command)
      const [action, ...bankIdParts] = command.split('-')
      const bankId = bankIdParts.join('-')
      console.log('解析结果 - action:', action, 'bankId:', bankId)
      
      if (action === 'edit') {
        const bank = displayedBanks.value.find(b => b.id === bankId)
        if (bank) {
          editBank(bank)
        }
      } else if (action === 'practice') {
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
    const manageQuestions = async (rowOrId) => {
      // 支持传入 row 对象或 bankId 字符串
      let bankId
      if (typeof rowOrId === 'string') {
        bankId = rowOrId
      } else if (rowOrId.bankId) {
        bankId = rowOrId.bankId
      } else if (rowOrId.id) {
        // 处理 bank-xxx 格式的ID
        bankId = rowOrId.id.replace(/^bank-/, '')
      } else {
        ElMessage.error('无法获取题库ID')
        return
      }
      
      try {
        const bank = questionBanks.value.find(b => b && b.id === bankId)
        if (!bank || !bank.id) {
          // 如果本地找不到，尝试从API获取
          try {
            const bankDetails = await examStore.getQuestionBankDetails(bankId)
            currentBank.value = bankDetails
          } catch (err) {
          ElMessage.error('题库不存在或数据不完整')
          return
        }
        } else {
        currentBank.value = bank
        }
        
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
        type: 'choice', // 默认选择题
        explanation: ''
      }
      showQuestionEditDialog.value = true
    }

    // 处理题目类型变化
    const handleQuestionTypeChange = () => {
      const questionType = currentQuestion.value.type
      if (questionType === 'judgment') {
        // 判断题：选项固定为["错误", "正确"]，答案默认为0（错误）
        currentQuestion.value.options = ['错误', '正确']
        currentQuestion.value.answer = 0
      } else {
        // 选择题：恢复选项编辑功能
        if (currentQuestion.value.options.length < 2) {
          currentQuestion.value.options = ['', '']
        }
      }
    }
    
    // 编辑题目
    const editQuestion = (question) => {
      const questionType = question.type || 'choice'
      currentQuestion.value = {
        id: question.id,
        question: question.question,
        options: questionType === 'judgment' ? ['错误', '正确'] : [...question.options],
        answer: questionType === 'judgment' ? (Array.isArray(question.answer) ? question.answer[0] : question.answer) : question.answer,
        type: questionType,
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
      
      // 验证题目内容
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
        // 选择题：验证选项
        validOptions = currentQuestion.value.options.filter(opt => opt.trim())
        if (validOptions.length < 2) {
          ElMessage.warning('至少需要2个有效选项')
          return
        }
        
        // 验证答案
        const answer = currentQuestion.value.answer
        if (answer === null || answer === undefined) {
          ElMessage.warning('请选择正确答案')
          return
        }
        
        // 找到答案在有效选项中的索引
        const originalIndex = answer
        if (originalIndex < 0 || originalIndex >= currentQuestion.value.options.length) {
          ElMessage.warning('答案索引无效')
          return
        }
        
        // 找到该选项在有效选项中的位置
        let validIndex = -1
        let currentValidIndex = 0
        for (let i = 0; i < currentQuestion.value.options.length; i++) {
          if (currentQuestion.value.options[i].trim()) {
            if (i === originalIndex) {
              validIndex = currentValidIndex
              break
            }
            currentValidIndex++
          }
        }
        
        if (validIndex === -1) {
          ElMessage.warning('答案对应的选项为空')
          return
        }
        
        finalAnswer = [validIndex]
      }
      
      try {
        const questionData = {
          bank_id: currentBank.value.id,
          question: currentQuestion.value.question.trim(),
          options: validOptions,
          answer: finalAnswer,
          type: questionType,
          explanation: currentQuestion.value.explanation || ''
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
        await loadCategories()
        await loadBanks()
      } catch (error) {
        console.error('加载失败:', error)
        ElMessage.error('加载失败，请刷新页面重试')
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
      handleFileExceed,
      uploadQuestionBank,
      openUploadDialog,
      resetUploadForm,
      openCreateDialog,
      resetCreateForm,
      addQuestion,
      removeQuestion,
      addOption,
      removeOption,
      handleAnswerChange,
      handleCreateFormQuestionTypeChange,
      createQuestionBank,
      startPractice,
      handleCommand,
      editBank,
      formatDate,
      // 题目管理相关变量
      currentBank,
      bankQuestions,
      // 筛选相关
      selectedCategoryId,
      categories,
      flatCategories,
      getCategoryPath,
      handleCategoryChange,
      loadBanks,
      // 分类管理相关
      showCategoryDialog,
      categoryTree,
      categoryDialogVisible,
      categoryForm,
      isEditingCategory,
      showCreateCategoryDialog,
      addChildCategory,
      editCategory,
      // 表格相关
      tableData,
      searchKeyword,
      currentPath,
      selectedRows,
      handleSearch,
      handleDelete,
      handleMoreCommand,
      handleSelectionChange,
      manageQuestions,
      saveCategory,
      deleteCategory,
      showQuestionEditDialog,
      currentQuestion,
      addNewQuestion,
      editQuestion,
      saveQuestion,
      deleteQuestion,
      handleQuestionTypeChange,
      // 编辑题库相关
      showEditDialog,
      editForm,
      saveEditBank,
      // 分类导航相关
      breadcrumbPaths,
      enterCategory,
      goToRoot,
      goToCategory
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

/* 面包屑导航 */
.breadcrumb-container {
  margin-bottom: 20px;
  padding: 10px 0;
}

/* 操作栏 */
.library-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 15px 20px;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.toolbar-left {
  display: flex;
  gap: 10px;
  align-items: center;
}

.toolbar-right {
  display: flex;
  gap: 10px;
  align-items: center;
}

/* 表格样式 */
.library-content {
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  min-height: 400px;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.folder-icon {
  color: #f39c12;
  font-size: 18px;
}

.document-icon {
  color: #409eff;
  font-size: 18px;
}

.name-text {
  flex: 1;
}

.name-cell.clickable {
  cursor: pointer;
  user-select: none;
}

.name-cell.clickable:hover {
  color: #409eff;
}

.name-cell.clickable:hover .folder-icon {
  color: #409eff;
}

.breadcrumb-link {
  cursor: pointer;
  color: #409eff;
}

.breadcrumb-link:hover {
  text-decoration: underline;
}

/* 分类树容器 */
.category-tree-container {
  max-height: 500px;
  overflow-y: auto;
  padding: 10px 0;
}

.category-tree-node {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding: 8px 0;
  min-height: 40px;
}

.category-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  margin-right: 15px;
}

.category-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 4px;
}

.category-description {
  font-size: 12px;
  color: #909399;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.category-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.action-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

/* 加载和空状态 */
.loading-container {
  padding: 40px;
}

.empty-state {
  padding: 80px 0;
  text-align: center;
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
  padding: 20px 20px 10px;
  border-bottom: 1px solid #f0f0f0;
}

.bank-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
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
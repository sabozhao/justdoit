import { defineStore } from 'pinia'
import { questionBankAPI, wrongQuestionAPI, examResultAPI } from '../api'
import { ElMessage } from 'element-plus'

export const useExamStore = defineStore('exam', {
  state: () => ({
    questionBanks: [],
    wrongQuestions: [],
    currentExam: null,
    examResults: [],
    loading: false
  }),

  getters: {
    getQuestionBankById: (state) => (id) => {
      return state.questionBanks.find(bank => bank.id === id)
    },
    
    getWrongQuestionsByBank: (state) => (bankId) => {
      return state.wrongQuestions.filter(q => q.bank_id === bankId)
    }
  },

  actions: {
    // 加载所有题库
    async loadQuestionBanks() {
      try {
        this.loading = true
        console.log('开始加载题库...')
        const result = await questionBankAPI.getAll()
        console.log('题库加载结果:', result)
        this.questionBanks = result || []
      } catch (error) {
        console.error('加载题库失败:', error)
        ElMessage.error('加载题库失败: ' + error.message)
        this.questionBanks = []
      } finally {
        this.loading = false
      }
    },

    // 获取题库详情（包含题目）
    async getQuestionBankWithQuestions(id) {
      try {
        return await questionBankAPI.getById(id)
      } catch (error) {
        ElMessage.error('获取题库详情失败: ' + error.message)
        throw error
      }
    },

    // 添加题库
    async addQuestionBank(bank) {
      try {
        this.loading = true
        const result = await questionBankAPI.create(bank)
        await this.loadQuestionBanks() // 重新加载题库列表
        ElMessage.success('题库创建成功')
        return result
      } catch (error) {
        ElMessage.error('创建题库失败: ' + error.message)
        throw error
      } finally {
        this.loading = false
      }
    },

    // 上传题库文件
    async uploadQuestionBankFile(bankId, formData) {
      try {
        this.loading = true
        const result = await questionBankAPI.uploadFile(bankId, formData)
        await this.loadQuestionBanks() // 重新加载题库列表
        ElMessage.success(result.message || '题库上传成功')
        return result
      } catch (error) {
        ElMessage.error('上传题库文件失败: ' + error.message)
        throw error
      } finally {
        this.loading = false
      }
    },

    // 删除题库
    async deleteQuestionBank(id) {
      try {
        this.loading = true
        await questionBankAPI.delete(id)
        await this.loadQuestionBanks() // 重新加载题库列表
        await this.loadWrongQuestions() // 重新加载错题（可能有相关错题被删除）
        ElMessage.success('题库删除成功')
      } catch (error) {
        ElMessage.error('删除题库失败: ' + error.message)
        throw error
      } finally {
        this.loading = false
      }
    },

    // 获取单个题库详情（从API）
    async getQuestionBankDetails(id) {
      try {
        console.log('正在获取题库详情，ID:', id)
        const bankDetails = await questionBankAPI.getById(id)
        console.log('获取到题库详情:', bankDetails)
        return bankDetails
      } catch (error) {
        console.error('获取题库详情失败:', error)
        ElMessage.error('获取题库详情失败: ' + error.message)
        throw error
      }
    },

    // 加载所有错题
    async loadWrongQuestions() {
      try {
        const result = await wrongQuestionAPI.getAll()
        this.wrongQuestions = result || []
      } catch (error) {
        ElMessage.error('加载错题失败: ' + error.message)
        console.error('Failed to load wrong questions:', error)
        this.wrongQuestions = []
      }
    },

    // 添加错题（支持多选）
    async addWrongQuestion(question, bankId) {
      try {
        const wrongQuestionData = {
          bankId,
          questionId: question.id || Date.now().toString(),
          question: question.question,
          options: question.options,
          answer: question.answer, // 支持数组（多选）
          is_multiple: question.is_multiple || false, // 传递题目类型
          explanation: question.explanation
        }
        
        await wrongQuestionAPI.add(wrongQuestionData)
        await this.loadWrongQuestions() // 重新加载错题列表
      } catch (error) {
        // 如果错题已存在，不显示错误信息
        if (!error.message.includes('already exists')) {
          ElMessage.error('添加错题失败: ' + error.message)
        }
        console.error('Failed to add wrong question:', error)
      }
    },

    // 从错题库移除
    async removeWrongQuestion(questionId) {
      try {
        await wrongQuestionAPI.remove(questionId)
        await this.loadWrongQuestions() // 重新加载错题列表
      } catch (error) {
        ElMessage.error('移除错题失败: ' + error.message)
        throw error
      }
    },

    // 清空所有错题
    async clearAllWrongQuestions() {
      try {
        await wrongQuestionAPI.clear()
        await this.loadWrongQuestions() // 重新加载错题列表
        ElMessage.success('错题库已清空')
      } catch (error) {
        ElMessage.error('清空错题库失败: ' + error.message)
        throw error
      }
    },

    // 设置当前考试
    setCurrentExam(exam) {
      this.currentExam = exam
    },

    // 保存考试结果
    async saveExamResult(result) {
      try {
        await examResultAPI.save(result)
      } catch (error) {
        ElMessage.error('保存考试结果失败: ' + error.message)
        console.error('Failed to save exam result:', error)
      }
    },

    // 获取统计信息
    async getExamStats() {
      try {
        return await examResultAPI.getStats()
      } catch (error) {
        ElMessage.error('获取统计信息失败: ' + error.message)
        console.error('Failed to get exam stats:', error)
        return null
      }
    },

    // 获取题库题目
    async getQuestions(bankId) {
      try {
        return await questionBankAPI.getQuestions(bankId)
      } catch (error) {
        ElMessage.error('获取题目失败: ' + error.message)
        console.error('Failed to get questions:', error)
        return []
      }
    },

    // 添加题目
    async addQuestion(questionData) {
      try {
        return await questionBankAPI.addQuestion(questionData)
      } catch (error) {
        ElMessage.error('添加题目失败: ' + error.message)
        console.error('Failed to add question:', error)
        throw error
      }
    },

    // 更新题目
    async updateQuestion(questionId, questionData) {
      try {
        return await questionBankAPI.updateQuestion(questionId, questionData)
      } catch (error) {
        ElMessage.error('更新题目失败: ' + error.message)
        console.error('Failed to update question:', error)
        throw error
      }
    },

    // 删除题目
    async deleteQuestion(questionId) {
      try {
        await questionBankAPI.deleteQuestion(questionId)
        ElMessage.success('题目删除成功')
      } catch (error) {
        ElMessage.error('删除题目失败: ' + error.message)
        console.error('Failed to delete question:', error)
        throw error
      }
    }
  }
})
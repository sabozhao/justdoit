import { defineStore } from 'pinia'
import { questionBankAPI, publicQuestionBankAPI, wrongQuestionAPI, examResultAPI, categoryAPI } from '../api'
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
    // 加载所有题库（支持type和category_id参数）
    async loadQuestionBanks(params = {}) {
      try {
        this.loading = true
        console.log('开始加载题库...', params)
        const result = await questionBankAPI.getAll(params)
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

    // 获取题库详情（包含题目，自动判断个人题库或共享题库）
    async getQuestionBankWithQuestions(id) {
      try {
        // 先尝试个人题库
        try {
          return await questionBankAPI.getById(id)
        } catch (personalError) {
          // 如果个人题库不存在（404），尝试共享题库
          if (personalError.message && personalError.message.includes('404')) {
            console.log('个人题库不存在，尝试获取共享题库...')
            try {
              return await publicQuestionBankAPI.getById(id)
            } catch (publicError) {
              console.error('共享题库也不存在:', publicError)
              throw publicError
            }
          } else {
            // 其他错误直接抛出
            throw personalError
          }
        }
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

    // 更新题库
    async updateQuestionBank(id, bank) {
      try {
        this.loading = true
        const result = await questionBankAPI.update(id, bank)
        await this.loadQuestionBanks() // 重新加载题库列表
        ElMessage.success('题库更新成功')
        return result
      } catch (error) {
        ElMessage.error('更新题库失败: ' + error.message)
        throw error
      } finally {
        this.loading = false
      }
    },

    // 上传题库文件（支持创建新题库，bankId为'new'时创建新题库）
    async uploadQuestionBankFile(bankId, formData) {
      try {
        this.loading = true
        const result = await questionBankAPI.uploadFile(bankId, formData)
        // 注意：这里不自动重新加载，由调用方决定是否重新加载（因为可能需要保持筛选条件）
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
    async deleteQuestionBank(id, silent = false) {
      try {
        this.loading = true
        await questionBankAPI.delete(id)
        await this.loadQuestionBanks() // 重新加载题库列表
        await this.loadWrongQuestions() // 重新加载错题（可能有相关错题被删除）
        if (!silent) {
          ElMessage.success('题库删除成功')
        }
      } catch (error) {
        if (!silent) {
          ElMessage.error('删除题库失败: ' + error.message)
        }
        throw error
      } finally {
        this.loading = false
      }
    },

    // 获取单个题库详情（从API，自动判断个人题库或共享题库）
    async getQuestionBankDetails(id) {
      try {
        console.log('正在获取题库详情，ID:', id)
        // 先尝试个人题库
        try {
          const bankDetails = await questionBankAPI.getById(id)
          console.log('获取到个人题库详情:', bankDetails)
          return bankDetails
        } catch (personalError) {
          // 如果个人题库不存在（404或NotFound），尝试共享题库
          const errorMsg = personalError.message || personalError.toString() || ''
          if (errorMsg.includes('404') || errorMsg.includes('NotFound') || errorMsg.includes('not found') || errorMsg.includes('不存在')) {
            console.log('个人题库不存在，尝试获取共享题库...')
            try {
              const bankDetails = await publicQuestionBankAPI.getById(id)
              console.log('获取到共享题库详情:', bankDetails)
              return bankDetails
            } catch (publicError) {
              console.error('共享题库也不存在:', publicError)
              throw publicError
            }
          } else {
            // 其他错误直接抛出
            throw personalError
          }
        }
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
          type: question.type || 'choice', // 传递题目类型（选择题或判断题）
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

    // 批量添加错题
    async addWrongQuestionsBatch(questions, bankId) {
      try {
        const wrongQuestionsData = questions.map(question => ({
          bankId,
          questionId: question.id || Date.now().toString(),
          question: question.question,
          options: question.options,
          answer: question.answer, // 支持数组（多选）
          is_multiple: question.is_multiple || false, // 传递题目类型
          type: question.type || 'choice', // 传递题目类型（选择题或判断题）
          explanation: question.explanation || ''
        }))
        
        const result = await wrongQuestionAPI.addBatch({ questions: wrongQuestionsData })
        await this.loadWrongQuestions() // 重新加载错题列表
        return result
      } catch (error) {
        ElMessage.error('批量添加错题失败: ' + error.message)
        console.error('Failed to add wrong questions batch:', error)
        throw error
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

    // 获取题库题目（自动判断个人题库或共享题库）
    async getQuestions(bankId) {
      try {
        // 先尝试个人题库
        try {
          return await questionBankAPI.getQuestions(bankId)
        } catch (personalError) {
          // 如果个人题库不存在（404），尝试共享题库
          if (personalError.message && personalError.message.includes('404')) {
            console.log('个人题库不存在，尝试获取共享题库题目...')
            try {
              return await publicQuestionBankAPI.getQuestions(bankId)
            } catch (publicError) {
              console.error('共享题库也不存在:', publicError)
              throw publicError
            }
          } else {
            // 其他错误直接抛出
            throw personalError
          }
        }
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
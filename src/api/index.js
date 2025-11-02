// API 基础配置 - 根据环境动态配置
// 开发环境使用 localhost，生产环境使用相对路径（由打包脚本替换）
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || (import.meta.env.DEV ? 'http://localhost:3005/api' : '/api');

// 打印当前API配置（仅在开发环境）
if (import.meta.env.DEV) {
  console.log('🌐 API Base URL:', API_BASE_URL);
  console.log('🔧 Environment:', import.meta.env.MODE);
}

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 通用请求函数
async function request(url, options = {}) {
  try {
    console.log(`API请求: ${API_BASE_URL}${url}`, options);
    
    // 添加认证头
    const headers = {
      'Content-Type': 'application/json',
      ...options.headers,
    }
    
    const token = getToken()
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }
    
    const response = await fetch(`${API_BASE_URL}${url}`, {
      headers,
      ...options,
    });

    console.log(`API响应状态: ${response.status}`);

    if (!response.ok) {
      let errorMessage = `HTTP ${response.status}`;
      try {
        const error = await response.json();
        errorMessage = error.error || errorMessage;
      } catch {
        errorMessage = `请求失败: ${response.statusText}`;
      }
      throw new Error(errorMessage);
    }

    const result = await response.json();
    console.log('API响应数据:', result);
    return result;
  } catch (error) {
    console.error('API请求失败:', error);
    if (error.name === 'TypeError' && error.message.includes('fetch')) {
      throw new Error('无法连接到服务器，请检查后端服务是否启动');
    }
    throw error;
  }
}

// 题库相关API
export const questionBankAPI = {
  // 获取所有题库
  getAll: () => request('/question-banks'),
  
  // 获取单个题库详情
  getById: (id) => request(`/question-banks/${id}`),
  
  // 创建题库
  create: (data) => request('/question-banks', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  
  // 上传题库文件
  uploadFile: async (bankId, formData) => {
    try {
      const token = getToken()
      const headers = {}
      
      if (token) {
        headers.Authorization = `Bearer ${token}`
      }
      
      const response = await fetch(`${API_BASE_URL}/question-banks/${bankId}/upload`, {
        method: 'POST',
        headers,
        body: formData,
      })

      if (!response.ok) {
        let errorMessage = `HTTP ${response.status}`;
        try {
          const error = await response.json();
          errorMessage = error.error || errorMessage;
        } catch {
          errorMessage = `请求失败: ${response.statusText}`;
        }
        throw new Error(errorMessage);
      }

      return await response.json();
    } catch (error) {
      console.error('文件上传失败:', error);
      throw error;
    }
  },
  
  // 删除题库
  delete: (id) => request(`/question-banks/${id}`, {
    method: 'DELETE',
  }),

  // 获取题库题目
  getQuestions: (bankId) => request(`/question-banks/${bankId}/questions`),
  
  // 添加题目到题库
  addQuestion: (data) => request('/questions', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  
  // 更新题目
  updateQuestion: (id, data) => request(`/questions/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  
  // 删除题目
  deleteQuestion: (id) => request(`/questions/${id}`, {
    method: 'DELETE',
  }),
};

// 错题相关API
export const wrongQuestionAPI = {
  // 获取所有错题
  getAll: () => request('/wrong-questions'),
  
  // 添加错题
  add: (data) => request('/wrong-questions', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  
  // 删除错题
  remove: (id) => request(`/wrong-questions/${id}`, {
    method: 'DELETE',
  }),
  
  // 清空所有错题
  clear: () => request('/wrong-questions', {
    method: 'DELETE',
  }),
};

// 考试结果相关API
export const examResultAPI = {
  // 保存考试结果
  save: (data) => request('/exam-results', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  
  // 获取统计信息
  getStats: () => request('/exam-results/stats'),
};

// 用户认证相关API
export const authAPI = {
  // 用户注册
  register: (data) => request('/auth/register', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  
  // 用户登录
  login: (data) => request('/auth/login', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  
  // 获取当前用户信息
  getCurrentUser: () => request('/auth/me'),
};

// 管理员相关API
export const adminAPI = {
  // 获取所有用户
  getUsers: () => request('/admin/users'),
  
  // 更新用户信息
  updateUser: (id, data) => request(`/admin/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  }),
  
  // 删除用户
  deleteUser: (id) => request(`/admin/users/${id}`, {
    method: 'DELETE',
  }),
  
  // 获取所有题库
  getQuestionBanks: () => request('/admin/question-banks'),
  
  // 删除题库
  deleteQuestionBank: (id) => request(`/admin/question-banks/${id}`, {
    method: 'DELETE',
  }),
  
  // 获取系统统计
  getStats: () => request('/admin/stats'),
  
  // 获取系统设置
  getSettings: () => request('/admin/settings'),
  
  // 更新系统设置
  updateSettings: (data) => request('/admin/settings', {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  
  // 获取题库题目
  getQuestions: (bankId) => request(`/question-banks/${bankId}/questions`),
  
  // 更新题库信息
  updateQuestionBank: (id, data) => request(`/question-banks/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  
  // 创建题目
  createQuestion: (data) => request('/questions', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  
  // 更新题目
  updateQuestion: (id, data) => request(`/questions/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  
  // 删除题目
  deleteQuestion: (id) => request(`/questions/${id}`, {
    method: 'DELETE',
  }),
};
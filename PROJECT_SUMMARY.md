# 刷题网站项目总结

## 🎯 项目概述

这是一个功能完整的在线刷题网站，支持用户上传自定义题库、在线考试、错题收集等功能。项目提供了两种后端实现方案：Node.js 和 Go 语言。

## 📋 功能特性

### ✅ 核心功能
- **用户系统**: 注册/登录/JWT身份验证
- **题库管理**: 创建、上传、删除题库
- **多格式支持**: JSON、Excel (.xlsx/.xls)、CSV 文件上传
- **在线考试**: 实时答题、进度导航、自动评分
- **错题收集**: 自动收集错题、专项练习
- **数据统计**: 考试结果统计、个人数据分析

### 🎨 界面特色
- 现代化 UI 设计
- 响应式布局（每行最多5个题库卡片）
- 美观的渐变背景
- 流畅的动画效果
- 直观的操作界面

## 🏗️ 技术架构

### 前端技术栈
- **Vue 3** - 渐进式JavaScript框架
- **Element Plus** - Vue 3 UI组件库
- **Pinia** - Vue 状态管理
- **Vue Router** - 路由管理
- **Vite** - 构建工具

### 后端技术栈（两种实现）

#### Node.js 版本 (server/)
- **Express** - Web框架
- **SQLite3** - 数据库
- **JWT** - 身份认证
- **bcryptjs** - 密码加密
- **multer** - 文件上传
- **xlsx** - Excel文件处理

#### Go 版本 (go-server/)
- **Gin** - Web框架
- **SQLite** - 数据库
- **JWT** - 身份认证
- **bcrypt** - 密码加密
- **xlsx** - Excel文件处理

## 📁 项目结构

```
exam_test1/
├── src/                          # 前端源码
│   ├── components/               # Vue组件
│   ├── views/                    # 页面组件
│   │   ├── Home.vue             # 首页
│   │   ├── Library.vue          # 题库管理
│   │   ├── Practice.vue         # 练习选择
│   │   ├── Exam.vue             # 考试页面
│   │   ├── WrongQuestions.vue   # 错题管理
│   │   └── Login.vue            # 登录注册
│   ├── stores/                   # Pinia状态管理
│   │   ├── auth.js              # 用户认证
│   │   └── exam.js              # 考试数据
│   ├── api/                      # API接口
│   │   └── index.js             # API客户端
│   └── router/                   # 路由配置
│       └── index.js             # 路由定义
├── server/                       # Node.js后端
│   ├── server.js                # 主服务器文件
│   ├── package.json             # 依赖配置
│   └── uploads/                 # 文件上传目录
├── go-server/                    # Go后端
│   ├── main.go                  # 主文件和数据结构
│   ├── routes.go                # 路由配置
│   ├── handlers.go              # 题库处理函数
│   ├── wrong_questions.go       # 错题处理函数
│   ├── exam_results.go          # 考试结果处理
│   ├── go.mod                   # Go模块文件
│   ├── start.sh                 # 启动脚本
│   ├── install-go.sh            # Go环境安装脚本
│   └── README.md                # Go版本说明
├── package.json                  # 前端依赖配置
└── PROJECT_SUMMARY.md           # 项目总结文档
```

## 🚀 快速开始

### 方案一：使用 Node.js 后端

1. **启动后端服务**
```bash
cd server
npm install
npm start
```

2. **启动前端服务**
```bash
npm install
npm run dev
```

### 方案二：使用 Go 后端

1. **安装 Go 环境**（如果未安装）
```bash
cd go-server
chmod +x install-go.sh
./install-go.sh
source ~/.zshrc  # 重新加载环境变量
```

2. **启动 Go 后端服务**
```bash
cd go-server
chmod +x start.sh
./start.sh
```

3. **启动前端服务**
```bash
npm install
npm run dev
```

## 📊 数据库设计

### 用户表 (users)
- id: 用户ID
- username: 用户名
- password: 加密密码
- email: 邮箱
- created_at: 创建时间

### 题库表 (question_banks)
- id: 题库ID
- user_id: 用户ID（外键）
- name: 题库名称
- description: 题库描述
- created_at: 创建时间

### 题目表 (questions)
- id: 题目ID
- bank_id: 题库ID（外键）
- question: 题目内容
- options: 选项（JSON格式）
- answer: 正确答案索引
- explanation: 答案解析

### 错题表 (wrong_questions)
- id: 错题ID
- user_id: 用户ID（外键）
- bank_id: 题库ID（外键）
- question_id: 题目ID
- question: 题目内容
- options: 选项（JSON格式）
- answer: 正确答案索引
- explanation: 答案解析
- added_at: 添加时间

### 考试结果表 (exam_results)
- id: 结果ID
- user_id: 用户ID（外键）
- bank_id: 题库ID（外键）
- score: 得分
- correct_count: 正确题数
- wrong_count: 错误题数
- total_questions: 总题数
- total_time: 考试用时
- created_at: 创建时间

## 📝 文件格式支持

### Excel/CSV 格式要求

| 中文列名 | 英文列名 | 是否必需 | 说明 |
|---------|---------|---------|------|
| 题目 | question/Question | 必需 | 题目内容 |
| 选项A | A/optionA | 必需 | 选项A内容 |
| 选项B | B/optionB | 必需 | 选项B内容 |
| 选项C | C/optionC | 可选 | 选项C内容 |
| 选项D | D/optionD | 可选 | 选项D内容 |
| 正确答案 | answer/Answer | 必需 | A/B/C/D 或 1/2/3/4 |
| 解析 | explanation | 可选 | 答案解析 |

### JSON 格式示例

```json
{
  "questions": [
    {
      "question": "题目内容",
      "options": ["选项A", "选项B", "选项C", "选项D"],
      "answer": 0,
      "explanation": "答案解析（可选）"
    }
  ]
}
```

## 🔧 API 接口

### 认证相关
- `POST /api/auth/register` - 用户注册
- `POST /api/auth/login` - 用户登录
- `GET /api/auth/me` - 获取当前用户信息

### 题库管理
- `GET /api/question-banks` - 获取题库列表
- `GET /api/question-banks/:id` - 获取题库详情
- `POST /api/question-banks` - 创建题库
- `POST /api/question-banks/upload` - 上传题库文件
- `DELETE /api/question-banks/:id` - 删除题库

### 错题管理
- `GET /api/wrong-questions` - 获取错题列表
- `POST /api/wrong-questions` - 添加错题
- `DELETE /api/wrong-questions/:id` - 删除错题
- `DELETE /api/wrong-questions` - 清空所有错题

### 考试结果
- `POST /api/exam-results` - 保存考试结果
- `GET /api/exam-results/stats` - 获取统计信息

## 🎯 核心功能实现

### 1. 用户认证系统
- JWT Token 身份验证
- bcrypt 密码加密
- 路由守卫保护
- 用户数据隔离

### 2. 文件上传解析
- 支持多种文件格式
- 智能列名识别
- 数据验证和错误处理
- 文件临时存储和清理

### 3. 考试系统
- 实时答题界面
- 进度导航条
- 题目跳转功能
- 自动评分算法

### 4. 错题收集
- 自动错题收集
- 错题专项练习
- 错题管理功能

## 🔒 安全特性

- JWT Token 认证
- 密码 bcrypt 加密
- SQL 注入防护
- CORS 跨域配置
- 文件类型验证
- 用户数据隔离

## 🚀 部署建议

### 开发环境
- 前端：Vite 开发服务器
- 后端：Node.js/Go 本地服务器
- 数据库：SQLite 文件数据库

### 生产环境
- 前端：静态文件部署（Nginx）
- 后端：PM2/Docker 容器部署
- 数据库：PostgreSQL/MySQL
- 反向代理：Nginx
- HTTPS 证书配置

## 📈 性能优化

- 前端代码分割
- 图片懒加载
- API 请求缓存
- 数据库索引优化
- 文件上传大小限制

## 🎉 项目亮点

1. **双后端实现**: 提供 Node.js 和 Go 两种后端选择
2. **多格式支持**: 智能解析 JSON、Excel、CSV 文件
3. **现代化界面**: Vue 3 + Element Plus 美观UI
4. **完整用户系统**: 注册登录、数据隔离
5. **智能错题收集**: 自动收集错题，支持专项练习
6. **进度导航**: 考试过程中可视化进度和快速跳转
7. **响应式设计**: 适配不同屏幕尺寸
8. **安全可靠**: JWT认证、密码加密、数据验证

## 🔮 未来扩展

- 题目分类标签系统
- 多人在线考试
- 题目评论和讨论
- 学习进度分析
- 移动端适配
- 题目推荐算法
- 成绩排行榜
- 导出学习报告

---

这个刷题网站项目实现了完整的在线学习平台功能，代码结构清晰，功能丰富，可以作为学习或商业项目的基础进行进一步开发。
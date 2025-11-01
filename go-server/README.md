# Go 后端服务器

这是一个用 Go 语言实现的刷题网站后端服务，提供完整的 RESTful API。

## 功能特性

- 用户认证系统（注册/登录/JWT）
- 题库管理（CRUD操作）
- 多格式文件上传（JSON/Excel/CSV）
- 错题收集和管理
- 考试结果统计
- SQLite 数据库存储
- CORS 跨域支持

## 技术栈

- **Go 1.21+**
- **Gin** - Web 框架
- **SQLite** - 数据库
- **JWT** - 身份认证
- **bcrypt** - 密码加密
- **xlsx** - Excel 文件处理

## 安装和运行

### 1. 安装 Go

```bash
# macOS (使用 Homebrew)
brew install go

# 或者从官网下载安装包
# https://golang.org/dl/
```

### 2. 安装依赖

```bash
cd go-server
go mod tidy
```

### 3. 运行服务器

```bash
go run *.go
```

服务器将在 `http://localhost:3004` 启动。

### 4. 构建可执行文件

```bash
go build -o exam-server *.go
./exam-server
```

## API 接口

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

## 文件格式支持

### Excel/CSV 格式要求

支持以下列名（中英文均可）：

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

## 数据库

使用 SQLite 数据库，文件名为 `exam.db`，包含以下表：

- `users` - 用户表
- `question_banks` - 题库表
- `questions` - 题目表
- `wrong_questions` - 错题表
- `exam_results` - 考试结果表

## 环境变量

可以通过环境变量配置：

- `PORT` - 服务器端口（默认 3004）
- `JWT_SECRET` - JWT 密钥（生产环境请修改）
- `DB_PATH` - 数据库文件路径（默认 ./exam.db）

## 开发说明

项目结构：

```
go-server/
├── main.go           # 主文件和数据结构
├── routes.go         # 路由配置
├── handlers.go       # 题库相关处理函数
├── wrong_questions.go # 错题相关处理函数
├── exam_results.go   # 考试结果处理函数
├── go.mod           # Go 模块文件
└── README.md        # 说明文档
```

## 部署

### 使用 Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o exam-server *.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/exam-server .
EXPOSE 3004
CMD ["./exam-server"]
```

### 直接部署

```bash
# 构建
go build -o exam-server *.go

# 运行
./exam-server
```

## 注意事项

1. 生产环境请修改 JWT 密钥
2. 建议使用反向代理（如 Nginx）
3. 定期备份 SQLite 数据库文件
4. 上传文件大小限制可在代码中调整
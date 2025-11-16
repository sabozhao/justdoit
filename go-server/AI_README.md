# AI 题目识别功能说明（腾讯云混元大模型）

## 功能概述

系统已集成腾讯云混元大模型（免费7B版本），支持自动识别 PDF 和 DOC/DOCX 文件中的选择题，无需固定格式。

## 支持的文件格式

- **PDF** (.pdf) - 使用 AI 自动识别
- **DOC/DOCX** (.doc, .docx) - 使用 AI 自动识别
- **JSON** (.json) - 传统解析方式
- **Excel** (.xlsx, .xls) - 传统解析方式
- **CSV** (.csv) - 传统解析方式

## 配置腾讯云 API

### 1. 获取 SecretId 和 SecretKey

1. 访问 [腾讯云控制台](https://console.cloud.tencent.com/)
2. 登录或注册账号
3. 进入 [API密钥管理](https://console.cloud.tencent.com/cam/capi)
4. 点击"新建密钥"，或使用已有的密钥
5. 复制 **SecretId** 和 **SecretKey**
   - ⚠️ **重要**：SecretKey 只显示一次，请妥善保管

### 2. 开通混元大模型服务

1. 访问 [腾讯云混元大模型控制台](https://console.cloud.tencent.com/hunyuan)
2. 开通服务（通常有免费额度）
3. 查看免费额度和使用限制

### 3. 配置环境变量

在 `go-server/.env` 文件中添加：

```env
# 腾讯云混元大模型API配置
TENCENT_SECRET_ID=your-secret-id
TENCENT_SECRET_KEY=your-secret-key
TENCENT_REGION=ap-beijing
```

**参数说明**：
- `TENCENT_SECRET_ID`: 腾讯云API密钥ID（必填）
- `TENCENT_SECRET_KEY`: 腾讯云API密钥（必填）
- `TENCENT_REGION`: 地域（可选，默认：ap-beijing）
  - 可选值：ap-beijing（北京）、ap-shanghai（上海）、ap-guangzhou（广州）等

### 4. 重启服务

配置完成后，重启后端服务：

```bash
cd go-server
./exam-server
```

服务启动时会显示：
- ✅ "AI服务已初始化（腾讯云混元大模型）" - 表示配置成功
- ⚠️ "Warning: 腾讯云SecretId和SecretKey未配置，AI识别功能将被禁用" - 表示未配置

## 使用方法

1. 登录系统
2. 进入"管理题库"页面
3. 选择或创建一个题库
4. 点击"上传题目"
5. 选择 PDF 或 Word 文件
6. 系统会自动：
   - 提取文件中的文本
   - 使用腾讯云混元大模型识别选择题
   - 解析题目、选项、答案和解析
   - 保存到题库中

## AI 识别能力

腾讯云混元大模型可以识别各种格式的题目，包括：

- 标准格式：题目 + A. 选项 + B. 选项 + ... + 答案
- 非标准格式：各种自由格式的选择题
- 混合格式：同一文档中包含不同格式的题目
- 多语言：支持中文和英文题目

## 注意事项

1. **免费额度**: 腾讯云混元大模型提供免费额度，请查看控制台了解详情
2. **识别准确率**: 对于格式规范的题目，识别准确率较高；对于特殊格式，可能需要手动调整
3. **文件大小**: 建议单个文件不超过 10MB，确保文本提取完整
4. **识别限制**: 
   - 目前仅支持选择题
   - 答案必须是明确的（A/B/C/D 或 1/2/3/4）
   - 需要至少 2 个选项

## 故障排除

### 问题：提示 "AI服务未配置"
- **解决**: 检查 `.env` 文件中的 `TENCENT_SECRET_ID` 和 `TENCENT_SECRET_KEY` 是否正确配置

### 问题：提示 "AI识别失败"
- **原因**: API Key 无效、网络问题、API 配额用完或签名错误
- **解决**: 
  1. 检查 SecretId 和 SecretKey 是否正确
  2. 检查网络连接
  3. 查看腾讯云账户余额和配额
  4. 查看后端日志获取详细错误信息

### 问题：识别结果不准确
- **原因**: 题目格式特殊或文本提取不完整
- **解决**: 
  1. 尝试将文档转换为更规范的格式
  2. 对于复杂格式，建议使用 JSON 或 Excel 格式手动导入

## 技术实现

- **PDF 解析**: 使用 `github.com/gen2brain/go-fitz` (MuPDF)
- **DOCX 解析**: 直接解析 ZIP 压缩的 XML 文件
- **DOC 解析**: 使用 LibreOffice 命令行工具（需要安装 LibreOffice）
- **AI 服务**: 使用腾讯云混元大模型 API（免费7B版本）

## DOC 文件支持说明

系统支持 `.doc` 文件格式，但需要服务器安装 **LibreOffice**。

### 安装 LibreOffice

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install libreoffice
```

**CentOS/RHEL:**
```bash
sudo yum install libreoffice
# 或
sudo dnf install libreoffice
```

**macOS:**
```bash
brew install --cask libreoffice
```

**Windows:**
从 [LibreOffice 官网](https://www.libreoffice.org/download/) 下载安装程序。

### 验证安装

安装完成后，可以通过以下命令验证：
```bash
libreoffice --version
# 或
soffice --version
```

如果系统未安装 LibreOffice，上传 `.doc` 文件时会提示错误。建议：
1. 在服务器上安装 LibreOffice
2. 或者将 `.doc` 文件转换为 `.docx` 格式后再上传

## API 调用说明

系统使用腾讯云混元大模型的 ChatCompletions API，调用参数：
- **模型**: hunyuan-lite（混元精简版，免费）
- **最大Token**: 2000
- **温度**: 0.3（降低随机性，提高准确性）

## 获取API密钥链接

- [API密钥管理](https://console.cloud.tencent.com/cam/capi)
- [混元大模型控制台](https://console.cloud.tencent.com/hunyuan)
- [腾讯云文档中心](https://cloud.tencent.com/document/product/1729)

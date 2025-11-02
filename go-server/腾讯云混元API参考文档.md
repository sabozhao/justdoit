# 腾讯云混元大模型 API 参考文档

## 官方文档链接

### 1. 产品文档
- **主文档入口**: https://cloud.tencent.com/document/product/1729
- **API文档**: https://cloud.tencent.com/document/product/1729/98789
- **快速入门**: https://cloud.tencent.com/document/product/1729/98790

### 2. API 操作文档
- **ChatCompletions接口**: https://cloud.tencent.com/document/product/1729/98783
- **请求参数说明**: https://cloud.tencent.com/document/product/1729/98784
- **返回参数说明**: https://cloud.tencent.com/document/product/1729/98785

### 3. SDK 文档
- **Go SDK**: https://cloud.tencent.com/document/product/1729/98921
- **GitHub Go SDK**: https://github.com/tencentcloud/tencentcloud-sdk-go

### 4. 控制台和密钥管理
- **控制台**: https://console.cloud.tencent.com/hunyuan
- **API密钥管理**: https://console.cloud.tencent.com/cam/capi
- **服务开通**: https://console.cloud.tencent.com/hunyuan/service

## API 调用方式

### 方式一：使用官方 Go SDK（推荐）

```go
package main

import (
    "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
    "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
    hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
)

func callHunyuanAPI(secretId, secretKey, region string, prompt string) (string, error) {
    // 实例化认证对象
    credential := common.NewCredential(secretId, secretKey)
    
    // 实例化客户端配置对象
    cpf := profile.NewClientProfile()
    cpf.HttpProfile.Endpoint = "hunyuan.tencentcloudapi.com"
    
    // 实例化要请求产品的client对象
    client, _ := hunyuan.NewClient(credential, region, cpf)
    
    // 实例化请求对象
    request := hunyuan.NewChatCompletionsRequest()
    request.Model = common.StringPtr("hunyuan-lite")  // 混元精简版（免费）
    request.Messages = []*hunyuan.Message{
        {
            Role:    common.StringPtr("user"),
            Content: common.StringPtr(prompt),
        },
    }
    request.MaxTokens = common.Uint64Ptr(2000)
    request.Temperature = common.Float64Ptr(0.3)
    
    // 发送请求
    response, err := client.ChatCompletions(request)
    if err != nil {
        return "", err
    }
    
    // 处理响应
    if len(response.Response.Choices) > 0 {
        return *response.Response.Choices[0].Message.Content, nil
    }
    
    return "", fmt.Errorf("未返回有效结果")
}
```

### 方式二：直接使用 HTTP API（当前实现）

```http
POST https://hunyuan.tencentcloudapi.com/?Action=ChatCompletions&Version=2023-09-01
Content-Type: application/json
Authorization: TC3-HMAC-SHA256 Credential=...

{
  "Model": "hunyuan-lite",
  "Messages": [
    {
      "Role": "user",
      "Content": "你的提示词"
    }
  ],
  "MaxTokens": 2000,
  "Temperature": 0.3
}
```

## 当前代码实现分析

### 当前实现状态
- ✅ 使用 HTTP API 直接调用（无需额外依赖）
- ⚠️ 签名算法简化版本（需要完善）
- ✅ 支持基本的请求和响应处理

### 需要改进的地方
1. **签名算法**: 当前使用的是简化版签名，应该使用完整的 TC3-HMAC-SHA256 签名算法
2. **请求格式**: 应该使用 `Messages` 数组而不是 `Prompt` 字符串
3. **错误处理**: 需要更完善的错误处理机制

## 推荐方案

### 方案一：使用官方 SDK（最推荐）
- ✅ 自动处理签名算法
- ✅ 更稳定的API调用
- ✅ 更好的错误处理
- ⚠️ 需要添加依赖

### 方案二：完善当前 HTTP 实现
- ✅ 无额外依赖
- ⚠️ 需要实现完整的 TC3 签名算法
- ⚠️ 需要手动处理请求格式

## TC3 签名算法说明

腾讯云API使用 TC3-HMAC-SHA256 签名算法，具体步骤：

1. **规范请求**: 将HTTP请求规范化为固定格式的字符串
2. **签名请求**: 使用规范请求和SK生成签名串
3. **构建请求头**: 将签名信息放入Authorization请求头

详细算法说明：https://cloud.tencent.com/document/api/598/33140

## 快速测试

### 测试工具
- **API Explorer**: https://console.cloud.tencent.com/api/explorer?Product=hunyuan&Version=2023-09-01&Action=ChatCompletions
- **控制台测试**: https://console.cloud.tencent.com/hunyuan/chat

## 模型说明

### 可用的模型
- `hunyuan-lite`: 混元精简版（免费，适合一般场景）
- `hunyuan-pro`: 混元标准版（付费，性能更强）
- `hunyuan-turbo`: 混元加速版（付费，响应更快）

### 模型对比
| 模型 | 价格 | 适用场景 | 性能 |
|------|------|----------|------|
| hunyuan-lite | 免费 | 日常对话、简单任务 | 基础 |
| hunyuan-pro | 按量付费 | 复杂任务、高质量输出 | 优秀 |
| hunyuan-turbo | 按量付费 | 快速响应场景 | 快速 |

## 费用说明

- **hunyuan-lite**: 通常提供免费额度，超出后按量计费
- 具体价格：https://cloud.tencent.com/document/product/1729/104350

## 常见问题

### Q1: 如何获取免费额度？
A: 访问控制台开通服务，通常会赠送免费额度。

### Q2: API调用失败怎么办？
A: 
1. 检查 SecretId 和 SecretKey 是否正确
2. 检查网络连接
3. 查看错误码对照表：https://cloud.tencent.com/document/product/1729/98927

### Q3: 如何选择合适的模型？
A: 
- 日常使用选择 `hunyuan-lite`
- 需要高质量输出选择 `hunyuan-pro`
- 需要快速响应选择 `hunyuan-turbo`

## 相关资源

- **GitHub示例代码**: https://github.com/tencentcloud/hunyuan-sdk-go
- **开发者社区**: https://cloud.tencent.com/developer/tag/105
- **技术博客**: https://cloud.tencent.com/developer/column

## 更新日志

- 2025-11-01: 初始版本，使用 HTTP API 实现
- 待优化: 完善 TC3 签名算法或迁移到官方 SDK


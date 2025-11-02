// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// 直接测试AI识别的独立工具
// 使用方法: go run test_ai_direct.go
func testAIDirect() {
	fmt.Println("========================================")
	fmt.Println("AI识别功能直接测试工具")
	fmt.Println("========================================")
	fmt.Println()

	// 检查环境变量
	secretId := os.Getenv("TENCENT_SECRET_ID")
	secretKey := os.Getenv("TENCENT_SECRET_KEY")
	region := os.Getenv("TENCENT_REGION")
	if region == "" {
		region = "ap-beijing"
	}
	model := os.Getenv("TENCENT_MODEL")
	if model == "" {
		model = "hunyuan-lite"
	}

	if secretId == "" || secretKey == "" {
		fmt.Println("❌ 错误: 请先设置环境变量")
		fmt.Println("   export TENCENT_SECRET_ID=your_secret_id")
		fmt.Println("   export TENCENT_SECRET_KEY=your_secret_key")
		fmt.Println("   export TENCENT_MODEL=hunyuan-standard  # 可选，默认hunyuan-lite")
		fmt.Println()
		
		// 尝试从数据库配置读取
		fmt.Println("尝试从数据库配置读取...")
		initDB()
		if db != nil {
			configMutex.Lock()
			secretId = getSystemSetting("tencent_secret_id")
			secretKey = getSystemSetting("tencent_secret_key")
			region = getSystemSetting("tencent_region")
			if region == "" {
				region = "ap-beijing"
			}
			model = getSystemSetting("tencent_model")
			if model == "" {
				model = "hunyuan-lite"
			}
			configMutex.Unlock()
		}
		
		if secretId == "" || secretKey == "" {
			os.Exit(1)
		}
		fmt.Println("✓ 从数据库读取配置成功")
	}

	// 设置全局变量
	configMutex.Lock()
	tencentCloudSecretId = secretId
	tencentCloudSecretKey = secretKey
	tencentCloudRegion = region
	tencentCloudModel = model
	tencentCloudEndpoint = "hunyuan.tencentcloudapi.com"
	configMutex.Unlock()

	fmt.Printf("AI配置:\n")
	fmt.Printf("  区域: %s\n", region)
	fmt.Printf("  模型: %s\n", model)
	fmt.Printf("  端点: %s\n", tencentCloudEndpoint)
	fmt.Println()

	// 读取test.docx文件
	testFilePath := "test.docx"
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		testFilePath = "../test.docx"
	}
	
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		fmt.Printf("❌ 错误: 找不到test.docx文件\n")
		fmt.Printf("   当前目录: %s\n", testFilePath)
		os.Exit(1)
	}

	fmt.Printf("读取文件: %s\n", testFilePath)
	text, err := parseTestDocx(testFilePath)
	if err != nil {
		fmt.Printf("❌ 读取文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 提取文本成功，长度: %d 字符\n", len(text))
	fmt.Printf("\n文本内容预览（前500字符）:\n%s\n", testTruncateString(text, 500))
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("调用AI识别...")
	fmt.Println("========================================")
	fmt.Println()

	// 调用AI识别
	questions, err := recognizeQuestionsWithAI(text)
	if err != nil {
		fmt.Printf("❌ AI识别失败: %v\n", err)
		os.Exit(1)
	}

	// 显示结果
	if len(questions) == 0 {
		fmt.Println("⚠️  警告: 未识别到任何题目")
		os.Exit(1)
	}

	fmt.Printf("✅ 成功识别 %d 道题目:\n\n", len(questions))
	for i, q := range questions {
		fmt.Printf("题目 %d:\n", i+1)
		fmt.Printf("  题目: %s\n", q.Question)
		fmt.Printf("  选项数: %d\n", len(q.Options))
		for j, opt := range q.Options {
			fmt.Printf("    %s. %s\n", string(rune('A'+j)), opt)
		}
		fmt.Printf("  答案索引: %v\n", q.Answer)
		answerStr := ""
		for _, idx := range q.Answer {
			answerStr += string(rune('A'+idx)) + " "
		}
		fmt.Printf("  答案: %s\n", strings.TrimSpace(answerStr))
		fmt.Printf("  是否多选: %v\n", q.IsMultiple)
		if q.Explanation != "" {
			fmt.Printf("  解析: %s\n", q.Explanation)
		}
		fmt.Println()
	}

	// 验证JSON格式
	jsonData, err := json.MarshalIndent(questions, "", "  ")
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Printf("✅ JSON验证通过（长度: %d 字节）\n", len(jsonData))
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("JSON输出预览:")
	if len(jsonData) > 1000 {
		fmt.Printf("%s...\n", string(jsonData[:1000]))
	} else {
		fmt.Printf("%s\n", string(jsonData))
	}
}

// testTruncateString 截断字符串（测试工具专用）
func testTruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}


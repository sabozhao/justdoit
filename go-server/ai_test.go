package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestAIRecognition 测试AI识别功能
func TestAIRecognition(t *testing.T) {
	// 从环境变量读取配置（测试环境不需要数据库）
	configMutex.Lock()
	defer configMutex.Unlock()
	
	tencentCloudSecretId = os.Getenv("TENCENT_SECRET_ID")
	tencentCloudSecretKey = os.Getenv("TENCENT_SECRET_KEY")
	tencentCloudRegion = os.Getenv("TENCENT_REGION")
	if tencentCloudRegion == "" {
		tencentCloudRegion = "ap-beijing"
	}
	tencentCloudModel = os.Getenv("TENCENT_MODEL")
	if tencentCloudModel == "" {
		tencentCloudModel = "hunyuan-lite"
	}
	tencentCloudEndpoint = "hunyuan.tencentcloudapi.com"
	
	if tencentCloudSecretId == "" || tencentCloudSecretKey == "" {
		t.Skip("跳过测试：未配置腾讯云AI密钥（需要设置TENCENT_SECRET_ID和TENCENT_SECRET_KEY）")
	}

	// 读取test.docx文件
	testFilePath := "../test.docx"
	text, err := parseTestDocxForTest(testFilePath)
	if err != nil {
		t.Fatalf("读取test.docx失败: %v", err)
	}

	fmt.Printf("========================================\n")
	fmt.Printf("测试AI识别功能\n")
	fmt.Printf("========================================\n")
	fmt.Printf("\n提取的文本内容（前500字符）:\n%s\n", truncateString(text, 500))
	fmt.Printf("\n完整文本长度: %d 字符\n", len(text))
	fmt.Printf("\n========================================\n\n")

	// 调用AI识别
	questions, err := recognizeQuestionsWithAI(text)
	if err != nil {
		t.Fatalf("AI识别失败: %v", err)
	}

	// 验证结果
	if len(questions) == 0 {
		t.Error("未识别到任何题目")
		return
	}

	fmt.Printf("成功识别 %d 道题目:\n\n", len(questions))
	for i, q := range questions {
		fmt.Printf("题目 %d:\n", i+1)
		fmt.Printf("  题目: %s\n", q.Question)
		fmt.Printf("  选项数: %d\n", len(q.Options))
		for j, opt := range q.Options {
			fmt.Printf("    %s. %s\n", string(rune('A'+j)), opt)
		}
		fmt.Printf("  答案: %v\n", q.Answer)
		fmt.Printf("  是否多选: %v\n", q.IsMultiple)
		if q.Explanation != "" {
			fmt.Printf("  解析: %s\n", q.Explanation)
		}
		fmt.Println()
	}

	// 验证JSON格式（尝试序列化）
	jsonData, err := json.MarshalIndent(questions, "", "  ")
	if err != nil {
		t.Errorf("序列化题目为JSON失败: %v", err)
		return
	}

	fmt.Printf("========================================\n")
	fmt.Printf("JSON验证通过（长度: %d 字节）\n", len(jsonData))
	fmt.Printf("========================================\n")
}

// parseTestDocxForTest 解析test.docx文件并提取文本（测试专用）
func parseTestDocxForTest(filePath string) (string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在: %s", filePath)
	}

	// 使用debug_api.go中的parseTestDocx函数（已存在）
	return parseTestDocx(filePath)
}

// TestAIPromptOnly 仅测试提示词，不调用API（用于优化提示词）
func TestAIPromptOnly(t *testing.T) {
	// 读取test.docx文件
	testFilePath := "../test.docx"
	text, err := parseTestDocxForTest(testFilePath)
	if err != nil {
		t.Fatalf("读取test.docx失败: %v", err)
	}

	// 构建提示词（复制自ai_service.go）
	prompt := buildPrompt(text)

	fmt.Printf("========================================\n")
	fmt.Printf("测试提示词（不调用API）\n")
	fmt.Printf("========================================\n")
	fmt.Printf("\n提取的文本长度: %d 字符\n", len(text))
	fmt.Printf("\n提示词长度: %d 字符\n", len(prompt))
	fmt.Printf("\n提示词内容:\n%s\n", prompt)
	fmt.Printf("\n========================================\n")
}

// buildPrompt 构建AI提示词（从ai_service.go复制，便于测试和优化）
func buildPrompt(text string) string {
	return "你是一个专业的题目解析助手。请从以下文本中识别出所有选择题，并严格按照JSON格式返回。\n\n" +
		"【核心要求 - 必须严格遵守】\n" +
		"1. 你的回复必须是且只能是纯JSON格式，不能包含任何其他内容\n" +
		"2. 禁止使用markdown代码块标记（如```json或```），直接返回JSON对象\n" +
		"3. 禁止在JSON前后添加任何文字说明、注释、解释或标点符号\n" +
		"4. 禁止在JSON内部添加注释或说明性文字\n" +
		"5. 你的整个回复必须能够直接被JSON.parse()或json.Unmarshal()解析成功\n" +
		"6. 回复必须以{开始，以}结束，中间不能有任何非JSON内容\n\n" +
		"【JSON格式规范 - 严格遵守】\n" +
		"1. 所有字符串中的特殊字符必须正确转义：\n" +
		"   - 双引号 \" 必须转义为 \\\"\n" +
		"   - 反斜杠 \\ 必须转义为 \\\\\n" +
		"   - 换行符必须转义为 \\\\n\n" +
		"   - 制表符必须转义为 \\\\t\n" +
		"2. 所有字符串值必须用双引号包裹，不能用单引号\n" +
		"3. 数组和对象必须正确闭合，括号和花括号必须匹配\n" +
		"4. 数组元素之间用逗号分隔，最后一个元素后不能有逗号\n" +
		"5. JSON对象的所有键必须用双引号包裹\n" +
		"6. 每个字段之间必须用逗号分隔（最后一个字段除外）\n\n" +
		"【题目识别要求】\n" +
		"1. 识别所有选择题（包括单选题和多选题）\n" +
		"2. 每个题目包含：题目内容、选项（至少2个，最多10个）、正确答案、解析（如果有）\n" +
		"3. 如果文本中没有找到题目，返回 {\"questions\": []}\n" +
		"4. 支持各种格式的题目，不限于固定格式\n\n" +
		"【选项处理要求（重要）】\n" +
		"1. 选项可能是A. 选项内容、B. 选项内容，或者直接是选项内容\n" +
		"2. 如果选项前有字母前缀（如\"A.\"、\"A \"、\"A-\"），请去除前缀，只保留选项内容\n" +
		"3. 选项中的特殊字符必须正确转义\n" +
		"4. 【特别注意】不要把正确答案的内容也放到选项列表中：\n" +
		"   - 如果题目中已经明确标明了正确答案（如\"答案：C\"、\"正确答案是B\"等），不要将答案内容本身作为选项\n" +
		"   - 选项列表中应该只包含题目给出的待选项（如A、B、C、D等选项内容）\n" +
		"   - 答案字段（answer）只需标注选项字母（如[\"C\"]），不要将答案内容重复放入options数组\n\n" +
		"【答案格式要求】\n" +
		"1. 统一使用答案数组格式：[\"A\"] 或 [\"A\", \"B\", \"C\"]\n" +
		"2. 单选题：返回单个元素的数组，如 [\"A\"] 或 [\"B\"]\n" +
		"3. 多选题：返回多个元素的数组，如 [\"A\", \"B\", \"C\"] 或 [\"A\", \"B\", \"C\", \"D\"]\n" +
		"4. 答案使用字母格式（A, B, C, D, E, F, G, H, I, J），对应选项的顺序\n" +
		"5. 根据答案数组的长度自动判断题目类型（1个答案=单选，多个答案=多选）\n" +
		"6. 答案字母必须与options数组中的选项顺序一一对应（第0个选项=A，第1个选项=B，以此类推）\n\n" +
		"【返回格式示例 - 严格按照此格式】\n" +
		"你的回复必须是以下格式，一个字都不能多，一个字都不能少：\n" +
		"{\n" +
		"  \"questions\": [\n" +
		"    {\n" +
		"      \"question\": \"题目内容（所有特殊字符都要转义）\",\n" +
		"      \"options\": [\"选项A（特殊字符已转义）\", \"选项B\", \"选项C\", \"选项D\"],\n" +
		"      \"answer\": [\"A\"],\n" +
		"      \"explanation\": \"解析内容（可选，没有则空字符串）\"\n" +
		"    },\n" +
		"    {\n" +
		"      \"question\": \"多选题题目内容\",\n" +
		"      \"options\": [\"选项A\", \"选项B\", \"选项C\", \"选项D\"],\n" +
		"      \"answer\": [\"A\", \"B\", \"C\"],\n" +
		"      \"explanation\": \"\"\n" +
		"    }\n" +
		"  ]\n" +
		"}\n\n" +
		"【最终检查清单 - 返回前必须确认】\n" +
		"✓ 回复是纯JSON，没有任何前缀（如\"根据要求\"、\"以下是\"、\"好的\"等）\n" +
		"✓ 回复是纯JSON，没有任何后缀（如\"希望这些信息对您有帮助\"、\"以上是\"等）\n" +
		"✓ 回复是纯JSON，没有markdown代码块标记（没有```json或```）\n" +
		"✓ 所有双引号都已正确转义（字符串中的\"变为\\\"）\n" +
		"✓ 所有反斜杠都已正确转义（\\变为\\\\）\n" +
		"✓ answer字段始终是数组格式\n" +
		"✓ 可以直接被JSON解析器解析\n" +
		"✓ 回复以{开始，以}结束\n\n" +
		"【再次强调】\n" +
		"你的回复必须是且只能是JSON对象，以{开始，以}结束，中间不能有任何非JSON内容。\n" +
		"不要添加任何解释、说明、问候语或其他文字。\n" +
		"现在开始解析以下文本内容，严格按照上述要求返回JSON格式：\n\n" +
		text
}


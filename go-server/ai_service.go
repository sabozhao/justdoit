package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
)

var configMutex sync.RWMutex // 保护配置读取的互斥锁

// 腾讯云API配置
var (
	tencentCloudSecretId  string
	tencentCloudSecretKey string
	tencentCloudRegion    string = "ap-beijing"
	tencentCloudEndpoint  string = "hunyuan.tencentcloudapi.com"
	tencentCloudModel     string = "hunyuan-lite" // 默认使用混元精简版（免费）
)

// 初始化腾讯云API
func initTencentCloudAI() {
	configMutex.Lock()
	defer configMutex.Unlock()
	
	// 优先从数据库读取配置，如果数据库中没有则从环境变量读取
	secretId := getSystemSetting("tencent_secret_id")
	if secretId == "" {
		secretId = getEnv("TENCENT_SECRET_ID", "")
	}
	
	secretKey := getSystemSetting("tencent_secret_key")
	if secretKey == "" {
		secretKey = getEnv("TENCENT_SECRET_KEY", "")
	}
	
	region := getSystemSetting("tencent_region")
	if region == "" {
		region = getEnv("TENCENT_REGION", "ap-beijing")
	}
	
	model := getSystemSetting("tencent_model")
	if model == "" {
		model = getEnv("TENCENT_MODEL", "hunyuan-lite")
	}
	
	endpoint := getSystemSetting("tencent_endpoint")
	if endpoint == "" {
		endpoint = getEnv("TENCENT_ENDPOINT", "hunyuan.tencentcloudapi.com")
	}

	if secretId == "" || secretKey == "" {
		// 如果没有设置SecretId和SecretKey，AI功能将被禁用
		return
	}

	tencentCloudSecretId = secretId
	tencentCloudSecretKey = secretKey
	tencentCloudRegion = region
	tencentCloudModel = model
	tencentCloudEndpoint = endpoint
	
	log.Printf("AI服务配置 - 区域: %s, 模型: %s, 端点: %s", region, model, endpoint)
}

// 重新加载腾讯云API配置（从数据库）
func reloadTencentCloudConfig() {
	initTencentCloudAI()
}

// 获取当前腾讯云配置（用于API返回，隐藏敏感信息）
func getTencentCloudConfig() map[string]string {
	configMutex.RLock()
	defer configMutex.RUnlock()
	
	return map[string]string{
		"tencent_secret_id":  tencentCloudSecretId,
		"tencent_secret_key": maskSecretKey(tencentCloudSecretKey),
		"tencent_region":     tencentCloudRegion,
		"tencent_model":      tencentCloudModel,
		"tencent_endpoint":   tencentCloudEndpoint,
	}
}

// 掩码密钥（只显示前3位和后3位）
func maskSecretKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 6 {
		return "***"
	}
	return key[:3] + "***" + key[len(key)-3:]
}

// 使用腾讯云混元大模型识别题目
func recognizeQuestionsWithAI(text string) ([]Question, error) {
	if tencentCloudSecretId == "" || tencentCloudSecretKey == "" {
		return nil, fmt.Errorf("AI服务未配置，请在环境变量中设置 TENCENT_SECRET_ID 和 TENCENT_SECRET_KEY")
	}

	// 构建提示词（强化JSON格式要求，确保所有模型都能返回纯JSON）
	prompt := "你是一个专业的题目解析助手。请从以下文本中识别出所有选择题，并严格按照JSON格式返回。\n\n" +
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

	// 调用腾讯云混元大模型API
	responseText, err := callTencentCloudAPI(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI识别失败: %v", err)
	}

	// 保存原始响应（用于调试）
	originalResponseText := responseText
	
	// 打印AI返回的原始结果到日志（一次性完整打印）
	log.Printf("========== AI返回原始结果（完整内容，长度: %d 字符）==========", len(responseText))
	log.Printf("原始响应内容:\n%s", originalResponseText)
	log.Printf("===================================")

	// 解析AI返回的JSON
	responseText = strings.TrimSpace(responseText)
	if strings.HasPrefix(responseText, "```json") {
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimSuffix(responseText, "```")
	} else if strings.HasPrefix(responseText, "```") {
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
	}
	responseText = strings.TrimSpace(responseText)
	
	// 打印清理后的响应
	log.Printf("========== 清理后的响应（长度: %d 字符）==========", len(responseText))
	log.Printf("清理后的内容:\n%s", responseText)
	log.Printf("===================================")

	// 使用临时结构体，answer字段统一为数组格式
	type TempQuestion struct {
		Question    string   `json:"question"`
		Options     []string `json:"options"`
		Answer      []string `json:"answer"` // 统一为字符串数组格式，如["A"]或["A","B","C"]
		Explanation string   `json:"explanation"`
	}

	var result struct {
		Questions []TempQuestion `json:"questions"`
	}

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		log.Printf("========== JSON解析错误 ==========")
		log.Printf("错误类型: %T", err)
		log.Printf("错误信息: %v", err)
		log.Printf("尝试解析的内容长度: %d 字符", len(responseText))
		log.Printf("尝试解析的内容字节长度: %d 字节", len([]byte(responseText)))
		
		// 打印完整的响应内容（用于调试）
		log.Printf("========== 完整响应内容（用于调试）==========")
		log.Printf("原始响应（清理前，长度: %d 字符）:\n%s", len(originalResponseText), originalResponseText)
		log.Printf("清理后响应（长度: %d 字符）:\n%s", len(responseText), responseText)
		
		// 打印响应的前100和最后100个字符
		if len(responseText) > 200 {
			log.Printf("响应前100字符: %s", responseText[:100])
			log.Printf("响应最后100字符: %s", responseText[len(responseText)-100:])
		} else {
			log.Printf("完整响应（小于200字符）: %s", responseText)
		}
		
		// 打印字节级别的详细信息（如果有特殊字符）
		log.Printf("响应字节内容（十六进制，前200字节）:")
		responseBytes := []byte(responseText)
		maxBytes := 200
		if len(responseBytes) < maxBytes {
			maxBytes = len(responseBytes)
		}
		log.Printf("% x", responseBytes[:maxBytes])
		log.Printf("===================================")
		
		// 策略1: 尝试提取JSON部分（去掉可能的markdown标记和多余文本）
		jsonStart := strings.Index(responseText, "{")
		jsonEnd := strings.LastIndex(responseText, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			extractedJSON := responseText[jsonStart : jsonEnd+1]
			log.Printf("策略1: 尝试提取JSON部分: %d-%d, 长度: %d\n", jsonStart, jsonEnd, len(extractedJSON))
			err = json.Unmarshal([]byte(extractedJSON), &result)
			if err == nil {
				log.Printf("策略1成功：提取JSON部分后解析成功\n")
				responseText = extractedJSON
			} else {
				log.Printf("策略1失败: %v\n", err)
			}
		}
		
		// 策略2: 尝试使用更宽松的JSON解析（逐个解析题目对象）
		if err != nil {
			log.Printf("尝试策略2：逐个解析题目对象\n")
			// 找到questions数组
			questionsStart := strings.Index(responseText, `"questions":`)
			if questionsStart >= 0 {
				// 找到数组开始 [
				arrayStartIdx := strings.Index(responseText[questionsStart:], "[")
				if arrayStartIdx >= 0 {
					arrayStartIdx += questionsStart
					
					// 找到匹配的 ]，但要注意嵌套的数组
					bracketCount := 0
					arrayEndIdx := -1
					for i := arrayStartIdx; i < len(responseText); i++ {
						if responseText[i] == '[' {
							bracketCount++
						} else if responseText[i] == ']' {
							bracketCount--
							if bracketCount == 0 {
								arrayEndIdx = i
								break
							}
						}
					}
					
					if arrayEndIdx > arrayStartIdx {
						// 提取questions数组部分
						questionsArrayText := responseText[arrayStartIdx+1 : arrayEndIdx]
						log.Printf("找到questions数组: %d-%d, 长度: %d\n", arrayStartIdx+1, arrayEndIdx, len(questionsArrayText))
						
						// 尝试手动解析每个题目对象
						var parsedQuestions []TempQuestion
						currentPos := 0
						for currentPos < len(questionsArrayText) {
							// 找到下一个 {（题目对象的开始）
							objStart := strings.Index(questionsArrayText[currentPos:], "{")
							if objStart < 0 {
								break
							}
							objStart += currentPos
							
							// 找到匹配的 }
							braceCount := 0
							objEnd := -1
							for i := objStart; i < len(questionsArrayText); i++ {
								if questionsArrayText[i] == '"' && i > 0 && questionsArrayText[i-1] != '\\' {
									// 跳过字符串内容，直到找到未转义的引号
									j := i + 1
									for j < len(questionsArrayText) {
										if questionsArrayText[j] == '"' && questionsArrayText[j-1] != '\\' {
											i = j
											break
										}
										j++
									}
									continue
								}
								if questionsArrayText[i] == '{' {
									braceCount++
								} else if questionsArrayText[i] == '}' {
									braceCount--
									if braceCount == 0 {
										objEnd = i
										break
									}
								}
							}
							
							if objEnd > objStart {
								objText := questionsArrayText[objStart : objEnd+1]
								// 包装成完整的JSON对象
								fullObjText := "{" + objText + "}"
								var q struct {
									Question    string   `json:"question"`
									Options     []string `json:"options"`
									Answer      []string `json:"answer"` // 统一为字符串数组格式
									Explanation string   `json:"explanation"`
								}
								if json.Unmarshal([]byte(fullObjText), &q) == nil {
									parsedQuestions = append(parsedQuestions, TempQuestion{
										Question:    q.Question,
										Options:     q.Options,
										Answer:      q.Answer,
										Explanation: q.Explanation,
									})
									log.Printf("成功解析第 %d 个题目对象\n", len(parsedQuestions))
								} else {
									log.Printf("解析题目对象失败，跳过\n")
								}
								currentPos = objEnd + 1
							} else {
								break
							}
						}
						
						if len(parsedQuestions) > 0 {
							log.Printf("策略2成功：手动解析出 %d 道题目\n", len(parsedQuestions))
							result.Questions = parsedQuestions
							err = nil
						} else {
							log.Printf("策略2失败：未能解析出任何题目\n")
						}
					}
				}
			}
		}
		
		// 如果所有策略都失败，返回详细错误信息
		if err != nil {
			errorMsg := err.Error()
			log.Printf("========== 最终JSON解析失败 ==========")
			log.Printf("错误信息: %v", err)
			log.Printf("尝试解析的内容长度: %d 字符", len(responseText))
			log.Printf("原始响应长度: %d 字符", len(originalResponseText))
			
			// 打印完整的原始响应（不截断）
			log.Printf("========== 完整原始响应（不截断）==========")
			log.Printf("%s", originalResponseText)
			log.Printf("===================================")
			
			// 打印完整的清理后响应（不截断）
			log.Printf("========== 完整清理后响应（不截断）==========")
			log.Printf("%s", responseText)
			log.Printf("===================================")
			
			// 打印响应统计信息
			log.Printf("========== 响应统计信息 ==========")
			log.Printf("原始响应字符数: %d", len(originalResponseText))
			log.Printf("清理后响应字符数: %d", len(responseText))
			log.Printf("原始响应字节数: %d", len([]byte(originalResponseText)))
			log.Printf("清理后响应字节数: %d", len([]byte(responseText)))
			
			// 检查是否以 { 开始，以 } 结束
			trimmed := strings.TrimSpace(responseText)
			startsWithBrace := len(trimmed) > 0 && trimmed[0] == '{'
			endsWithBrace := len(trimmed) > 0 && trimmed[len(trimmed)-1] == '}'
			log.Printf("是否以 { 开始: %v", startsWithBrace)
			log.Printf("是否以 } 结束: %v", endsWithBrace)
			
			// 检查括号匹配
			openBraces := strings.Count(responseText, "{")
			closeBraces := strings.Count(responseText, "}")
			log.Printf("开括号数量: %d", openBraces)
			log.Printf("闭括号数量: %d", closeBraces)
			log.Printf("括号是否匹配: %v", openBraces == closeBraces)
			
			// 检查中括号匹配
			openBrackets := strings.Count(responseText, "[")
			closeBrackets := strings.Count(responseText, "]")
			log.Printf("开中括号数量: %d", openBrackets)
			log.Printf("闭中括号数量: %d", closeBrackets)
			log.Printf("中括号是否匹配: %v", openBrackets == closeBrackets)
			log.Printf("===================================")
			
			return nil, fmt.Errorf("解析AI返回的JSON失败: %v\n原始响应长度: %d字符\n清理后响应长度: %d字符\n错误详情: %s\n提示：已打印完整响应到日志，请检查日志获取详细信息", 
				err, len(originalResponseText), len(responseText), errorMsg)
		}
	}
	
	log.Printf("成功解析JSON，识别出 %d 道题目\n", len(result.Questions))

	// 验证和清理题目数据，并转换answer字段
	var validQuestions []Question
	for i, tempQ := range result.Questions {
		// 确保选项数量正确（至少2个，最多10个）
		if len(tempQ.Options) < 2 {
			log.Printf("警告: 第%d题选项不足，已跳过 (选项数: %d)\n", i+1, len(tempQ.Options))
			continue // 跳过选项不足的题目
		}

		// 转换answer字段为索引数组（answer现在统一是字符串数组格式，如["A"]或["A","B","C"]）
		answerIndices := parseAnswerArrayFromStringArray(tempQ.Answer, len(tempQ.Options))
		if len(answerIndices) == 0 {
			// 如果无法解析答案，记录详细信息并跳过该题目
			log.Printf("警告: 第%d题答案解析失败，已跳过。答案: %v (类型: %T)，选项数: %d\n", i+1, tempQ.Answer, tempQ.Answer, len(tempQ.Options))
			continue
		}

		// 验证选项数量（最多10个）
		if len(tempQ.Options) > 10 {
			log.Printf("警告: 第%d题选项数量超过10个（当前：%d），已跳过\n", i+1, len(tempQ.Options))
			continue
		}

		// 创建Question对象
		q := Question{
			Question:    strings.TrimSpace(tempQ.Question),
			Options:     make([]string, len(tempQ.Options)),
			Answer:      answerIndices,
			IsMultiple:  len(answerIndices) > 1, // 多个答案则为多选题
			Explanation: strings.TrimSpace(tempQ.Explanation),
		}

		// 清理选项：移除开头的字母前缀（如"A "、"A. "、"A- "、"B "等），并过滤无效选项
		var validOptions []string
		for _, opt := range tempQ.Options {
			cleanedOpt := cleanOptionPrefix(opt)
			
			// 过滤无效选项：
			// 1. 空选项
			// 2. 只包含"说明书"或"正确答案"等提示性文字的选项
			// 3. 长度过短的选项（可能是误解析）
			if cleanedOpt == "" {
				continue
			}
			
			// 检查是否是"说明书"、"正确答案"等提示性文字（不应该是选项）
			optLower := strings.ToLower(strings.TrimSpace(cleanedOpt))
			if strings.Contains(optLower, "说明书") && len(optLower) < 20 {
				// 如果选项主要是"说明书"，跳过
				continue
			}
			
			// 如果选项长度太短（少于3个字符），可能是无效选项
			if len(strings.TrimSpace(cleanedOpt)) < 3 {
				continue
			}
			
			validOptions = append(validOptions, cleanedOpt)
		}
		
		// 更新选项数组和答案索引
		if len(validOptions) < 2 {
			log.Printf("警告: 第%d题有效选项不足2个，已跳过 (有效选项数: %d)\n", i+1, len(validOptions))
			continue
		}
		
		// 如果选项数量变化了，需要重新映射答案索引
		// 由于我们过滤的是后面的选项（如E选项），前面的选项索引不变
		// 所以答案索引应该仍然是有效的
		q.Options = validOptions
		
		// 验证答案索引是否仍然有效
		maxAnswerIndex := -1
		for _, idx := range q.Answer {
			if idx > maxAnswerIndex {
				maxAnswerIndex = idx
			}
		}
		if maxAnswerIndex >= len(validOptions) {
			log.Printf("警告: 第%d题答案索引超出范围，已跳过 (最大索引: %d, 有效选项数: %d)\n", i+1, maxAnswerIndex, len(validOptions))
			continue
		}

		validQuestions = append(validQuestions, q)
	}

	if len(validQuestions) == 0 {
		log.Printf("警告: 所有题目验证失败，原始题目数: %d\n", len(result.Questions))
		return nil, fmt.Errorf("AI未能识别出有效的题目（原始识别 %d 道，验证后 0 道）", len(result.Questions))
	}

	log.Printf("成功验证 %d 道题目（原始 %d 道）\n", len(validQuestions), len(result.Questions))
	return validQuestions, nil
}

// 调用腾讯云混元大模型API（使用官方SDK，确保签名正确）
func callTencentCloudAPI(prompt string) (string, error) {
	if tencentCloudSecretId == "" || tencentCloudSecretKey == "" {
		return "", fmt.Errorf("AI服务未配置，请在环境变量中设置 TENCENT_SECRET_ID 和 TENCENT_SECRET_KEY")
	}

	// 实例化认证对象
	credential := common.NewCredential(tencentCloudSecretId, tencentCloudSecretKey)

	// 实例化客户端配置对象
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = tencentCloudEndpoint
	cpf.HttpProfile.ReqTimeout = 300 // 设置请求超时时间为300秒（5分钟）

	// 实例化要请求产品的client对象
	client, err := hunyuan.NewClient(credential, tencentCloudRegion, cpf)
	if err != nil {
		return "", fmt.Errorf("创建客户端失败: %v", err)
	}

	// 实例化请求对象
	request := hunyuan.NewChatCompletionsRequest()

	// 设置模型（从配置读取）
	request.Model = common.StringPtr(tencentCloudModel)

	// 设置消息
	message := hunyuan.Message{
		Role:    common.StringPtr("user"),
		Content: common.StringPtr(prompt),
	}
	request.Messages = []*hunyuan.Message{&message}

	// 设置温度
	request.Temperature = common.Float64Ptr(0.3)
	
	// 尝试设置MaxTokens（虽然SDK可能不支持，但先尝试）
	// 注意：腾讯云混元API可能通过模型自身限制输出长度
	// 如果需要更长输出，可能需要使用流式传输或拆分请求
	
	log.Printf("API请求配置 - 模型: %s, 温度: 0.3", tencentCloudModel)

	// 发送请求
	response, err := client.ChatCompletions(request)
	if err != nil {
		return "", fmt.Errorf("API调用失败: %v", err)
	}

	// 检查响应
	if response.Response == nil {
		return "", fmt.Errorf("API响应为空")
	}

	if response.Response.Choices == nil || len(response.Response.Choices) == 0 {
		return "", fmt.Errorf("API未返回有效结果")
	}

	// 返回内容
	choice := response.Response.Choices[0]
	if choice.Message == nil {
		return "", fmt.Errorf("API返回的Message为空")
	}

	if choice.Message.Content == nil {
		return "", fmt.Errorf("API返回的Content为空")
	}

	content := *choice.Message.Content
	
	// 检查响应是否完整（检查FinishReason）
	if choice.FinishReason != nil {
		finishReason := *choice.FinishReason
		log.Printf("API响应完成原因: %s", finishReason)
		
		// 如果是长度限制，说明响应被截断
		if finishReason == "length" || finishReason == "max_tokens" {
			log.Printf("警告: 响应因长度限制而被截断，可能需要增加输出长度限制或拆分请求")
		}
	}
	
	log.Printf("API返回内容长度: %d 字符, %d 字节", len(content), len([]byte(content)))
	
	// 检查响应是否以 } 结束（JSON应该以 } 结束）
	trimmed := strings.TrimSpace(content)
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] != '}' {
		log.Printf("警告: 响应可能不完整（不以 } 结束），实际长度: %d 字符", len(content))
		log.Printf("响应最后50字符: %s", func() string {
			if len(content) > 50 {
				return content[len(content)-50:]
			}
			return content
		}())
	}

	return content, nil
}

// parseAnswerFromInterface 从多种格式的答案中解析出索引
// 支持：数字（索引）、字符串（如"A"或"C"）、数组（如["A","B","C","D"]，取第一个）
func parseAnswerFromInterface(answer interface{}, optionCount int) int {
	if answer == nil {
		return -1
	}

	// 如果是数字（已经是索引）
	switch v := answer.(type) {
	case float64:
		idx := int(v)
		if idx >= 0 && idx < optionCount {
			return idx
		}
		// 如果是1-based索引，转换为0-based
		if idx > 0 && idx <= optionCount {
			return idx - 1
		}
	case int:
		idx := v
		if idx >= 0 && idx < optionCount {
			return idx
		}
		// 如果是1-based索引，转换为0-based
		if idx > 0 && idx <= optionCount {
			return idx - 1
		}
	case int64:
		idx := int(v)
		if idx >= 0 && idx < optionCount {
			return idx
		}
		if idx > 0 && idx <= optionCount {
			return idx - 1
		}
	}

	// 如果是字符串
	if str, ok := answer.(string); ok {
		// 先尝试解析为单个答案
		idx := parseAnswerFromString(str, optionCount)
		if idx != -1 {
			return idx
		}
		// 如果是多选题格式（如"ABC", "A,B,C", "A B C"），返回第一个有效答案
		// 注意：这里只返回第一个，完整的解析由 parseAnswerArrayFromInterface 处理
		return -1
	}

	// 如果是数组（多选题，取第一个答案）
	if arr, ok := answer.([]interface{}); ok && len(arr) > 0 {
		// 取第一个答案
		return parseAnswerFromInterface(arr[0], optionCount)
	}

	// 如果是数组（字符串数组）
	if strArr, ok := answer.([]string); ok && len(strArr) > 0 {
		return parseAnswerFromString(strArr[0], optionCount)
	}

	return -1
}

// parseAnswerArrayFromStringArray 从字符串数组格式的答案中解析出索引数组（支持多选）
// 输入格式：["A"] 或 ["A", "B", "C"]，统一为字符串数组
// 输出：索引数组，如[0]或[0,1,2]
func parseAnswerArrayFromStringArray(answer []string, optionCount int) []int {
	if answer == nil || len(answer) == 0 {
		return nil
	}

	var result []int
	seen := make(map[int]bool)

	for _, item := range answer {
		idx := parseAnswerFromString(item, optionCount)
		if idx != -1 {
			// 去重
			if !seen[idx] {
				result = append(result, idx)
				seen[idx] = true
			}
		}
	}

	if len(result) > 0 {
		return result
	}

	return nil
}

// 从答案字符串中解析出索引（单选题）
func parseAnswerFromString(answerStr string, optionCount int) int {
	answerStr = strings.ToUpper(strings.TrimSpace(answerStr))

	// 尝试解析为字母（A, B, C, D）
	if len(answerStr) == 1 {
		char := answerStr[0]
		if char >= 'A' && char <= 'Z' {
			idx := int(char - 'A')
			if idx < optionCount {
				return idx
			}
		}
	}

	// 尝试解析为数字
	if idx, err := strconv.Atoi(answerStr); err == nil {
		// 可能是1-based索引，先尝试不减1
		if idx >= 0 && idx < optionCount {
			return idx
		}
		// 尝试减1（转换为0-based索引）
		idx -= 1
		if idx >= 0 && idx < optionCount {
			return idx
		}
	}

	return -1
}

// parseMultipleAnswerFromString 从多选题字符串格式解析出索引数组
// 支持格式：不区分大小写的字母组合（如 "ABC", "abc", "A,B,C", "A B C", "A、B、C"）
// 例如："ABC" -> [0,1,2], "ACD" -> [0,2,3]
func parseMultipleAnswerFromString(answerStr string, optionCount int) []int {
	answerStr = strings.ToUpper(strings.TrimSpace(answerStr))
	if answerStr == "" {
		return nil
	}

	var result []int
	seen := make(map[int]bool)

	// 尝试多种分隔符：逗号、空格、中文顿号、斜杠等
	separators := []string{",", " ", "、", "/", "|"}
	var parts []string
	found := false
	
	for _, sep := range separators {
		if strings.Contains(answerStr, sep) {
			parts = strings.Split(answerStr, sep)
			// 去除每个部分的前后空白
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			found = true
			break
		}
	}

	if !found {
		// 如果没有分隔符，尝试作为连续字母解析（如"ABC"）
		// 检查是否都是有效字母
		allLetters := true
		for _, char := range answerStr {
			if char < 'A' || char > 'Z' {
				allLetters = false
				break
			}
		}
		
		if allLetters && len(answerStr) > 1 {
			// 作为连续字母解析
			for _, char := range answerStr {
				if char >= 'A' && char <= 'Z' {
					idx := int(char - 'A')
					if idx < optionCount && !seen[idx] {
						result = append(result, idx)
						seen[idx] = true
					}
				}
			}
			if len(result) > 0 {
				return result
			}
		}
		// 如果不是连续字母格式，返回空
		return nil
	}

	// 如果有分隔符，解析每个部分
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// 尝试解析为单个字母
		if len(part) == 1 {
			char := part[0]
			if char >= 'A' && char <= 'Z' {
				idx := int(char - 'A')
				if idx < optionCount && !seen[idx] {
					result = append(result, idx)
					seen[idx] = true
				}
			}
		} else {
			// 尝试解析为数字
			if idx, err := strconv.Atoi(part); err == nil {
				if idx >= 0 && idx < optionCount && !seen[idx] {
					result = append(result, idx)
					seen[idx] = true
				} else if idx > 0 && idx <= optionCount && !seen[idx-1] {
					// 1-based索引转换为0-based
					result = append(result, idx-1)
					seen[idx-1] = true
				}
			}
		}
	}

	if len(result) > 0 {
		return result
	}
	
	return nil
}

// cleanOptionPrefix 清理选项内容，移除开头的字母前缀
// 支持的格式：A、A.、A-、A )、A)、A. )、B、B.、B-、B )、B)等
// 例如："A. 选项内容" -> "选项内容"，"B- 选项内容" -> "选项内容"
func cleanOptionPrefix(option string) string {
	option = strings.TrimSpace(option)
	if option == "" {
		return option
	}

	// 匹配模式：开头的字母（A-Z），后面可能跟着以下字符：.、-、 )、)、空格等，然后是实际内容
	// 使用正则表达式匹配
	re := regexp.MustCompile(`^([A-Z])([.\-)\s]*)\s*`)
	matches := re.FindStringSubmatch(option)
	
	if len(matches) > 0 && matches[0] != "" {
		// 找到匹配的前缀，移除它
		prefix := matches[0]
		cleaned := strings.TrimPrefix(option, prefix)
		cleaned = strings.TrimSpace(cleaned)
		
		// 如果移除前缀后内容为空，保留原内容
		if cleaned == "" {
			return option
		}
		return cleaned
	}

	return option
}

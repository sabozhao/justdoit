package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func debugTencentAPI() {
	fmt.Println("========================================")
	fmt.Println("腾讯云混元API详细调试")
	fmt.Println("========================================")
	fmt.Println()

	// 检查环境变量或数据库配置
	if tencentCloudSecretId == "" || tencentCloudSecretKey == "" {
		fmt.Println("⚠️  AI配置未初始化，尝试从环境变量读取...")
		configMutex.Lock()
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
		configMutex.Unlock()
		
		if tencentCloudSecretId == "" || tencentCloudSecretKey == "" {
			fmt.Println("❌ 错误: 未找到腾讯云AI配置")
			fmt.Println("   请设置环境变量 TENCENT_SECRET_ID 和 TENCENT_SECRET_KEY")
			fmt.Println("   或者确保数据库已初始化并包含配置")
			return
		}
		fmt.Println("✓ 从环境变量读取配置成功")
	}

	fmt.Printf("AI配置:\n")
	fmt.Printf("  区域: %s\n", tencentCloudRegion)
	fmt.Printf("  模型: %s\n", tencentCloudModel)
	fmt.Printf("  端点: %s\n", tencentCloudEndpoint)
	fmt.Println()

	// 读取test.docx文件
	fmt.Println("步骤 1: 读取 test.docx 文件...")
	testFilePath := "../test.docx"
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		testFilePath = "test.docx"
	}
	
	text, err := parseTestDocx(testFilePath)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		return
	}
	fmt.Printf("✓ 文本长度: %d 字符\n", len(text))
	if len(text) > 200 {
		fmt.Printf("文本预览: %s...\n\n", text[:200])
	}

	// 构建提示词（使用优化后的提示词）
	prompt := buildOptimizedPrompt(text)

	// 调用API
	fmt.Println("步骤 2: 调用腾讯云混元API（使用官方SDK）...")
	response, err := callTencentCloudAPI(prompt)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("API响应:")
	fmt.Println("========================================")
	fmt.Println(response)
}

func parseTestDocx(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var docXML *zip.File
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			docXML = f
			break
		}
	}

	if docXML == nil {
		return "", fmt.Errorf("未找到 document.xml")
	}

	rc, err := docXML.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	xmlData, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	return extractTextFromXML(string(xmlData)), nil
}

// buildOptimizedPrompt 构建优化后的提示词（与ai_service.go保持一致）
func buildOptimizedPrompt(text string) string {
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

func extractTextFromXML(xmlContent string) string {
	var textBuilder strings.Builder
	start := 0
	for {
		startIdx := strings.Index(xmlContent[start:], "<w:t>")
		if startIdx == -1 {
			break
		}
		startIdx += start + 5
		endIdx := strings.Index(xmlContent[startIdx:], "</w:t>")
		if endIdx == -1 {
			break
		}
		text := xmlContent[startIdx : startIdx+endIdx]
		text = strings.ReplaceAll(text, "&lt;", "<")
		text = strings.ReplaceAll(text, "&gt;", ">")
		text = strings.ReplaceAll(text, "&amp;", "&")
		text = strings.ReplaceAll(text, "&quot;", "\"")
		text = strings.ReplaceAll(text, "&apos;", "'")
		textBuilder.WriteString(text)
		textBuilder.WriteString(" ")
		start = startIdx + endIdx + 6
	}
	result := textBuilder.String()
	result = strings.ReplaceAll(result, "  ", " ")
	return strings.TrimSpace(result)
}

func testCallAPI(prompt string) (string, error) {
	// 构建请求体
	requestBody := map[string]interface{}{
		"Model": "hunyuan-lite",
		"Messages": []map[string]interface{}{
			{
				"Role":    "user",
				"Content": prompt,
			},
		},
		"Temperature": 0.3,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	fmt.Printf("请求体: %s\n\n", string(jsonData))

	// 构建请求
	service := "hunyuan"
	action := "ChatCompletions"
	version := "2023-09-01"
	requestURL := fmt.Sprintf("https://%s/", tencentCloudEndpoint)

	req, err := http.NewRequestWithContext(context.Background(), "POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	// 设置请求头
	timestamp := time.Now().Unix()
	// 注意：Content-Type应该只是 application/json，不要包含 charset=utf-8
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", tencentCloudEndpoint)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Region", tencentCloudRegion)

	// 生成签名
	fmt.Println("步骤 3: 生成TC3签名...")
	fmt.Printf("SecretId: %s...\n", tencentCloudSecretId[:10])
	fmt.Printf("SecretKey: %s...\n", tencentCloudSecretKey[:10])
	fmt.Printf("Region: %s\n", tencentCloudRegion)
	fmt.Printf("Endpoint: %s\n", tencentCloudEndpoint)
	fmt.Printf("Timestamp: %d\n", timestamp)
	
	authorization, canonicalReq, stringToSign, err := generateTC3SignatureDebug(req, service, timestamp, string(jsonData))
	if err != nil {
		return "", err
	}
	
	fmt.Printf("\n规范请求 (Canonical Request):\n%s\n\n", canonicalReq)
	fmt.Printf("待签名字符串 (String To Sign):\n%s\n\n", stringToSign)
	fmt.Printf("Authorization头:\n%s\n\n", authorization)
	
	req.Header.Set("Authorization", authorization)
	req.Body = io.NopCloser(bytes.NewBuffer(jsonData))

	// 发送请求
	fmt.Println("步骤 4: 发送HTTP请求...")
	client := &http.Client{Timeout: 5 * time.Minute} // 5分钟超时
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fmt.Printf("响应状态码: %d\n\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("错误响应体: %s\n", string(body))
		
		var errorResp struct {
			Response struct {
				Error struct {
					Code    string `json:"Code"`
					Message string `json:"Message"`
				} `json:"Error"`
			} `json:"Response"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Response.Error.Code != "" {
			return "", fmt.Errorf("%s - %s", errorResp.Response.Error.Code, errorResp.Response.Error.Message)
		}
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp struct {
		Response struct {
			Choices []struct {
				Message struct {
					Content string `json:"Content"`
				} `json:"Message"`
			} `json:"Choices"`
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v\n响应内容: %s", err, string(body))
	}

	if apiResp.Response.Error.Code != "" {
		return "", fmt.Errorf("%s - %s", apiResp.Response.Error.Code, apiResp.Response.Error.Message)
	}

	if len(apiResp.Response.Choices) == 0 {
		return "", fmt.Errorf("未返回有效结果")
	}

	return apiResp.Response.Choices[0].Message.Content, nil
}

func generateTC3SignatureDebug(req *http.Request, service string, timestamp int64, payload string) (string, string, string, error) {
	// 1. 构建规范请求
	canonicalRequest := buildCanonicalRequestDebug(req, payload)
	fmt.Printf("1. 规范请求哈希: %s\n", hex.EncodeToString(hashSHA256Debug(canonicalRequest)))

	// 2. 构建待签名字符串
	// 尝试两种日期格式
	date1 := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	date2 := time.Unix(timestamp, 0).UTC().Format("20060102")
	
	fmt.Printf("尝试日期格式1 (YYYY-MM-DD): %s\n", date1)
	fmt.Printf("尝试日期格式2 (YYYYMMDD): %s\n", date2)
	
	// 使用YYYY-MM-DD格式（根据错误提示）
	dateForCredential := date1
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", dateForCredential, service)
	fmt.Printf("CredentialScope: %s\n", credentialScope)
	
	stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%d\n%s\n%s",
		timestamp,
		credentialScope,
		hex.EncodeToString(hashSHA256Debug(canonicalRequest)),
	)

	// 3. 计算签名（使用YYYY-MM-DD格式计算签名密钥）
	dateForSigning := date1
	fmt.Printf("签名计算日期 (YYYY-MM-DD): %s\n", dateForSigning)
	
	dateKey := hmacSHA256Debug([]byte("TC3"+tencentCloudSecretKey), dateForSigning)
	fmt.Printf("DateKey: %s...\n", hex.EncodeToString(dateKey)[:16])
	
	serviceKey := hmacSHA256Debug(dateKey, service)
	fmt.Printf("ServiceKey: %s...\n", hex.EncodeToString(serviceKey)[:16])
	
	regionKey := hmacSHA256Debug(serviceKey, tencentCloudRegion)
	fmt.Printf("RegionKey: %s...\n", hex.EncodeToString(regionKey)[:16])
	
	signingKey := hmacSHA256Debug(regionKey, "tc3_request")
	fmt.Printf("SigningKey: %s...\n", hex.EncodeToString(signingKey)[:16])
	
	signature := hex.EncodeToString(hmacSHA256Debug(signingKey, stringToSign))
	fmt.Printf("最终签名: %s\n", signature)

	// 4. 构建Authorization头
	authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s",
		tencentCloudSecretId,
		credentialScope,
		signature,
	)

	return authorization, canonicalRequest, stringToSign, nil
}

func buildCanonicalRequestDebug(req *http.Request, payload string) string {
	method := req.Method
	reqURL, _ := url.Parse(req.URL.String())
	canonicalURI := reqURL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQueryString := reqURL.RawQuery

	signedHeaders := []string{"content-type", "host"}
	headerMap := make(map[string]string)
	for _, header := range signedHeaders {
		value := req.Header.Get(header)
		if value == "" && header == "host" {
			value = req.Host
			if value == "" {
				value = req.URL.Host
			}
		}
		// Content-Type值需要完整保留，但可能需要规范化
		if header == "content-type" {
			// 保持原始值，但去掉多余空格
			headerMap[header] = strings.ToLower(strings.TrimSpace(value))
		} else {
			headerMap[header] = strings.ToLower(strings.TrimSpace(value))
		}
	}

	sort.Strings(signedHeaders)
	var canonicalHeaders strings.Builder
	for _, header := range signedHeaders {
		canonicalHeaders.WriteString(header)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(headerMap[header]))
		canonicalHeaders.WriteString("\n")
	}

	signedHeadersStr := strings.Join(signedHeaders, ";")
	payloadHash := hex.EncodeToString(hashSHA256Debug(payload))

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders.String(),
		signedHeadersStr,
		payloadHash,
	)

	return canonicalRequest
}

func hashSHA256Debug(data string) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hmacSHA256Debug(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}


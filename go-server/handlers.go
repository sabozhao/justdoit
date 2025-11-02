package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tealeg/xlsx/v3"
)

// 题库相关处理函数
func getQuestionBanks(c *gin.Context) {
	userID := c.GetString("userID")

	query := `
		SELECT qb.id, qb.user_id, qb.name, qb.description, qb.created_at, COUNT(q.id) as question_count
		FROM question_banks qb 
		LEFT JOIN questions q ON qb.id = q.bank_id 
		WHERE qb.user_id = ?
		GROUP BY qb.id 
		ORDER BY qb.created_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var banks []QuestionBank
	for rows.Next() {
		var bank QuestionBank
		err := rows.Scan(&bank.ID, &bank.UserID, &bank.Name, &bank.Description, &bank.CreatedAt, &bank.QuestionCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		banks = append(banks, bank)
	}

	c.JSON(http.StatusOK, banks)
}

func getQuestionBankByID(c *gin.Context) {
	userID := c.GetString("userID")
	bankID := c.Param("id")

	// 获取题库信息
	var bank QuestionBank
	err := db.QueryRow("SELECT id, user_id, name, description, created_at FROM question_banks WHERE id = ? AND user_id = ?",
		bankID, userID).Scan(&bank.ID, &bank.UserID, &bank.Name, &bank.Description, &bank.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question bank not found"})
		return
	}

	// 获取题目
	rows, err := db.Query("SELECT id, bank_id, question, options, answer, is_multiple, explanation FROM questions WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		var optionsJSON, answerJSON string
		var isMultiple bool
		err := rows.Scan(&q.ID, &q.BankID, &q.Question, &optionsJSON, &answerJSON, &isMultiple, &q.Explanation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 解析选项JSON
		err = json.Unmarshal([]byte(optionsJSON), &q.Options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse options"})
			return
		}

		// 解析答案JSON（支持数组）
		err = json.Unmarshal([]byte(answerJSON), &q.Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse answer"})
			return
		}

		q.IsMultiple = isMultiple
		questions = append(questions, q)
	}

	bank.Questions = questions
	c.JSON(http.StatusOK, bank)
}

func createQuestionBank(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		Name        string     `json:"name" binding:"required"`
		Description string     `json:"description"`
		Questions   []Question `json:"questions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// 插入题库
	bankID := generateUUID()
	_, err = tx.Exec("INSERT INTO question_banks (id, user_id, name, description) VALUES (?, ?, ?, ?)",
		bankID, userID, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create question bank"})
		return
	}

	// 插入题目
	for _, q := range req.Questions {
		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal options"})
			return
		}

		answerJSON, err := json.Marshal(q.Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal answer"})
			return
		}

		// 判断是否为多选题
		isMultiple := len(q.Answer) > 1

		questionID := generateUUID()
		_, err = tx.Exec("INSERT INTO questions (id, bank_id, question, options, answer, is_multiple, explanation) VALUES (?, ?, ?, ?, ?, ?, ?)",
			questionID, bankID, q.Question, string(optionsJSON), string(answerJSON), isMultiple, q.Explanation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create question"})
			return
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          bankID,
		"name":        req.Name,
		"description": req.Description,
		"message":     "Question bank created successfully",
	})
}

func uploadQuestionBankFile(c *gin.Context) {
	userID := c.GetString("userID")
	bankID := c.Param("id")

	// 检查题库是否存在且属于当前用户
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM question_banks WHERE id = ? AND user_id = ?)", bankID, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question bank not found"})
		return
	}

	// 获取解析模式（默认为固定格式）
	parseMode := c.PostForm("parseMode")
	if parseMode == "" {
		parseMode = "format" // 默认使用固定格式
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}
	defer file.Close()

	// 获取文件扩展名
	filename := header.Filename
	ext := strings.ToLower(filepath.Ext(filename))

	var questions []Question

	// 根据解析模式处理文件
	if parseMode == "ai" {
		// AI 自动分析模式：所有文件类型都通过 AI 解析
		log.Printf("使用AI模式解析文件: %s", filename)
		questions, err = parseFileWithAI(file, header, ext)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "AI解析失败: " + err.Error()})
			return
		}
	} else {
		// 固定格式解析模式：根据文件类型使用对应的解析器
		log.Printf("使用固定格式模式解析文件: %s", filename)
		switch ext {
		case ".xlsx", ".xls":
			questions, err = parseExcelFile(file)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Excel文件解析失败: " + err.Error()})
				return
			}
		case ".csv":
			questions, err = parseCSVFile(file)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "CSV文件解析失败: " + err.Error()})
				return
			}
		case ".docx":
			questions, err = parseDOCXFileFormat(file, header)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "DOCX文件解析失败: " + err.Error()})
				return
			}
		case ".doc":
			c.JSON(http.StatusBadRequest, gin.H{"error": "旧版DOC格式(.doc)暂不支持固定格式解析，请转换为DOCX格式或使用AI自动分析"})
			return
		case ".pdf":
			c.JSON(http.StatusBadRequest, gin.H{"error": "PDF文件不支持固定格式解析，请使用AI自动分析"})
			return
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件格式"})
			return
		}
	}

	// 检查是否有题目
	if len(questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件中未找到有效的题目"})
		return
	}

	log.Printf("成功解析文件，识别出 %d 道题目（解析模式: %s）\n", len(questions), parseMode)

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// 插入题目
	successCount := 0
	for i, q := range questions {
		// 验证选项数量（最多10个）
		if len(q.Options) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("第 %d 题选项数量超过10个（当前：%d）", i+1, len(q.Options))})
			return
		}

		// 验证答案索引
		if len(q.Answer) == 0 {
			log.Printf("警告: 第 %d 题答案为空，已跳过\n", i+1)
			continue // 跳过答案为空的题目，但继续处理其他题目
		}
		validAnswer := true
		for _, ansIdx := range q.Answer {
			if ansIdx < 0 || ansIdx >= len(q.Options) {
				log.Printf("警告: 第 %d 题答案索引 %d 超出选项范围（选项数: %d），已跳过\n", i+1, ansIdx, len(q.Options))
				validAnswer = false
				break
			}
		}
		if !validAnswer {
			continue // 跳过答案无效的题目
		}

		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			log.Printf("警告: 第 %d 题选项序列化失败: %v，已跳过\n", i+1, err)
			continue
		}

		answerJSON, err := json.Marshal(q.Answer)
		if err != nil {
			log.Printf("警告: 第 %d 题答案序列化失败: %v，已跳过\n", i+1, err)
			continue
		}

		// 判断是否为多选题（答案数量>1）
		isMultiple := len(q.Answer) > 1

		questionID := generateUUID()
		_, err = tx.Exec("INSERT INTO questions (id, bank_id, question, options, answer, is_multiple, explanation) VALUES (?, ?, ?, ?, ?, ?, ?)",
			questionID, bankID, q.Question, string(optionsJSON), string(answerJSON), isMultiple, q.Explanation)
		if err != nil {
			log.Printf("错误: 第 %d 题插入失败: %v\n", i+1, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("第 %d 题插入失败: %v", i+1, err)})
			return
		}
		successCount++
	}

	if successCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有成功插入任何题目，请检查题目格式"})
		return
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	log.Printf("成功插入 %d/%d 道题目到题库 %s\n", successCount, len(questions), bankID)

	// 获取题库信息
	var bankName, bankDescription string
	err = db.QueryRow("SELECT name, description FROM question_banks WHERE id = ?", bankID).Scan(&bankName, &bankDescription)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get question bank info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            bankID,
		"name":          bankName,
		"description":   bankDescription,
		"questionCount": successCount,
		"message":       fmt.Sprintf("成功导入 %d/%d 道题目到题库", successCount, len(questions)),
	})
}

func deleteQuestionBank(c *gin.Context) {
	userID := c.GetString("userID")
	bankID := c.Param("id")

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// 删除相关的错题
	_, err = tx.Exec("DELETE FROM wrong_questions WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wrong questions"})
		return
	}

	// 删除相关的考试结果
	_, err = tx.Exec("DELETE FROM exam_results WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete exam results"})
		return
	}

	// 删除题目
	_, err = tx.Exec("DELETE FROM questions WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete questions"})
		return
	}

	// 删除题库
	result, err := tx.Exec("DELETE FROM question_banks WHERE id = ? AND user_id = ?", bankID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete question bank"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question bank not found"})
		return
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question bank deleted successfully"})
}

// 文件解析函数
// downloadDemoFile 下载示例文件
func downloadDemoFile(c *gin.Context) {
	fileType := c.Param("type")
	
	switch fileType {
	case "excel", "xlsx":
		// 生成Excel示例文件
		downloadExcelDemo(c)
	case "csv":
		// 生成CSV示例文件
		downloadCSVDemo(c)
	case "docx":
		// 生成DOCX示例文件
		downloadDOCXDemo(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型"})
	}
}

// downloadExcelDemo 下载Excel示例文件
func downloadExcelDemo(c *gin.Context) {
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Sheet1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建Excel文件失败"})
		return
	}

	// 添加表头（新格式：题目、正确答案、选项A-J、解析）
	headerRow := sheet.AddRow()
	headerRow.AddCell().SetValue("题目")
	headerRow.AddCell().SetValue("正确答案") // 答案列在第2列（固定位置）
	// 添加选项列（A-J，最多10个）
	for i := 0; i < 10; i++ {
		optionLabel := string(rune('A' + i))
		headerRow.AddCell().SetValue("选项" + optionLabel)
	}
	headerRow.AddCell().SetValue("解析")

	// 添加示例数据
	// 格式：题目、答案、选项A、选项B、选项C、选项D、选项E...、解析
	examples := [][]string{
		{"这是一道单选题？", "A", "选项A的内容", "选项B的内容", "选项C的内容", "选项D的内容", "", "", "", "", "", "", "这是单选题的解析"},
		{"这是另一道单选题？", "B", "第一个选项", "第二个选项", "第三个选项", "第四个选项", "", "", "", "", "", "", "这是第二题的解析"},
		{"多选题示例？", "A,B,C", "选项A", "选项B", "选项C", "选项D", "", "", "", "", "", "", "这是多选题的解析"},
		{"最多选项示例？", "J", "选项A", "选项B", "选项C", "选项D", "选项E", "选项F", "选项G", "选项H", "选项I", "选项J", "支持最多10个选项"},
	}

	for _, example := range examples {
		row := sheet.AddRow()
		for _, value := range example {
			row.AddCell().SetValue(value)
		}
	}

	// 设置响应头
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=题库格式示例.xlsx")
	c.Header("Content-Transfer-Encoding", "binary")

	// 写入响应
	err = file.Write(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入Excel文件失败"})
		return
	}
}

// downloadCSVDemo 下载CSV示例文件
func downloadCSVDemo(c *gin.Context) {
	// 设置响应头
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=题库格式示例.csv")
	c.Header("Content-Transfer-Encoding", "binary")

	// CSV内容（使用UTF-8 BOM支持中文）
	// 新格式：题目、正确答案、选项A-J、解析
	csvContent := "\xEF\xBB\xBF" // UTF-8 BOM
	
	// 表头
	csvContent += "题目,正确答案"
	for i := 0; i < 10; i++ {
		optionLabel := string(rune('A' + i))
		csvContent += ",选项" + optionLabel
	}
	csvContent += ",解析\n"
	
	// 示例数据
	csvContent += "这是一道单选题？,A,选项A的内容,选项B的内容,选项C的内容,选项D的内容,,,,,,,这是单选题的解析\n"
	csvContent += "这是另一道单选题？,B,第一个选项,第二个选项,第三个选项,第四个选项,,,,,,,这是第二题的解析\n"
	csvContent += "多选题示例？,\"A,B,C\",选项A,选项B,选项C,选项D,,,,,,,这是多选题的解析\n"
	csvContent += "最多选项示例？,J,选项A,选项B,选项C,选项D,选项E,选项F,选项G,选项H,选项I,选项J,支持最多10个选项\n"

	// 写入响应
	c.String(http.StatusOK, csvContent)
}

// downloadDOCXDemo 下载DOCX示例文件
func downloadDOCXDemo(c *gin.Context) {
	// DOCX示例内容（固定格式）- 注意：开头不要有格式说明，直接是题目示例
	docxContent := `这是一道单选题？
A. 选项A的内容
B. 选项B的内容
C. 选项C的内容（可选）
D. 选项D的内容（可选）
答案：A
解析：这是单选题的解析（可选）

这是另一道单选题？
A. 第一个选项
B. 第二个选项
C. 第三个选项
D. 第四个选项
答案：B
解析：这是第二题的解析

多选题示例？
A. 选项A
B. 选项B
C. 选项C
D. 选项D
答案：A,B,C
解析：这是多选题的解析`

	// 设置响应头
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Header("Content-Disposition", "attachment; filename=题库格式示例.docx")
	c.Header("Content-Transfer-Encoding", "binary")

	// 注意：这里我们需要生成一个真正的DOCX文件
	// 由于DOCX是ZIP压缩的XML文件，我们创建一个简单的DOCX文件
	err := generateSimpleDOCX(c, docxContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成DOCX文件失败: " + err.Error()})
		return
	}
}

// generateSimpleDOCX 生成简单的DOCX文件
func generateSimpleDOCX(c *gin.Context, content string) error {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "docx-demo-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建ZIP文件
	zipPath := filepath.Join(tempDir, "demo.docx")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 添加基本DOCX结构文件
	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

	addFileToZip(zipWriter, "[Content_Types].xml", contentTypes)

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	addFileToZip(zipWriter, "_rels/.rels", rels)

	// word/_rels/document.xml.rels
	wordRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`

	addFileToZip(zipWriter, "word/_rels/document.xml.rels", wordRels)

	// word/document.xml - 这是主要内容
	// 将文本内容转换为Word的段落格式
	lines := strings.Split(content, "\n")
	var paragraphs []string
	for _, line := range lines {
		// 保留原行的换行和空格，但去掉首尾空白
		trimmedLine := strings.TrimRight(line, "\r\n")
		
		// 转义XML特殊字符（注意顺序：先转义&，再转义<和>）
		escapedLine := strings.ReplaceAll(trimmedLine, "&", "&amp;")
		escapedLine = strings.ReplaceAll(escapedLine, "<", "&lt;")
		escapedLine = strings.ReplaceAll(escapedLine, ">", "&gt;")
		escapedLine = strings.ReplaceAll(escapedLine, "\"", "&quot;")
		
		if escapedLine == "" {
			paragraphs = append(paragraphs, `<w:p><w:r><w:t></w:t></w:r></w:p>`)
		} else {
			paragraphs = append(paragraphs, fmt.Sprintf(`<w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, escapedLine))
		}
	}

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
` + strings.Join(paragraphs, "\n") + `
</w:body>
</w:document>`

	err = addFileToZip(zipWriter, "word/document.xml", documentXML)
	if err != nil {
		return fmt.Errorf("添加document.xml失败: %v", err)
	}

	zipWriter.Close()
	zipFile.Close()

	// 读取生成的文件并写入响应
	fileData, err := os.ReadFile(zipPath)
	if err != nil {
		return fmt.Errorf("读取DOCX文件失败: %v", err)
	}

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", fileData)
	return nil
}

// addFileToZip 添加文件到ZIP
func addFileToZip(zipWriter *zip.Writer, filename string, content string) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(content))
	return err
}

func parseUploadedFile(file multipart.File, header *multipart.FileHeader) ([]Question, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))

	switch ext {
	case ".json":
		return parseJSONFile(file)
	case ".xlsx", ".xls":
		return parseExcelFile(file)
	case ".csv":
		return parseCSVFile(file)
	case ".pdf":
		// 提取PDF文本，然后使用AI识别
		text, err := parsePDFFile(file, header)
		if err != nil {
			return nil, err
		}
		// 使用AI识别题目
		return recognizeQuestionsWithAI(text)
	case ".doc", ".docx":
		// 提取DOC/DOCX文本，然后使用AI识别
		text, err := parseDOCFile(file, header)
		if err != nil {
			return nil, err
		}
		// 使用AI识别题目
		return recognizeQuestionsWithAI(text)
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s。支持格式：JSON, Excel, CSV, PDF, DOC/DOCX", ext)
	}
}

func parseJSONFile(file multipart.File) ([]Question, error) {
	var data struct {
		Questions []Question `json:"questions"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON文件解析失败: %v", err)
	}

	if len(data.Questions) == 0 {
		return nil, fmt.Errorf("未找到有效的题目数据")
	}

	return data.Questions, nil
}

func parseExcelFile(file multipart.File) ([]Question, error) {
	// 创建临时文件来保存上传的Excel文件
	tempFile, err := os.CreateTemp("", "upload_*.xlsx")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 将上传的文件内容复制到临时文件
	_, err = io.Copy(tempFile, file)
	if err != nil {
		return nil, fmt.Errorf("保存临时文件失败: %v", err)
	}

	// 打开Excel文件
	xlFile, err := xlsx.OpenFile(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("Excel文件打开失败: %v", err)
	}

	if len(xlFile.Sheets) == 0 {
		return nil, fmt.Errorf("Excel文件中没有工作表")
	}

	sheet := xlFile.Sheets[0]

	// 将sheet转换为二维字符串数组
	var rows [][]string

	// 遍历工作表的行
	err = sheet.ForEachRow(func(r *xlsx.Row) error {
		var rowData []string
		err := r.ForEachCell(func(c *xlsx.Cell) error {
			value := c.String()
			rowData = append(rowData, value)
			return nil
		})
		if err != nil {
			return err
		}
		rows = append(rows, rowData)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("读取Excel数据失败: %v", err)
	}

	return parseExcelData(rows)
}

func parseCSVFile(file multipart.File) ([]Question, error) {
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV文件读取失败: %v", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV文件数据不足")
	}

	return parseCSVData(records)
}

// parseFileWithAI 使用AI解析文件（支持PDF、DOC、DOCX、Excel、CSV等）
func parseFileWithAI(file multipart.File, header *multipart.FileHeader, ext string) ([]Question, error) {
	var text string
	var err error

	// 根据文件类型提取文本
	switch ext {
	case ".pdf":
		// PDF文件：提取文本，然后使用AI识别
		text, err = parsePDFFile(file, header)
		if err != nil {
			return nil, fmt.Errorf("PDF文本提取失败: %v", err)
		}
	case ".doc", ".docx":
		// DOCX文件解析
		text, err = parseDOCXFile(file)
		if err != nil {
			return nil, fmt.Errorf("Word文件解析失败: %v", err)
		}
	case ".xlsx", ".xls":
		// Excel文件：先解析为文本格式
		xlFile, err := parseExcelFileAsText(file)
		if err != nil {
			return nil, fmt.Errorf("Excel文件解析失败: %v", err)
		}
		text = xlFile
	case ".csv":
		// CSV文件：转换为文本格式
		csvText, err := parseCSVFileAsText(file)
		if err != nil {
			return nil, fmt.Errorf("CSV文件解析失败: %v", err)
		}
		text = csvText
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	// 使用AI识别题目
	questions, err := recognizeQuestionsWithAI(text)
	if err != nil {
		return nil, fmt.Errorf("AI识别失败: %v", err)
	}

	return questions, nil
}

// parseExcelFileAsText 将Excel文件解析为文本格式（用于AI分析）
func parseExcelFileAsText(file multipart.File) (string, error) {
	// 读取文件到临时文件
	tempFile, err := os.CreateTemp("", "excel-*.xlsx")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		return "", fmt.Errorf("保存临时文件失败: %v", err)
	}

	// 打开Excel文件
	xlFile, err := xlsx.OpenFile(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("Excel文件打开失败: %v", err)
	}

	if len(xlFile.Sheets) == 0 {
		return "", fmt.Errorf("Excel文件中没有工作表")
	}

	sheet := xlFile.Sheets[0]
	var textBuilder strings.Builder

	// 遍历工作表的行，转换为文本格式
	err = sheet.ForEachRow(func(r *xlsx.Row) error {
		err := r.ForEachCell(func(c *xlsx.Cell) error {
			value := c.String()
			textBuilder.WriteString(value)
			textBuilder.WriteString("\t")
			return nil
		})
		if err != nil {
			return err
		}
		textBuilder.WriteString("\n")
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("读取Excel数据失败: %v", err)
	}

	return textBuilder.String(), nil
}

// parseCSVFileAsText 将CSV文件解析为文本格式（用于AI分析）
func parseCSVFileAsText(file multipart.File) (string, error) {
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("CSV文件读取失败: %v", err)
	}

	var textBuilder strings.Builder
	for _, record := range records {
		textBuilder.WriteString(strings.Join(record, "\t"))
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

// parseDOCXFile 解析DOCX文件（提取文本，用于AI分析）
func parseDOCXFile(file multipart.File) (string, error) {
	// 读取文件内容到临时文件
	tempFile, err := os.CreateTemp("", "docx-*.docx")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		return "", fmt.Errorf("保存临时文件失败: %v", err)
	}
	tempFile.Close()

	// 创建FileHeader
	dummyHeader := &multipart.FileHeader{
		Filename: "temp.docx",
	}
	
	// 重新打开文件
	tempFileReader, err := os.Open(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("打开临时文件失败: %v", err)
	}
	defer tempFileReader.Close()

	// 调用parseDOCFile函数（在pdf_doc_parser.go中定义）
	return parseDOCFile(tempFileReader, dummyHeader)
}

// parseDOCXFileFormat 解析DOCX文件的固定格式（用于固定格式解析）
func parseDOCXFileFormat(file multipart.File, header *multipart.FileHeader) ([]Question, error) {
	// 先提取文本
	text, err := parseDOCXFile(file)
	if err != nil {
		log.Printf("解析DOCX文件错误: 提取文本失败: %v", err)
		return nil, fmt.Errorf("提取DOCX文本失败: %v", err)
	}

	// 记录提取的文本内容（用于调试）
	log.Printf("DOCX文件提取的文本内容（前500字符）: %s", truncateString(text, 500))
	log.Printf("DOCX文件文本总长度: %d 字符", len(text))

	// 按照固定格式解析文本
	questions, err := parseFormattedText(text)
	if err != nil {
		log.Printf("解析DOCX文件错误: 解析格式化文本失败: %v", err)
		log.Printf("DOCX文件文本内容（前1000字符）: %s", truncateString(text, 1000))
		return nil, err
	}

	log.Printf("DOCX文件解析成功: 共解析出 %d 道题目", len(questions))
	return questions, nil
}

// truncateString 截断字符串，如果超过长度则添加...
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// parseFormattedText 解析固定格式的文本内容
// 格式示例：
// 题目内容？
// A. 选项A内容
// B. 选项B内容
// C. 选项C内容（可选）
// D. 选项D内容（可选）
// 答案：A
// 解析：这是解析内容（可选）
//
// [空行分隔下一题]
func parseFormattedText(text string) ([]Question, error) {
	var questions []Question
	lines := strings.Split(text, "\n")
	
	log.Printf("开始解析格式化文本: 共 %d 行", len(lines))
	
	var currentQuestion *Question
	var currentSection string // "question", "options", "answer", "explanation"
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// 记录前10行内容用于调试
		if i < 10 {
			log.Printf("解析第 %d 行: [%s]", i+1, line)
		}
		
		// 跳过空行（如果是空行且在题目中间，表示题目结束）
		if line == "" {
			if currentQuestion != nil && len(currentQuestion.Options) >= 2 {
				// 验证并保存当前题目
				if currentQuestion.Question == "" {
					log.Printf("解析错误: 第 %d 行附近：题目内容为空", i+1)
					return nil, fmt.Errorf("第 %d 行附近：题目内容为空", i+1)
				}
				if len(currentQuestion.Answer) == 0 {
					log.Printf("解析错误: 第 %d 行附近：未找到答案，题目: %s, 选项数: %d", i+1, currentQuestion.Question, len(currentQuestion.Options))
					return nil, fmt.Errorf("第 %d 行附近：未找到答案", i+1)
				}
				log.Printf("保存题目 %d: 题目=%s, 选项数=%d, 答案=%v, 是否多选=%v", 
					len(questions)+1, currentQuestion.Question, len(currentQuestion.Options), 
					currentQuestion.Answer, currentQuestion.IsMultiple)
				questions = append(questions, *currentQuestion)
				currentQuestion = nil
			}
			currentSection = ""
			continue
		}
		
		// 检测答案行
		if strings.HasPrefix(line, "答案：") || strings.HasPrefix(line, "答案:") || 
		   strings.HasPrefix(line, "正确答案：") || strings.HasPrefix(line, "正确答案:") ||
		   strings.HasPrefix(line, "Answer：") || strings.HasPrefix(line, "Answer:") {
			if currentQuestion == nil {
				log.Printf("解析错误: 第 %d 行：答案出现在题目定义之前，答案行: %s", i+1, line)
				return nil, fmt.Errorf("第 %d 行：答案出现在题目定义之前", i+1)
			}
			log.Printf("解析第 %d 行: 找到答案行: %s", i+1, line)
			
			// 提取答案
			answerStr := strings.TrimPrefix(line, "答案：")
			answerStr = strings.TrimPrefix(answerStr, "答案:")
			answerStr = strings.TrimPrefix(answerStr, "正确答案：")
			answerStr = strings.TrimPrefix(answerStr, "正确答案:")
			answerStr = strings.TrimPrefix(answerStr, "Answer：")
			answerStr = strings.TrimPrefix(answerStr, "Answer:")
			answerStr = strings.TrimSpace(answerStr)
			
			// 解析答案（支持单选和多选）
			answers := strings.Split(answerStr, ",")
			var answerIndices []int
			for _, ans := range answers {
				ans = strings.TrimSpace(ans)
				if ans == "" {
					continue
				}
				
				// 解析字母答案
				var idx int = -1
				if len(ans) == 1 {
					switch strings.ToUpper(ans) {
					case "A":
						idx = 0
					case "B":
						idx = 1
					case "C":
						idx = 2
					case "D":
						idx = 3
					case "E":
						idx = 4
					case "F":
						idx = 5
					case "G":
						idx = 6
					case "H":
						idx = 7
					case "I":
						idx = 8
					case "J":
						idx = 9
					}
				}
				
				// 解析数字答案（1-10）
				if idx == -1 {
					if num, err := strconv.Atoi(ans); err == nil {
						if num >= 1 && num <= 10 {
							idx = num - 1
						}
					}
				}
				
				if idx >= 0 && idx < len(currentQuestion.Options) {
					answerIndices = append(answerIndices, idx)
				}
			}
			
			if len(answerIndices) == 0 {
				log.Printf("解析错误: 第 %d 行：无法解析答案，答案字符串: %s，当前题目选项数: %d", 
					i+1, answerStr, len(currentQuestion.Options))
				return nil, fmt.Errorf("第 %d 行：无法解析答案", i+1)
			}
			
			log.Printf("解析答案成功: 第 %d 行，答案索引: %v，选项数: %d", 
				i+1, answerIndices, len(currentQuestion.Options))
			currentQuestion.Answer = answerIndices
			currentSection = "answer"
			continue
		}
		
		// 检测解析行
		if strings.HasPrefix(line, "解析：") || strings.HasPrefix(line, "解析:") ||
		   strings.HasPrefix(line, "Explanation：") || strings.HasPrefix(line, "Explanation:") {
			if currentQuestion == nil {
				return nil, fmt.Errorf("第 %d 行：解析出现在题目定义之前", i+1)
			}
			
			explanation := strings.TrimPrefix(line, "解析：")
			explanation = strings.TrimPrefix(explanation, "解析:")
			explanation = strings.TrimPrefix(explanation, "Explanation：")
			explanation = strings.TrimPrefix(explanation, "Explanation:")
			explanation = strings.TrimSpace(explanation)
			
			currentQuestion.Explanation = explanation
			currentSection = "explanation"
			continue
		}
		
		// 检测选项行（A. B. C. D. E. F. G. H. I. J.）
		isOption := false
		var optionIndex int = -1
		var optionText string
		
		optionPatterns := []string{"A.", "B.", "C.", "D.", "E.", "F.", "G.", "H.", "I.", "J."}
		for idx, pattern := range optionPatterns {
			if strings.HasPrefix(line, pattern) || strings.HasPrefix(line, strings.ToLower(pattern)) {
				optionText = strings.TrimPrefix(line, pattern)
				optionText = strings.TrimPrefix(optionText, strings.ToLower(pattern))
				optionText = strings.TrimSpace(optionText)
				if len(optionText) > 0 {
					isOption = true
					optionIndex = idx
					break
				}
			}
		}
		
		if isOption {
			log.Printf("解析第 %d 行: 找到选项 %s，选项文本: %s", i+1, string(rune('A'+optionIndex)), optionText)
			if currentQuestion == nil {
				currentQuestion = &Question{
					Question:    "",
					Options:     []string{},
					Answer:      []int{},
					Explanation: "",
					IsMultiple:  false,
				}
				log.Printf("创建新题目对象")
			}
			
			// 添加选项
			if optionIndex >= len(currentQuestion.Options) {
				// 填充缺失的选项
				for len(currentQuestion.Options) < optionIndex {
					currentQuestion.Options = append(currentQuestion.Options, "")
				}
			}
			if optionIndex == len(currentQuestion.Options) {
				currentQuestion.Options = append(currentQuestion.Options, optionText)
			} else {
				currentQuestion.Options[optionIndex] = optionText
			}
			
			log.Printf("当前题目选项数: %d", len(currentQuestion.Options))
			currentSection = "options"
			continue
		}
		
		// 其他情况：可能是题目内容
		// 检查是否是新的题目开始（当前题目已完整且有答案，且遇到新的题目行）
		// 新题目的特征：不是选项、不是答案、不是解析，且当前题目已经有答案
		// 当currentSection为空且当前题目完整时，说明可能是新题目开始
		isNewQuestionStart := currentQuestion != nil && 
			len(currentQuestion.Answer) > 0 && 
			len(currentQuestion.Options) >= 2 &&
			(currentSection == "" || currentSection == "explanation") &&
			!strings.HasPrefix(line, "A.") && 
			!strings.HasPrefix(line, "B.") &&
			!strings.HasPrefix(line, "答案") &&
			!strings.HasPrefix(line, "解析") &&
			len(currentQuestion.Question) > 0
		
		if isNewQuestionStart {
			// 保存当前完整题目
			log.Printf("检测到新题目开始（第%d行: %s），保存当前题目: 题目=%s, 选项数=%d, 答案=%v", 
				i+1, line, currentQuestion.Question, len(currentQuestion.Options), currentQuestion.Answer)
			questions = append(questions, *currentQuestion)
			currentQuestion = nil
			currentSection = ""
		}
		
		if currentQuestion == nil {
			currentQuestion = &Question{
				Question:    "",
				Options:     []string{},
				Answer:      []int{},
				Explanation: "",
				IsMultiple:  false,
			}
		}
		
		// 如果还没有题目内容，且不是选项、答案、解析，则作为题目内容
		if currentQuestion.Question == "" && currentSection == "" {
			currentQuestion.Question = line
			currentSection = "question"
		} else if currentSection == "question" {
			// 题目内容可以跨多行
			currentQuestion.Question += "\n" + line
		}
	}
	
	// 处理最后一个题目
	if currentQuestion != nil {
		log.Printf("处理最后一个题目: 题目=%s, 选项数=%d, 答案=%v", 
			currentQuestion.Question, len(currentQuestion.Options), currentQuestion.Answer)
		if len(currentQuestion.Options) >= 2 {
			if currentQuestion.Question == "" {
				log.Printf("解析错误: 最后一个题目：题目内容为空")
				return nil, fmt.Errorf("最后一个题目：题目内容为空")
			}
			if len(currentQuestion.Answer) == 0 {
				log.Printf("解析错误: 最后一个题目：未找到答案，题目: %s, 选项数: %d", 
					currentQuestion.Question, len(currentQuestion.Options))
				return nil, fmt.Errorf("最后一个题目：未找到答案")
			}
			questions = append(questions, *currentQuestion)
			log.Printf("保存最后一个题目成功")
		} else {
			log.Printf("警告: 最后一个题目选项数不足（%d < 2），题目: %s", 
				len(currentQuestion.Options), currentQuestion.Question)
		}
	}
	
	if len(questions) == 0 {
		log.Printf("解析错误: 未找到任何题目，请检查文件格式。文本总长度: %d 字符", len(text))
		log.Printf("文本内容预览（前500字符）: %s", truncateString(text, 500))
		return nil, fmt.Errorf("未找到任何题目，请检查文件格式")
	}
	
	// 设置多选题标志
	for i := range questions {
		questions[i].IsMultiple = len(questions[i].Answer) > 1
	}
	
	return questions, nil
}

func parseExcelData(rows [][]string) ([]Question, error) {
	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel数据不足")
	}

	// 获取表头
	headers := make(map[string]int)
	for i, header := range rows[0] {
		headers[strings.TrimSpace(header)] = i
	}

	// 查找列索引
	// 新格式：题目（第1列）、正确答案（第2列，固定位置）、选项A-J、解析
	questionCol := findColumnIndex(headers, []string{"题目", "question", "Question", "问题"})
	
	// 答案列固定在第2列（索引1），但也支持通过列名查找（兼容旧格式）
	answerCol := 1 // 默认第2列（固定位置）
	if foundCol := findColumnIndex(headers, []string{"正确答案", "answer", "Answer", "答案"}); foundCol != -1 {
		answerCol = foundCol
	}
	
	// 查找选项列（A-J，最多10个）
	optionCols := make([]int, 10) // 最多10个选项（A-J）
	optionCols[0] = findColumnIndex(headers, []string{"选项A", "A", "optionA", "选择A"})
	optionCols[1] = findColumnIndex(headers, []string{"选项B", "B", "optionB", "选择B"})
	// 从第3列开始查找选项（索引2开始）
	for i := 2; i < 10; i++ {
		// 如果表头中有定义，使用定义的位置；否则使用默认位置（索引 i+1，因为前面有题目和答案列）
		optionLabel := string(rune('A' + i))
		colNames := []string{"选项" + optionLabel, optionLabel, "option" + optionLabel, "选择" + optionLabel}
		if foundCol := findColumnIndex(headers, colNames); foundCol != -1 {
			optionCols[i] = foundCol
		} else {
			// 默认位置：第3列开始（索引2），依次是选项A、B、C...
			optionCols[i] = i + 1
		}
	}
	
	explanationCol := findColumnIndex(headers, []string{"解析", "explanation", "Explanation", "说明"})

	if questionCol == -1 || optionCols[0] == -1 || optionCols[1] == -1 || answerCol == -1 {
		return nil, fmt.Errorf("缺少必要的列：题目（第1列）、正确答案（第2列）、选项A、选项B")
	}

	var questions []Question
	for i := 1; i < len(rows); i++ {
		row := rows[i]

		question := getExcelValue(row, questionCol)
		if question == "" {
			continue
		}

		// 收集所有非空选项（A-J，最多10个）
		var options []string
		for j := 0; j < 10; j++ {
			if optionCols[j] != -1 && optionCols[j] < len(row) {
				optionValue := getExcelValue(row, optionCols[j])
				if optionValue != "" {
					// 如果选项列表为空或当前索引匹配，添加选项
					if j == len(options) {
						options = append(options, optionValue)
					} else if j < len(options) {
						options[j] = optionValue
					}
				}
			} else if j < 12 { // 默认位置：从第3列开始（索引2）
				// 尝试使用默认位置
				colIndex := j + 2 // 题目(0) + 答案(1) + 选项从索引2开始
				if colIndex < len(row) {
					optionValue := getExcelValue(row, colIndex)
					if optionValue != "" {
						// 确保选项列表足够长
						for len(options) <= j {
							options = append(options, "")
						}
						options[j] = optionValue
					}
				}
			}
		}

		// 移除末尾的空选项
		for len(options) > 0 && options[len(options)-1] == "" {
			options = options[:len(options)-1]
		}

		if len(options) < 2 {
			continue // 至少需要2个选项
		}

		// 解析答案（支持单选和多选）
		answerStr := getExcelValue(row, answerCol)
		if answerStr == "" {
			continue
		}

		// 检查是否是多选题（答案中包含逗号）
		var answer []int
		if strings.Contains(answerStr, ",") {
			// 多选题：解析多个答案
			answers := strings.Split(answerStr, ",")
			for _, ans := range answers {
				ans = strings.TrimSpace(ans)
				if ans == "" {
					continue
				}
				idx, err := parseAnswerMultiple(ans, len(options))
				if err != nil {
					return nil, fmt.Errorf("第 %d 行答案格式错误: %v", i+1, err)
				}
				answer = append(answer, idx)
			}
			// 去重并排序
			seen := make(map[int]bool)
			var uniqueAnswer []int
			for _, idx := range answer {
				if !seen[idx] && idx >= 0 && idx < len(options) {
					seen[idx] = true
					uniqueAnswer = append(uniqueAnswer, idx)
				}
			}
			answer = uniqueAnswer
		} else {
			// 单选题
			idx, err := parseAnswerMultiple(answerStr, len(options))
			if err != nil {
				return nil, fmt.Errorf("第 %d 行答案格式错误: %v", i+1, err)
			}
			answer = []int{idx}
		}

		if len(answer) == 0 {
			continue
		}

		explanation := ""
		if explanationCol != -1 {
			explanation = getExcelValue(row, explanationCol)
		}

		isMultiple := len(answer) > 1
		questions = append(questions, Question{
			Question:    question,
			Options:     options,
			Answer:      answer,
			IsMultiple:  isMultiple,
			Explanation: explanation,
		})
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("未找到有效的题目数据")
	}

	return questions, nil
}

func parseCSVData(records [][]string) ([]Question, error) {
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV数据不足")
	}

	// 获取表头
	headers := make(map[string]int)
	for i, header := range records[0] {
		headers[strings.TrimSpace(header)] = i
	}

	// 查找列索引
	// 新格式：题目（第1列）、正确答案（第2列，固定位置）、选项A-J、解析
	questionCol := findColumnIndex(headers, []string{"题目", "question", "Question", "问题"})
	
	// 答案列固定在第2列（索引1），但也支持通过列名查找（兼容旧格式）
	answerCol := 1 // 默认第2列（固定位置）
	if foundCol := findColumnIndex(headers, []string{"正确答案", "answer", "Answer", "答案"}); foundCol != -1 {
		answerCol = foundCol
	}
	
	// 查找选项列（A-J，最多10个）
	optionCols := make([]int, 10) // 最多10个选项（A-J）
	optionCols[0] = findColumnIndex(headers, []string{"选项A", "A", "optionA", "选择A"})
	optionCols[1] = findColumnIndex(headers, []string{"选项B", "B", "optionB", "选择B"})
	// 从第3列开始查找选项（索引2开始）
	for i := 2; i < 10; i++ {
		// 如果表头中有定义，使用定义的位置；否则使用默认位置（索引 i+1，因为前面有题目和答案列）
		optionLabel := string(rune('A' + i))
		colNames := []string{"选项" + optionLabel, optionLabel, "option" + optionLabel, "选择" + optionLabel}
		if foundCol := findColumnIndex(headers, colNames); foundCol != -1 {
			optionCols[i] = foundCol
		} else {
			// 默认位置：第3列开始（索引2），依次是选项A、B、C...
			optionCols[i] = i + 1
		}
	}
	
	explanationCol := findColumnIndex(headers, []string{"解析", "explanation", "Explanation", "说明"})

	if questionCol == -1 || optionCols[0] == -1 || optionCols[1] == -1 || answerCol == -1 {
		return nil, fmt.Errorf("缺少必要的列：题目（第1列）、正确答案（第2列）、选项A、选项B")
	}

	var questions []Question
	for i := 1; i < len(records); i++ {
		record := records[i]

		question := getCSVValue(record, questionCol)
		if question == "" {
			continue
		}

		// 收集所有非空选项（A-J，最多10个）
		var options []string
		for j := 0; j < 10; j++ {
			if optionCols[j] != -1 && optionCols[j] < len(record) {
				optionValue := getCSVValue(record, optionCols[j])
				if optionValue != "" {
					// 如果选项列表为空或当前索引匹配，添加选项
					if j == len(options) {
						options = append(options, optionValue)
					} else if j < len(options) {
						options[j] = optionValue
					}
				}
			} else if j < 12 { // 默认位置：从第3列开始（索引2）
				// 尝试使用默认位置
				colIndex := j + 2 // 题目(0) + 答案(1) + 选项从索引2开始
				if colIndex < len(record) {
					optionValue := getCSVValue(record, colIndex)
					if optionValue != "" {
						// 确保选项列表足够长
						for len(options) <= j {
							options = append(options, "")
						}
						options[j] = optionValue
					}
				}
			}
		}

		// 移除末尾的空选项
		for len(options) > 0 && options[len(options)-1] == "" {
			options = options[:len(options)-1]
		}

		if len(options) < 2 {
			continue // 至少需要2个选项
		}

		// 解析答案（支持单选和多选）
		answerStr := getCSVValue(record, answerCol)
		if answerStr == "" {
			continue
		}

		// 去除引号（CSV中可能被引号包围）
		answerStr = strings.Trim(answerStr, `"'`)

		// 检查是否是多选题（答案中包含逗号）
		var answer []int
		if strings.Contains(answerStr, ",") {
			// 多选题：解析多个答案
			answers := strings.Split(answerStr, ",")
			for _, ans := range answers {
				ans = strings.TrimSpace(ans)
				if ans == "" {
					continue
				}
				idx, err := parseAnswerMultiple(ans, len(options))
				if err != nil {
					return nil, fmt.Errorf("第 %d 行答案格式错误: %v", i+1, err)
				}
				answer = append(answer, idx)
			}
			// 去重并排序
			seen := make(map[int]bool)
			var uniqueAnswer []int
			for _, idx := range answer {
				if !seen[idx] && idx >= 0 && idx < len(options) {
					seen[idx] = true
					uniqueAnswer = append(uniqueAnswer, idx)
				}
			}
			answer = uniqueAnswer
		} else {
			// 单选题
			idx, err := parseAnswerMultiple(answerStr, len(options))
			if err != nil {
				return nil, fmt.Errorf("第 %d 行答案格式错误: %v", i+1, err)
			}
			answer = []int{idx}
		}

		if len(answer) == 0 {
			continue
		}

		explanation := ""
		if explanationCol != -1 {
			explanation = getCSVValue(record, explanationCol)
		}

		isMultiple := len(answer) > 1
		questions = append(questions, Question{
			Question:    question,
			Options:     options,
			Answer:      answer,
			IsMultiple:  isMultiple,
			Explanation: explanation,
		})
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("未找到有效的题目数据")
	}

	return questions, nil
}

// 获取题库题目
func getBankQuestions(c *gin.Context) {
	bankID := c.Param("id")
	userID := c.GetString("userID")

	// 检查题库是否属于当前用户
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM question_banks WHERE id = ? AND user_id = ?)", bankID, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或无权访问"})
		return
	}

	rows, err := db.Query("SELECT id, question, options, answer, is_multiple, explanation FROM questions WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
		return
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		var optionsJSON string
		var answerJSON string
		var isMultiple bool
		err := rows.Scan(&q.ID, &q.Question, &optionsJSON, &answerJSON, &isMultiple, &q.Explanation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据解析失败"})
			return
		}

		// 解析选项JSON
		err = json.Unmarshal([]byte(optionsJSON), &q.Options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "选项解析失败"})
			return
		}

		// 解析答案JSON（支持数组格式）
		err = json.Unmarshal([]byte(answerJSON), &q.Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "答案解析失败"})
			return
		}

		// 设置是否为多选题
		q.IsMultiple = isMultiple

		questions = append(questions, q)
	}

	c.JSON(http.StatusOK, questions)
}

// 添加题目
func createQuestion(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		BankID      string   `json:"bank_id" binding:"required"`
		Question    string   `json:"question" binding:"required"`
		Options     []string `json:"options" binding:"required"`
		Answer      []int    `json:"answer" binding:"required"` // 支持多选
		Explanation string   `json:"explanation"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查题库是否属于当前用户
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM question_banks WHERE id = ? AND user_id = ?)", req.BankID, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或无权访问"})
		return
	}

	// 验证选项数量（最多10个）
	if len(req.Options) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("选项数量超过10个（当前：%d）", len(req.Options))})
		return
	}

	// 验证答案
	if len(req.Answer) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "答案不能为空"})
		return
	}
	for _, ansIdx := range req.Answer {
		if ansIdx < 0 || ansIdx >= len(req.Options) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "答案索引超出选项范围"})
			return
		}
	}

	// 序列化选项和答案
	optionsJSON, err := json.Marshal(req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "选项序列化失败"})
		return
	}

	answerJSON, err := json.Marshal(req.Answer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "答案序列化失败"})
		return
	}

	// 判断是否为多选题
	isMultiple := len(req.Answer) > 1

	// 插入题目
	questionID := generateUUID()
	_, err = db.Exec("INSERT INTO questions (id, bank_id, question, options, answer, is_multiple, explanation) VALUES (?, ?, ?, ?, ?, ?, ?)",
		questionID, req.BankID, req.Question, string(optionsJSON), string(answerJSON), isMultiple, req.Explanation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建题目失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      questionID,
		"message": "题目创建成功",
	})
}

// 更新题目
func updateQuestion(c *gin.Context) {
	userID := c.GetString("userID")
	questionID := c.Param("id")

	var req struct {
		Question    string   `json:"question" binding:"required"`
		Options     []string `json:"options" binding:"required"`
		Answer      []int    `json:"answer" binding:"required"` // 支持多选
		Explanation string   `json:"explanation"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证选项数量（最多10个）
	if len(req.Options) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("选项数量超过10个（当前：%d）", len(req.Options))})
		return
	}

	// 验证答案
	if len(req.Answer) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "答案不能为空"})
		return
	}
	for _, ansIdx := range req.Answer {
		if ansIdx < 0 || ansIdx >= len(req.Options) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "答案索引超出选项范围"})
			return
		}
	}

	// 检查题目是否属于当前用户的题库
	var bankID string
	err := db.QueryRow("SELECT q.bank_id FROM questions q JOIN question_banks qb ON q.bank_id = qb.id WHERE q.id = ? AND qb.user_id = ?",
		questionID, userID).Scan(&bankID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在或无权修改"})
		return
	}

	// 序列化选项和答案
	optionsJSON, err := json.Marshal(req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "选项序列化失败"})
		return
	}

	answerJSON, err := json.Marshal(req.Answer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "答案序列化失败"})
		return
	}

	// 判断是否为多选题
	isMultiple := len(req.Answer) > 1

	// 更新题目
	_, err = db.Exec("UPDATE questions SET question = ?, options = ?, answer = ?, is_multiple = ?, explanation = ? WHERE id = ?",
		req.Question, string(optionsJSON), string(answerJSON), isMultiple, req.Explanation, questionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题目失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题目更新成功"})
}

// 删除题目
func deleteQuestion(c *gin.Context) {
	userID := c.GetString("userID")
	questionID := c.Param("id")

	// 检查题目是否属于当前用户的题库
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM questions q JOIN question_banks qb ON q.bank_id = qb.id WHERE q.id = ? AND qb.user_id = ?)",
		questionID, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在或无权删除"})
		return
	}

	// 删除题目
	result, err := db.Exec("DELETE FROM questions WHERE id = ?", questionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除题目失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题目删除成功"})
}

// 获取题目（用于错题练习等）
func getQuestions(c *gin.Context) {
	bankID := c.Query("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bank_id参数不能为空"})
		return
	}

	rows, err := db.Query("SELECT id, question, options, answer, explanation FROM questions WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
		return
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		var optionsJSON string
		err := rows.Scan(&q.ID, &q.Question, &optionsJSON, &q.Answer, &q.Explanation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据解析失败"})
			return
		}

		// 解析选项JSON
		err = json.Unmarshal([]byte(optionsJSON), &q.Options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "选项解析失败"})
			return
		}

		questions = append(questions, q)
	}

	c.JSON(http.StatusOK, questions)
}

// 删除错题
func deleteWrongQuestion(c *gin.Context) {
	userID := c.GetString("userID")
	questionID := c.Param("id")

	result, err := db.Exec("DELETE FROM wrong_questions WHERE id = ? AND user_id = ?", questionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除错题失败"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "错题不存在或已删除"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "错题删除成功"})
}

// 清空错题
func clearWrongQuestions(c *gin.Context) {
	userID := c.GetString("userID")

	_, err := db.Exec("DELETE FROM wrong_questions WHERE user_id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空错题失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "错题本已清空"})
}

// 辅助函数
func findColumnIndex(headers map[string]int, candidates []string) int {
	for _, candidate := range candidates {
		if index, exists := headers[candidate]; exists {
			return index
		}
	}
	return -1
}

func getExcelValue(record []string, colIndex int) string {
	if colIndex < 0 || colIndex >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[colIndex])
}

func getCSVValue(record []string, colIndex int) string {
	if colIndex < 0 || colIndex >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[colIndex])
}

// parseAnswerMultiple 解析答案（支持单选和多选，返回索引）
// 支持格式：A/B/C/D/E/F/G/H/I/J 或 1/2/3/4/5/6/7/8/9/10
func parseAnswerMultiple(answerStr string, optionCount int) (int, error) {
	answerStr = strings.TrimSpace(answerStr)
	
	// 解析字母答案（A-J）
	switch strings.ToUpper(answerStr) {
	case "A":
		if optionCount > 0 {
			return 0, nil
		}
		return -1, fmt.Errorf("选项A不存在")
	case "B":
		if optionCount > 1 {
			return 1, nil
		}
		return -1, fmt.Errorf("选项B不存在")
	case "C":
		if optionCount > 2 {
			return 2, nil
		}
		return -1, fmt.Errorf("选项C不存在")
	case "D":
		if optionCount > 3 {
			return 3, nil
		}
		return -1, fmt.Errorf("选项D不存在")
	case "E":
		if optionCount > 4 {
			return 4, nil
		}
		return -1, fmt.Errorf("选项E不存在")
	case "F":
		if optionCount > 5 {
			return 5, nil
		}
		return -1, fmt.Errorf("选项F不存在")
	case "G":
		if optionCount > 6 {
			return 6, nil
		}
		return -1, fmt.Errorf("选项G不存在")
	case "H":
		if optionCount > 7 {
			return 7, nil
		}
		return -1, fmt.Errorf("选项H不存在")
	case "I":
		if optionCount > 8 {
			return 8, nil
		}
		return -1, fmt.Errorf("选项I不存在")
	case "J":
		if optionCount > 9 {
			return 9, nil
		}
		return -1, fmt.Errorf("选项J不存在")
	}

	// 尝试解析数字答案
	if num, err := strconv.Atoi(answerStr); err == nil {
		if num >= 1 && num <= optionCount {
			return num - 1, nil
		}
		return -1, fmt.Errorf("答案数字超出选项范围")
	}

	return -1, fmt.Errorf("无效的答案格式: %s", answerStr)
}

func parseAnswer(answerStr string, optionCount int) (int, error) {
	answerStr = strings.TrimSpace(strings.ToUpper(answerStr))

	// 尝试解析字母答案
	switch answerStr {
	case "A":
		return 0, nil
	case "B":
		return 1, nil
	case "C":
		if optionCount > 2 {
			return 2, nil
		}
		return -1, fmt.Errorf("选项C不存在")
	case "D":
		if optionCount > 3 {
			return 3, nil
		}
		return -1, fmt.Errorf("选项D不存在")
	}

	// 尝试解析数字答案
	if num, err := strconv.Atoi(answerStr); err == nil {
		if num >= 1 && num <= optionCount {
			return num - 1, nil
		}
		return -1, fmt.Errorf("答案数字超出选项范围")
	}

	return -1, fmt.Errorf("无效的答案格式: %s", answerStr)
}

package main

import (
	"archive/zip"
	"bytes"
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
	"time"

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

	// 获取题目，按类型排序：判断题 -> 单选题 -> 多选题
	rows, err := db.Query("SELECT id, bank_id, question, options, answer, is_multiple, type, explanation FROM questions WHERE bank_id = ? ORDER BY CASE WHEN type = 'judgment' THEN 1 WHEN type = 'choice' AND is_multiple = 0 THEN 2 WHEN type = 'choice' AND is_multiple = 1 THEN 3 ELSE 4 END, id", bankID)
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
		var questionType string
		err := rows.Scan(&q.ID, &q.BankID, &q.Question, &optionsJSON, &answerJSON, &isMultiple, &questionType, &q.Explanation)
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
		q.Type = questionType
		if q.Type == "" {
			q.Type = "choice" // 默认为选择题
		}
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
		// 设置题目类型（默认为选择题）
		questionType := q.Type
		if questionType == "" {
			questionType = "choice"
		}
		
		_, err = tx.Exec("INSERT INTO questions (id, bank_id, question, options, answer, is_multiple, type, explanation) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			questionID, bankID, q.Question, string(optionsJSON), string(answerJSON), isMultiple, questionType, q.Explanation)
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

	// 获取用户名（用于创建用户专属目录）
	username := c.GetString("username")
	if username == "" {
		// 如果中间件没有设置username，从数据库查询
		var dbUsername string
		err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&dbUsername)
	if err != nil {
			log.Printf("无法获取用户名，使用userID作为目录名: %v", err)
			username = userID
		} else {
			username = dbUsername
		}
	}

	// 根据解析模式处理文件
	if parseMode == "ai" {
		// AI 自动分析模式：所有文件类型都通过 AI 解析
		log.Printf("使用AI模式解析文件: %s (用户: %s)", filename, username)
		questions, err = parseFileWithAI(file, header, ext, username)
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
		// 设置题目类型（默认为选择题）
		questionType := q.Type
		if questionType == "" {
			questionType = "choice"
		}
		
		_, err = tx.Exec("INSERT INTO questions (id, bank_id, question, options, answer, is_multiple, type, explanation) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			questionID, bankID, q.Question, string(optionsJSON), string(answerJSON), isMultiple, questionType, q.Explanation)
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

	// 添加格式说明行
	infoRow := sheet.AddRow()
	infoCell := infoRow.AddCell()
	infoCell.SetValue("格式说明：")
	infoCell = infoRow.AddCell()
	infoCell.SetValue("1. 第1列为题目内容（必填）")
	infoCell = infoRow.AddCell()
	infoCell.SetValue("2. 第2列为正确答案（必填）")
	infoCell = infoRow.AddCell()
	infoCell.SetValue("3. 选择题答案格式：A、B、C等字母，多选题用逗号分隔（如：A,B,C）")
	infoCell = infoRow.AddCell()
	infoCell.SetValue("4. 判断题答案格式：填写\"正确\"或\"错误\"（判断题不需要填写选项列）")
	infoCell = infoRow.AddCell()
	infoCell.SetValue("5. 选项A-J列：选择题的选项内容（最多10个选项）")
	infoCell = infoRow.AddCell()
	infoCell.SetValue("6. 最后一列为解析（可选）")
	
	// 添加空行
	sheet.AddRow()
	
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
		{"这是一道判断题？", "正确", "", "", "", "", "", "", "", "", "", "", "判断题的解析：这是正确的"},
		{"这是另一道判断题？", "错误", "", "", "", "", "", "", "", "", "", "", "判断题的解析：这是错误的"},
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

	// 添加格式说明（使用单引号或避免使用引号，避免CSV解析错误）
	csvContent += "格式说明：\n"
	csvContent += "1. 第1列为题目内容（必填）\n"
	csvContent += "2. 第2列为正确答案（必填）\n"
	csvContent += "3. 选择题答案格式：A、B、C等字母，多选题用逗号分隔（如：A,B,C）\n"
	csvContent += "4. 判断题答案格式：填写'正确'或'错误'（判断题不需要填写选项列）\n"
	csvContent += "5. 选项A-J列：选择题的选项内容（最多10个选项）\n"
	csvContent += "6. 最后一列为解析（可选）\n"
	csvContent += "\n"

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
	csvContent += "这是一道判断题？,正确,,,,,,,,,,,判断题的解析：这是正确的\n"
	csvContent += "这是另一道判断题？,错误,,,,,,,,,,,判断题的解析：这是错误的\n"

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
解析：这是多选题的解析

这是一道判断题？
答案：正确
解析：判断题的解析：这是正确的

这是另一道判断题？
答案：错误
解析：判断题的解析：这是错误的`

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
	// 读取文件内容，去除UTF-8 BOM
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取CSV文件失败: %v", err)
	}
	
	// 去除UTF-8 BOM（如果存在）
	if len(fileBytes) >= 3 && fileBytes[0] == 0xEF && fileBytes[1] == 0xBB && fileBytes[2] == 0xBF {
		fileBytes = fileBytes[3:]
	}
	
	// 创建字符串读取器
	reader := csv.NewReader(strings.NewReader(string(fileBytes)))
	// 设置ReuseRecord为true，允许字段数量不一致（用于处理格式说明行）
	reader.ReuseRecord = true
	// 设置FieldsPerRecord为负数，允许字段数量不一致
	reader.FieldsPerRecord = -1
	// 设置LazyQuotes为true，允许宽松的引号处理（允许未转义的引号）
	reader.LazyQuotes = true
	// 设置TrimLeadingSpace为true，自动去除字段前导空格
	reader.TrimLeadingSpace = true
	
	allRecords, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV文件读取失败: %v（可能是字段数量不一致或引号格式错误，请检查CSV文件格式）", err)
	}

	if len(allRecords) < 2 {
		return nil, fmt.Errorf("CSV文件数据不足")
	}

	// 过滤掉格式说明行和空行，找到真正的表头
	var records [][]string
	var headerFound bool
	var headerIndex int
	
	for i, record := range allRecords {
		// 检查是否是表头行（包含"题目"或"question"等关键词）
		firstField := strings.TrimSpace(record[0])
		if !headerFound && (firstField == "题目" || strings.ToLower(firstField) == "question" || 
			strings.Contains(firstField, "题目") || strings.Contains(strings.ToLower(firstField), "question")) {
			headerFound = true
			headerIndex = i
			records = append(records, record) // 添加表头
			continue
		}
		
		// 如果已经找到表头，添加数据行
		if headerFound && i > headerIndex {
			// 跳过空行（所有字段都为空）
			isEmpty := true
			for _, field := range record {
				if strings.TrimSpace(field) != "" {
					isEmpty = false
					break
				}
			}
			if !isEmpty {
				records = append(records, record)
			}
		}
	}
	
	// 如果没有找到表头，使用所有记录（向后兼容）
	if !headerFound {
		records = allRecords
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV文件数据不足（可能只包含格式说明，没有实际数据）")
	}

	return parseCSVData(records)
}

// parseFileWithAI 使用AI解析文件（支持PDF、DOC、DOCX、Excel、CSV等）
// 新版本：将文件拆分为多个分片，顺序处理，避免超时和输出截断
func parseFileWithAI(file multipart.File, header *multipart.FileHeader, ext string, username string) ([]Question, error) {
	// 检查文件类型是否支持
	supportedExts := []string{".pdf", ".doc", ".docx", ".xlsx", ".xls", ".csv", ".txt", ".ppt", ".pptx"}
	isSupported := false
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			isSupported = true
			break
		}
	}
	if !isSupported {
		return nil, fmt.Errorf("不支持的文件格式: %s，支持格式: pdf, doc, docx, xlsx, xls, csv, txt, ppt, pptx", ext)
	}

	// 重置文件指针到开头（因为可能已经被读取过）
	file.Seek(0, 0)

	// 记录开始时间
	startTime := time.Now()
	fileSize := header.Size
	fileFormat := strings.TrimPrefix(ext, ".")

	log.Printf("开始使用分片并发方式解析文件: %s (大小: %d 字节, 格式: %s)", header.Filename, fileSize, fileFormat)

	// 1. 提取文件文本内容
	textExtractStart := time.Now()
	text, err := extractTextFromFile(file, header, ext)
	if err != nil {
		return nil, fmt.Errorf("提取文件文本失败: %v", err)
	}
	textExtractDuration := time.Since(textExtractStart)
	log.Printf("文件文本提取完成，文本长度: %d 字符，耗时: %v", len(text), textExtractDuration)

	// 保存提取的文本到本地文件（用于检查）
	if saveErr := saveExtractedTextToFile(header.Filename, text, username); saveErr != nil {
		log.Printf("保存提取文本到文件失败: %v（不影响解析流程）", saveErr)
	}

	// 2. 智能分片（每片约10KB，尽量在换行或题目边界分片）
	chunks := splitTextIntoSmartChunks(text, 10*1024) // 10KB per chunk
	log.Printf("文件已拆分为 %d 个分片", len(chunks))

	// 3. 顺序处理所有分片
	aiParseStart := time.Now()
	allQuestions, totalStats, err := processChunksSequentially(chunks, header.Filename, username)
	if err != nil {
		return nil, fmt.Errorf("顺序处理分片失败: %v", err)
	}
	aiParseDuration := time.Since(aiParseStart)

	// 4. 输出指标日志
	totalDuration := time.Since(startTime)
	modelName := getTencentCloudModel()
	log.Printf("【解析指标】文件: %s | 大小: %d 字节 | 格式: %s | 模型: %s | 文本提取耗时: %v | AI解析耗时: %v | 分片数: %d | 总题目: %d | 有效题目: %d | 报错题目: %d | 总耗时: %v",
		header.Filename, fileSize, fileFormat, modelName, textExtractDuration, aiParseDuration, len(chunks),
		totalStats.TotalQuestions, totalStats.ValidQuestions, totalStats.ErrorQuestions, totalDuration)

	return allQuestions, nil
}

// extractTextFromFile 从文件中提取文本内容
func extractTextFromFile(file multipart.File, header *multipart.FileHeader, ext string) (string, error) {
	file.Seek(0, 0)

	switch ext {
	case ".pdf":
		return parsePDFFile(file, header)
	case ".doc", ".docx":
		return parseDOCFile(file, header)
	case ".xlsx", ".xls":
		return parseExcelFileAsText(file)
	case ".csv":
		return parseCSVFileAsText(file)
	case ".txt":
		content, err := io.ReadAll(file)
		if err != nil {
			return "", fmt.Errorf("读取文本文件失败: %v", err)
		}
		return string(content), nil
	default:
		// 对于其他格式，尝试使用现有的解析方法
		return parseDOCFile(file, header)
	}
}

// splitTextIntoSmartChunks 智能分片：尽量在换行或题目边界分片
func splitTextIntoSmartChunks(text string, targetChunkSize int) []string {
	if len(text) <= targetChunkSize {
		return []string{text}
	}

	var chunks []string
	currentPos := 0
	textLen := len(text)

	for currentPos < textLen {
		remaining := textLen - currentPos
		if remaining <= targetChunkSize {
			// 剩余内容不足一个分片，直接添加
			chunks = append(chunks, text[currentPos:])
			break
		}

		// 计算当前分片的结束位置
		endPos := currentPos + targetChunkSize

		// 如果还没到文件末尾，尝试在合适的位置分片
		if endPos < textLen {
			// 优先在换行符处分片
			lastNewline := strings.LastIndex(text[currentPos:endPos], "\n")
			threshold70 := int(float64(targetChunkSize) * 0.7)
			if lastNewline > threshold70 { // 如果换行符在分片的70%之后，使用它
				endPos = currentPos + lastNewline + 1
			} else {
				// 尝试在题目结束处分片（查找常见的题目结束标记）
				// 查找 "答案"、"正确答案"、"解析" 等关键词后的换行
				searchEnd := endPos
				if searchEnd > textLen {
					searchEnd = textLen
				}
				searchText := text[currentPos:searchEnd]

				// 查找题目结束标记（答案、解析等关键词后的换行）
				questionEndMarkers := []string{"\n答案", "\n正确答案", "\n解析", "\n【答案", "\n【解析"}
				bestEndPos := -1
				threshold60 := int(float64(targetChunkSize) * 0.6)
				for _, marker := range questionEndMarkers {
					idx := strings.LastIndex(searchText, marker)
					if idx > threshold60 && idx > bestEndPos {
						// 找到标记后的第一个换行
						afterMarker := currentPos + idx + len(marker)
						nextNewline := strings.Index(text[afterMarker:], "\n")
						if nextNewline > 0 {
							bestEndPos = afterMarker + nextNewline + 1
						}
					}
				}

				if bestEndPos > currentPos {
					endPos = bestEndPos
				} else {
					// 如果找不到合适的分片点，在最近的空格或标点处分片
					threshold80 := int(float64(targetChunkSize) * 0.8)
					textRunes := []rune(text)
					for i := endPos - 1; i > currentPos+threshold80; i-- {
						if i < len(textRunes) {
							r := textRunes[i]
							if r == ' ' || r == '。' || r == '.' || r == '；' || r == ';' {
								endPos = i + 1
								break
							}
						}
					}
				}
			}
		}

		chunks = append(chunks, text[currentPos:endPos])
		currentPos = endPos
	}

	return chunks
}

// processChunksSequentially 顺序处理所有分片
func processChunksSequentially(chunks []string, filename string, username string) ([]Question, *ParseAIResponseStats, error) {
	if len(chunks) == 0 {
		return nil, nil, fmt.Errorf("没有分片需要处理")
	}

	// 构建提示词模板
	promptTemplate := buildPromptForChunk()

	// 顺序处理所有分片
	log.Printf("开始顺序处理 %d 个分片...", len(chunks))
	var allQuestions []Question
	totalStats := &ParseAIResponseStats{}
	errors := []error{}

	for i, chunk := range chunks {
		log.Printf("开始处理分片 %d/%d (长度: %d 字符)", i+1, len(chunks), len(chunk))

		// 保存分片的原始文本内容
		chunkBaseName := fmt.Sprintf("%s-chunk%d", filename, i+1)
		if saveErr := saveChunkTextToFile(chunkBaseName, chunk, username); saveErr != nil {
			log.Printf("保存分片 %d 原始文本失败: %v（不影响解析流程）", i+1, saveErr)
		}

		// 为每个分片构建完整的提示词
		prompt := promptTemplate + "\n\n" + chunk

		// 调用AI API（不使用FileID，直接使用文本）
		responseText, err := callTencentCloudAPI(prompt, nil)
		if err != nil {
			log.Printf("分片 %d AI API调用失败: %v", i+1, err)
			errors = append(errors, fmt.Errorf("分片 %d 处理失败: %v", i+1, err))
			continue
		}

		// 保存每个分片的AI解析响应
		if saveErr := saveChunkAIResponseToFile(chunkBaseName, responseText, username); saveErr != nil {
			log.Printf("保存分片 %d AI解析结果失败: %v（不影响解析流程）", i+1, saveErr)
		}

		// 解析响应
		questions, stats, errorQuestions, err := parseAIResponse(responseText)
		if err != nil {
			log.Printf("分片 %d 解析失败: %v", i+1, err)
			errors = append(errors, fmt.Errorf("分片 %d 解析失败: %v", i+1, err))
			continue
		}

		// 保存报错的题目到文件
		if len(errorQuestions) > 0 {
			chunkBaseName := fmt.Sprintf("%s-chunk%d", filename, i+1)
			if saveErr := saveErrorQuestionsToFile(chunkBaseName, errorQuestions, username); saveErr != nil {
				log.Printf("保存分片 %d 错误题目失败: %v（不影响解析流程）", i+1, saveErr)
			}
		}

		log.Printf("分片 %d 处理完成，识别出 %d 道题目（有效: %d, 错误: %d）",
			i+1, stats.TotalQuestions, stats.ValidQuestions, stats.ErrorQuestions)

		// 合并题目
		allQuestions = append(allQuestions, questions...)

		// 累计统计
		totalStats.TotalQuestions += stats.TotalQuestions
		totalStats.ValidQuestions += stats.ValidQuestions
		totalStats.ErrorQuestions += stats.ErrorQuestions
	}

	// 如果有部分分片失败，记录警告但继续处理
	if len(errors) > 0 {
		log.Printf("警告: %d 个分片处理失败，但已处理 %d 个分片成功", len(errors), len(chunks)-len(errors))
	}

	// 去重（基于题目内容）
	uniqueQuestions := deduplicateQuestions(allQuestions)
	log.Printf("合并后共 %d 道题目，去重后 %d 道题目", len(allQuestions), len(uniqueQuestions))

	if len(uniqueQuestions) == 0 {
		return nil, totalStats, fmt.Errorf("未能识别出有效的题目")
	}

	return uniqueQuestions, totalStats, nil
}

// buildPromptForChunk 构建分片处理的提示词
func buildPromptForChunk() string {
	return "你是一个专业的题目解析助手。请从以下文本中识别出所有题目（包括选择题和判断题），并严格按照JSON格式返回。\n\n" +
		"【核心要求 - 必须严格遵守】\n" +
		"1. 你的回复必须是且只能是纯JSON格式，不能包含任何其他内容\n" +
		"2. 禁止使用markdown代码块标记（如```json或```），直接返回JSON对象\n" +
		"3. 禁止在JSON前后添加任何文字说明、注释、解释或标点符号\n" +
		"4. 禁止在JSON内部添加注释或说明性文字\n" +
		"5. 你的整个回复必须能够直接被JSON.parse()或json.Unmarshal()解析成功\n" +
		"6. 回复必须以{开始，以}结束，中间不能有任何非JSON内容\n\n" +
		"【JSON格式规范 - 严格遵守】\n" +
		"1. 所有字符串中的特殊字符必须正确转义\n" +
		"2. 所有字符串值必须用双引号包裹，不能用单引号\n" +
		"3. 数组和对象必须正确闭合，括号和花括号必须匹配\n" +
		"4. 数组元素之间用逗号分隔，最后一个元素后不能有逗号\n" +
		"5. JSON对象的所有键必须用双引号包裹\n" +
		"6. 每个字段之间必须用逗号分隔（最后一个字段除外）\n\n" +
		"【题目识别要求】\n" +
		"1. 识别所有题目，包括选择题（单选题和多选题）和判断题\n" +
		"2. 选择题：包含题目内容、选项（至少2个，最多10个）、正确答案、解析（如果有）\n" +
		"3. 判断题：包含题目内容、答案（正确/错误）、解析（如果有）\n" +
		"4. 如果文本中没有找到题目，返回 {\"questions\": []}\n" +
		"5. 支持各种格式的题目，不限于固定格式\n\n" +
		"【题目类型判断 - 非常重要】\n" +
		"1. 判断题识别规则（优先级最高）：\n" +
		"   - 如果文本中明确标注了\"判断题\"、\"（三）判断题\"等字样，该部分所有题目都是判断题\n" +
		"   - 如果题目末尾有\"（  ）\"、\"（ ）\"、\"(  )\"、\"( )\"等空白括号，通常是判断题\n" +
		"   - 如果答案格式是\"（×）\"、\"（√）\"、\"（X）\"、\"（V）\"、\"答案：（×）\"、\"答案：（√）\"等，这是判断题\n" +
		"   - 如果答案直接是\"正确\"、\"错误\"、\"TRUE\"、\"FALSE\"、\"T\"、\"F\"、\"√\"、\"×\"、\"对\"、\"错\"等，这是判断题\n" +
		"   - 判断题不需要构造选项，只需要识别题目内容和答案（正确/错误）\n" +
		"   - 判断题的options字段必须固定为：[\"错误\", \"正确\"]\n" +
		"   - 判断题的answer字段必须是：[\"正确\"] 或 [\"错误\"]\n" +
		"2. 选择题特征：\n" +
		"   - 题目有明确的选项（A、B、C、D等），答案对应选项字母\n" +
		"   - 选择题必须至少有2个选项，最多10个选项\n" +
		"3. 重要提醒：\n" +
		"   - 如果题目是判断题格式（有空白括号或答案格式为×/√），绝对不要将其转换为选择题\n" +
		"   - 判断题不要自己构造选项，不要将题目内容改写成选择题形式\n" +
		"   - 判断题保持原题目内容，只提取题目文本和答案（正确/错误）\n\n" +
		"【答案格式要求】\n" +
		"1. 选择题：统一使用答案数组格式：[\"A\"] 或 [\"A\", \"B\", \"C\"]\n" +
		"   单选题：返回单个元素的数组，如 [\"A\"] 或 [\"B\"]\n" +
		"   多选题：返回多个元素的数组，如 [\"A\", \"B\", \"C\"]\n" +
		"   答案使用字母格式（A, B, C, D, E, F, G, H, I, J），对应选项的顺序\n" +
		"2. 判断题：统一使用答案数组格式：[\"正确\"] 或 [\"错误\"]\n" +
		"   答案必须是 [\"正确\"] 或 [\"错误\"]，不能使用其他格式\n\n" +
		"【返回格式示例】\n" +
		"选择题示例：\n" +
		"{\n" +
		"  \"questions\": [\n" +
		"    {\n" +
		"      \"question\": \"题目内容？\",\n" +
		"      \"options\": [\"选项A\", \"选项B\", \"选项C\", \"选项D\"],\n" +
		"      \"answer\": [\"A\"],\n" +
		"      \"explanation\": \"解析内容（可选）\"\n" +
		"    }\n" +
		"  ]\n" +
		"}\n\n" +
		"判断题示例（注意：保持原题目格式，不要改写成选择题）：\n" +
		"原始文本：\"1.对公存款周计划是指中国光大银行以7天为一个周期，为对公签约客户循环存款的本外币存款业务。（  ）\\n答案：（×）对公存款周计划是指中国光大银行以7天为一个周期，为对公签约客户循环存款的人民币存款业务。\"\n" +
		"正确识别为：\n" +
		"{\n" +
		"  \"questions\": [\n" +
		"    {\n" +
		"      \"question\": \"对公存款周计划是指中国光大银行以7天为一个周期，为对公签约客户循环存款的本外币存款业务。\",\n" +
		"      \"options\": [\"错误\", \"正确\"],\n" +
		"      \"answer\": [\"错误\"],\n" +
		"      \"explanation\": \"对公存款周计划是指中国光大银行以7天为一个周期，为对公签约客户循环存款的人民币存款业务。\"\n" +
		"    }\n" +
		"  ]\n" +
		"}\n\n" +
		"原始文本：\"2.资信证明业务分单项/多项资信证明两种。（  ）\\n答案：（√）\"\n" +
		"正确识别为：\n" +
		"{\n" +
		"  \"questions\": [\n" +
		"    {\n" +
		"      \"question\": \"资信证明业务分单项/多项资信证明两种。\",\n" +
		"      \"options\": [\"错误\", \"正确\"],\n" +
		"      \"answer\": [\"正确\"],\n" +
		"      \"explanation\": \"\"\n" +
		"    }\n" +
		"  ]\n" +
		"}\n\n" +
		"错误示例（不要这样做）：\n" +
		"{\n" +
		"  \"questions\": [\n" +
		"    {\n" +
		"      \"question\": \"对公存款周计划是指什么？\",\n" +
		"      \"options\": [\"本外币存款业务\", \"人民币存款业务\"],\n" +
		"      \"answer\": [\"B\"]\n" +
		"    }\n" +
		"  ]\n" +
		"}\n" +
		"（这是错误的！判断题不能改写成选择题形式）\n\n" +
		"请仔细阅读以下文本内容，识别出所有题目（包括选择题和判断题），并严格按照上述要求返回JSON格式。\n\n"
}

// deduplicateQuestions 基于题目内容去重
func deduplicateQuestions(questions []Question) []Question {
	seen := make(map[string]bool)
	var unique []Question

	for _, q := range questions {
		// 使用题目的规范化内容作为唯一标识
		normalized := strings.TrimSpace(strings.ToLower(q.Question))
		if !seen[normalized] && normalized != "" {
			seen[normalized] = true
			unique = append(unique, q)
		}
	}

	return unique
}

// saveAIResponseToFile 将AI返回的原始数据保存到本地文件（按用户名分目录）
func saveAIResponseToFile(originalFilename string, responseText string, username string) error {
	// 获取模型名称
	modelName := getTencentCloudModel()
	if modelName == "" {
		modelName = "unknown"
	}

	// 构建文件名：原始文件名-模型名-解析.json
	// 移除原始文件名的扩展名
	ext := filepath.Ext(originalFilename)
	baseName := strings.TrimSuffix(originalFilename, ext)

	// 清理文件名中的特殊字符，避免文件系统问题
	baseName = strings.ReplaceAll(baseName, "/", "_")
	baseName = strings.ReplaceAll(baseName, "\\", "_")
	baseName = strings.ReplaceAll(baseName, ":", "_")
	baseName = strings.ReplaceAll(baseName, "*", "_")
	baseName = strings.ReplaceAll(baseName, "?", "_")
	baseName = strings.ReplaceAll(baseName, "\"", "_")
	baseName = strings.ReplaceAll(baseName, "<", "_")
	baseName = strings.ReplaceAll(baseName, ">", "_")
	baseName = strings.ReplaceAll(baseName, "|", "_")

	// 清理用户名中的特殊字符，避免文件系统问题
	safeUsername := strings.ReplaceAll(username, "/", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\\", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ":", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "*", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "?", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\"", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "<", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ">", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "|", "_")

	outputFilename := fmt.Sprintf("%s-%s-解析.json", baseName, modelName)

	// 创建用户专属的输出目录（如果不存在）
	outputDir := filepath.Join("ai-responses", safeUsername)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建用户输出目录失败: %v", err)
	}

	// 构建完整文件路径
	outputPath := filepath.Join(outputDir, outputFilename)

	// 写入文件
	err := os.WriteFile(outputPath, []byte(responseText), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	log.Printf("AI响应已保存到文件: %s", outputPath)
	return nil
}

// saveExtractedTextToFile 将提取的文本保存到本地文件（用于检查提取结果）
func saveExtractedTextToFile(originalFilename string, extractedText string, username string) error {
	// 移除原始文件名的扩展名
	ext := filepath.Ext(originalFilename)
	baseName := strings.TrimSuffix(originalFilename, ext)

	// 清理文件名中的特殊字符，避免文件系统问题
	baseName = strings.ReplaceAll(baseName, "/", "_")
	baseName = strings.ReplaceAll(baseName, "\\", "_")
	baseName = strings.ReplaceAll(baseName, ":", "_")
	baseName = strings.ReplaceAll(baseName, "*", "_")
	baseName = strings.ReplaceAll(baseName, "?", "_")
	baseName = strings.ReplaceAll(baseName, "\"", "_")
	baseName = strings.ReplaceAll(baseName, "<", "_")
	baseName = strings.ReplaceAll(baseName, ">", "_")
	baseName = strings.ReplaceAll(baseName, "|", "_")

	// 清理用户名中的特殊字符，避免文件系统问题
	safeUsername := strings.ReplaceAll(username, "/", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\\", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ":", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "*", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "?", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\"", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "<", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ">", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "|", "_")

	outputFilename := fmt.Sprintf("%s-提取文本.txt", baseName)

	// 创建用户专属的输出目录（如果不存在）
	outputDir := filepath.Join("ai-responses", safeUsername)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建用户输出目录失败: %v", err)
	}

	// 构建完整文件路径
	outputPath := filepath.Join(outputDir, outputFilename)

	// 写入文件
	err := os.WriteFile(outputPath, []byte(extractedText), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	log.Printf("提取文本已保存到文件: %s (长度: %d 字符)", outputPath, len(extractedText))
	return nil
}

// saveChunkTextToFile 保存分片的原始文本内容
func saveChunkTextToFile(chunkBaseName string, chunkText string, username string) error {
	// 清理文件名中的特殊字符，避免文件系统问题
	baseName := strings.ReplaceAll(chunkBaseName, "/", "_")
	baseName = strings.ReplaceAll(baseName, "\\", "_")
	baseName = strings.ReplaceAll(baseName, ":", "_")
	baseName = strings.ReplaceAll(baseName, "*", "_")
	baseName = strings.ReplaceAll(baseName, "?", "_")
	baseName = strings.ReplaceAll(baseName, "\"", "_")
	baseName = strings.ReplaceAll(baseName, "<", "_")
	baseName = strings.ReplaceAll(baseName, ">", "_")
	baseName = strings.ReplaceAll(baseName, "|", "_")

	// 清理用户名中的特殊字符，避免文件系统问题
	safeUsername := strings.ReplaceAll(username, "/", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\\", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ":", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "*", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "?", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\"", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "<", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ">", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "|", "_")

	outputFilename := fmt.Sprintf("%s-原始文本.txt", baseName)

	// 创建用户专属的输出目录（如果不存在）
	outputDir := filepath.Join("ai-responses", safeUsername)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建用户输出目录失败: %v", err)
	}

	// 构建完整文件路径
	outputPath := filepath.Join(outputDir, outputFilename)

	// 写入文件
	err := os.WriteFile(outputPath, []byte(chunkText), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	log.Printf("分片原始文本已保存到文件: %s (长度: %d 字符)", outputPath, len(chunkText))
	return nil
}

// saveChunkAIResponseToFile 保存分片的AI解析结果
func saveChunkAIResponseToFile(chunkBaseName string, aiResponse string, username string) error {
	// 获取模型名称
	modelName := getTencentCloudModel()
	if modelName == "" {
		modelName = "unknown"
	}

	// 清理文件名中的特殊字符，避免文件系统问题
	baseName := strings.ReplaceAll(chunkBaseName, "/", "_")
	baseName = strings.ReplaceAll(baseName, "\\", "_")
	baseName = strings.ReplaceAll(baseName, ":", "_")
	baseName = strings.ReplaceAll(baseName, "*", "_")
	baseName = strings.ReplaceAll(baseName, "?", "_")
	baseName = strings.ReplaceAll(baseName, "\"", "_")
	baseName = strings.ReplaceAll(baseName, "<", "_")
	baseName = strings.ReplaceAll(baseName, ">", "_")
	baseName = strings.ReplaceAll(baseName, "|", "_")

	// 清理用户名中的特殊字符，避免文件系统问题
	safeUsername := strings.ReplaceAll(username, "/", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\\", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ":", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "*", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "?", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\"", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "<", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ">", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "|", "_")

	outputFilename := fmt.Sprintf("%s-AI解析-%s.json", baseName, modelName)

	// 创建用户专属的输出目录（如果不存在）
	outputDir := filepath.Join("ai-responses", safeUsername)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建用户输出目录失败: %v", err)
	}

	// 构建完整文件路径
	outputPath := filepath.Join(outputDir, outputFilename)

	// 写入文件
	err := os.WriteFile(outputPath, []byte(aiResponse), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	log.Printf("分片AI解析结果已保存到文件: %s (长度: %d 字符)", outputPath, len(aiResponse))
	return nil
}

// saveErrorQuestionsToFile 保存报错的题目到本地文件
func saveErrorQuestionsToFile(chunkBaseName string, errorQuestions []ErrorQuestion, username string) error {
	// 清理文件名中的特殊字符，避免文件系统问题
	baseName := strings.ReplaceAll(chunkBaseName, "/", "_")
	baseName = strings.ReplaceAll(baseName, "\\", "_")
	baseName = strings.ReplaceAll(baseName, ":", "_")
	baseName = strings.ReplaceAll(baseName, "*", "_")
	baseName = strings.ReplaceAll(baseName, "?", "_")
	baseName = strings.ReplaceAll(baseName, "\"", "_")
	baseName = strings.ReplaceAll(baseName, "<", "_")
	baseName = strings.ReplaceAll(baseName, ">", "_")
	baseName = strings.ReplaceAll(baseName, "|", "_")

	// 清理用户名中的特殊字符，避免文件系统问题
	safeUsername := strings.ReplaceAll(username, "/", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\\", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ":", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "*", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "?", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "\"", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "<", "_")
	safeUsername = strings.ReplaceAll(safeUsername, ">", "_")
	safeUsername = strings.ReplaceAll(safeUsername, "|", "_")

	outputFilename := fmt.Sprintf("%s-报错题目.json", baseName)

	// 创建用户专属的输出目录（如果不存在）
	outputDir := filepath.Join("ai-responses", safeUsername)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建用户输出目录失败: %v", err)
	}

	// 构建完整文件路径
	outputPath := filepath.Join(outputDir, outputFilename)

	// 将错误题目列表转换为JSON
	jsonData, err := json.MarshalIndent(errorQuestions, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化错误题目失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile(outputPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	log.Printf("报错题目已保存到文件: %s (共 %d 道)", outputPath, len(errorQuestions))
	return nil
}

// getTencentCloudModel 函数已在 ai_service.go 中定义

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
	// 读取整个文件内容以检测和移除UTF-8 BOM
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取CSV文件失败: %v", err)
	}

	// 检测并移除UTF-8 BOM
	if len(fileContent) >= 3 && fileContent[0] == 0xEF && fileContent[1] == 0xBB && fileContent[2] == 0xBF {
		fileContent = fileContent[3:]
	}

	// 创建CSV reader
	reader := csv.NewReader(bytes.NewReader(fileContent))
	// 设置宽松的引号处理
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
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

// parseDOCXFile 解析DOC/DOCX文件（提取文本，用于AI分析）
func parseDOCXFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".docx" && ext != ".doc" {
		return "", fmt.Errorf("不支持的文件格式: %s，仅支持DOC和DOCX格式", ext)
	}

	// 读取文件内容到临时文件（根据实际扩展名创建临时文件）
	tempFile, err := os.CreateTemp("", "doc-*"+ext)
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

	// 重新打开文件
	tempFileReader, err := os.Open(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("打开临时文件失败: %v", err)
	}
	defer tempFileReader.Close()

	// 调用parseDOCFile函数（在pdf_doc_parser.go中定义）
	return parseDOCFile(tempFileReader, header)
}

// parseDOCXFileFormat 解析DOCX文件的固定格式（用于固定格式解析）
func parseDOCXFileFormat(file multipart.File, header *multipart.FileHeader) ([]Question, error) {
	// 先提取文本
	text, err := parseDOCXFile(file, header)
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

			// 检查是否为简答题，如果是则跳过
			questionText := strings.TrimSpace(currentQuestion.Question)
			questionTextLower := strings.ToLower(questionText)
			
			// 简答题关键词
			essayKeywords := []string{"简述", "说明", "论述", "分析", "解释", "描述", "阐述", "阐述", "评述", "评价", "比较", "对比", "总结", "概述", "介绍", "说明原因", "说明理由", "说明方法", "说明步骤"}
			isEssayQuestion := false
			for _, keyword := range essayKeywords {
				if strings.Contains(questionTextLower, keyword) {
					isEssayQuestion = true
					break
				}
			}
			
			// 如果题目包含简答题关键词，且没有选项，则跳过
			if isEssayQuestion && len(currentQuestion.Options) == 0 {
				log.Printf("跳过简答题: %s", func() string {
					preview := questionText
					if len(preview) > 50 {
						return preview[:50] + "..."
					}
					return preview
				}())
				continue
			}
			
			// 如果答案长度较长（超过30个字符），且没有选项，则可能是简答题，跳过
			if len(answerStr) > 30 && len(currentQuestion.Options) == 0 {
				log.Printf("跳过简答题（答案过长且无选项）: %s", func() string {
					preview := questionText
					if len(preview) > 50 {
						return preview[:50] + "..."
					}
					return preview
				}())
				continue
			}

			// 判断题目类型：判断题或选择题
			answerStrUpper := strings.ToUpper(answerStr)
			isJudgment := answerStrUpper == "正确" || answerStrUpper == "错误" || 
				answerStrUpper == "TRUE" || answerStrUpper == "FALSE" ||
				answerStrUpper == "T" || answerStrUpper == "F" ||
				answerStrUpper == "√" || answerStrUpper == "×" ||
				answerStrUpper == "对" || answerStrUpper == "错" ||
				answerStrUpper == "是" || answerStrUpper == "否"

			var answerIndices []int
			var questionType string

			if isJudgment {
				// 判断题：选项固定为["错误", "正确"]，答案：0=错误，1=正确
				questionType = "judgment"
				currentQuestion.Options = []string{"错误", "正确"}
				
				if answerStrUpper == "正确" || answerStrUpper == "TRUE" || answerStrUpper == "T" || 
					answerStrUpper == "√" || answerStrUpper == "对" || answerStrUpper == "是" {
					answerIndices = []int{1} // 正确
				} else {
					answerIndices = []int{0} // 错误
				}
			} else {
				// 选择题：解析答案（支持单选和多选）
				questionType = "choice"
				answers := strings.Split(answerStr, ",")
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
			}

			log.Printf("解析答案成功: 第 %d 行，题目类型: %s，答案索引: %v，选项数: %d",
				i+1, questionType, answerIndices, len(currentQuestion.Options))
			currentQuestion.Answer = answerIndices
			currentQuestion.Type = questionType
			currentQuestion.IsMultiple = len(answerIndices) > 1 && questionType == "choice"
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
				Type:        "choice", // 默认为选择题
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

		// 解析答案（支持单选和多选）
		answerStr := getExcelValue(row, answerCol)
		if answerStr == "" {
			continue
		}
		answerStr = strings.TrimSpace(answerStr)

		// 检查是否为简答题，如果是则跳过
		questionText := strings.TrimSpace(question)
		questionTextLower := strings.ToLower(questionText)
		
		// 简答题关键词
		essayKeywords := []string{"简述", "说明", "论述", "分析", "解释", "描述", "阐述", "阐述", "评述", "评价", "比较", "对比", "总结", "概述", "介绍", "说明原因", "说明理由", "说明方法", "说明步骤"}
		isEssayQuestion := false
		for _, keyword := range essayKeywords {
			if strings.Contains(questionTextLower, keyword) {
				isEssayQuestion = true
				break
			}
		}
		
		// 如果题目包含简答题关键词，且没有选项，则跳过
		if isEssayQuestion && len(options) == 0 {
			log.Printf("跳过简答题: %s", func() string {
				preview := questionText
				if len(preview) > 50 {
					return preview[:50] + "..."
				}
				return preview
			}())
			continue
		}
		
		// 如果答案长度较长（超过30个字符），且没有选项，则可能是简答题，跳过
		if len(answerStr) > 30 && len(options) == 0 {
			log.Printf("跳过简答题（答案过长且无选项）: %s", func() string {
				preview := questionText
				if len(preview) > 50 {
					return preview[:50] + "..."
				}
				return preview
			}())
			continue
		}

		// 判断题目类型：判断题或选择题
		answerStrUpper := strings.ToUpper(answerStr)
		isJudgment := answerStrUpper == "正确" || answerStrUpper == "错误" || 
			answerStrUpper == "TRUE" || answerStrUpper == "FALSE" ||
			answerStrUpper == "T" || answerStrUpper == "F" ||
			answerStrUpper == "√" || answerStrUpper == "×" ||
			answerStrUpper == "对" || answerStrUpper == "错" ||
			answerStrUpper == "是" || answerStrUpper == "否"

		var questionType string
		var answer []int
		var finalOptions []string

		if isJudgment {
			// 判断题：选项固定为["错误", "正确"]，答案：0=错误，1=正确
			questionType = "judgment"
			finalOptions = []string{"错误", "正确"}
			
			if answerStrUpper == "正确" || answerStrUpper == "TRUE" || answerStrUpper == "T" || 
				answerStrUpper == "√" || answerStrUpper == "对" || answerStrUpper == "是" {
				answer = []int{1} // 正确
			} else {
				answer = []int{0} // 错误
			}
		} else {
			// 选择题：至少需要2个选项
			if len(options) < 2 {
				continue
			}
			
			questionType = "choice"
			finalOptions = options
			
			// 检查是否是多选题（答案中包含逗号）
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
		}

		if len(answer) == 0 {
			continue
		}

		explanation := ""
		if explanationCol != -1 {
			explanation = getExcelValue(row, explanationCol)
		}

		isMultiple := len(answer) > 1 && questionType == "choice"
		questions = append(questions, Question{
			Question:    question,
			Options:     finalOptions,
			Answer:      answer,
			IsMultiple:  isMultiple,
			Type:        questionType,
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
	// 从选项C开始查找（索引2开始）
	for i := 2; i < 10; i++ {
		// 如果表头中有定义，使用定义的位置；否则使用默认位置（索引 i+2，因为前面有题目列0和答案列1）
		optionLabel := string(rune('A' + i))
		colNames := []string{"选项" + optionLabel, optionLabel, "option" + optionLabel, "选择" + optionLabel}
		if foundCol := findColumnIndex(headers, colNames); foundCol != -1 {
			optionCols[i] = foundCol
		} else {
			// 默认位置：第3列开始（索引2），依次是选项A、B、C...
			// 选项A=索引2, 选项B=索引3, 选项C=索引4, ... 选项J=索引11
			optionCols[i] = i + 2
		}
	}
	
	// 如果选项A和选项B没有找到，使用默认位置
	if optionCols[0] == -1 {
		optionCols[0] = 2 // 选项A默认在第3列（索引2）
	}
	if optionCols[1] == -1 {
		optionCols[1] = 3 // 选项B默认在第4列（索引3）
	}

	explanationCol := findColumnIndex(headers, []string{"解析", "explanation", "Explanation", "说明"})

	if questionCol == -1 || answerCol == -1 {
		return nil, fmt.Errorf("缺少必要的列：题目（第1列）、正确答案（第2列）")
	}

	var questions []Question
	for i := 1; i < len(records); i++ {
		record := records[i]
		
		// 检查记录长度，如果字段数量不足，记录警告并跳过
		if len(record) < 3 {
			log.Printf("警告: 第 %d 行字段数量不足（期望至少3列：题目、答案、选项），实际 %d 列，跳过该行", i+1, len(record))
			continue
		}

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

		// 解析答案（支持单选和多选）
		answerStr := getCSVValue(record, answerCol)
		if answerStr == "" {
			continue
		}

		// 去除引号（CSV中可能被引号包围）
		answerStr = strings.Trim(answerStr, `"'`)
		answerStr = strings.TrimSpace(answerStr)

		// 检查是否为简答题，如果是则跳过
		questionText := strings.TrimSpace(question)
		questionTextLower := strings.ToLower(questionText)
		
		// 简答题关键词
		essayKeywords := []string{"简述", "说明", "论述", "分析", "解释", "描述", "阐述", "阐述", "评述", "评价", "比较", "对比", "总结", "概述", "介绍", "说明原因", "说明理由", "说明方法", "说明步骤"}
		isEssayQuestion := false
		for _, keyword := range essayKeywords {
			if strings.Contains(questionTextLower, keyword) {
				isEssayQuestion = true
				break
			}
		}
		
		// 如果题目包含简答题关键词，且没有选项，则跳过
		if isEssayQuestion && len(options) == 0 {
			log.Printf("跳过简答题: %s", func() string {
				preview := questionText
				if len(preview) > 50 {
					return preview[:50] + "..."
				}
				return preview
			}())
			continue
		}
		
		// 如果答案长度较长（超过30个字符），且没有选项，则可能是简答题，跳过
		if len(answerStr) > 30 && len(options) == 0 {
			log.Printf("跳过简答题（答案过长且无选项）: %s", func() string {
				preview := questionText
				if len(preview) > 50 {
					return preview[:50] + "..."
				}
				return preview
			}())
			continue
		}

		// 判断题目类型：判断题或选择题
		answerStrUpper := strings.ToUpper(answerStr)
		isJudgment := answerStrUpper == "正确" || answerStrUpper == "错误" || 
			answerStrUpper == "TRUE" || answerStrUpper == "FALSE" ||
			answerStrUpper == "T" || answerStrUpper == "F" ||
			answerStrUpper == "√" || answerStrUpper == "×" ||
			answerStrUpper == "对" || answerStrUpper == "错" ||
			answerStrUpper == "是" || answerStrUpper == "否"

		var questionType string
		var answer []int
		var finalOptions []string

		if isJudgment {
			// 判断题：选项固定为["错误", "正确"]，答案：0=错误，1=正确
			questionType = "judgment"
			finalOptions = []string{"错误", "正确"}
			
			if answerStrUpper == "正确" || answerStrUpper == "TRUE" || answerStrUpper == "T" || 
				answerStrUpper == "√" || answerStrUpper == "对" || answerStrUpper == "是" {
				answer = []int{1} // 正确
			} else {
				answer = []int{0} // 错误
			}
		} else {
			// 选择题：至少需要2个选项
			if len(options) < 2 {
				continue
			}
			
			questionType = "choice"
			finalOptions = options

			// 检查是否是多选题（答案中包含逗号）
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
		}

		if len(answer) == 0 {
			continue
		}

		explanation := ""
		if explanationCol != -1 {
			explanation = getCSVValue(record, explanationCol)
		}

		isMultiple := len(answer) > 1 && questionType == "choice"
		questions = append(questions, Question{
			Question:    question,
			Options:     finalOptions,
			Answer:      answer,
			IsMultiple:  isMultiple,
			Type:        questionType,
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

	// 获取题目，按类型排序：判断题 -> 单选题 -> 多选题
	rows, err := db.Query("SELECT id, question, options, answer, is_multiple, type, explanation FROM questions WHERE bank_id = ? ORDER BY CASE WHEN type = 'judgment' THEN 1 WHEN type = 'choice' AND is_multiple = 0 THEN 2 WHEN type = 'choice' AND is_multiple = 1 THEN 3 ELSE 4 END, id", bankID)
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
		var questionType string
		err := rows.Scan(&q.ID, &q.Question, &optionsJSON, &answerJSON, &isMultiple, &questionType, &q.Explanation)
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

		// 设置是否为多选题和题目类型
		q.IsMultiple = isMultiple
		q.Type = questionType
		if q.Type == "" {
			q.Type = "choice" // 默认为选择题
		}

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
		Type        string   `json:"type"`                       // 题目类型：choice（选择题）或judgment（判断题）
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

	// 设置题目类型（默认为选择题）
	questionType := req.Type
	if questionType == "" {
		questionType = "choice"
	}

	// 处理判断题
	var finalOptions []string
	var finalAnswer []int
	if questionType == "judgment" {
		// 判断题：选项固定为["错误", "正确"]，答案：0=错误，1=正确
		finalOptions = []string{"错误", "正确"}
		if len(req.Answer) > 0 {
			// 验证答案只能是0或1
			if req.Answer[0] != 0 && req.Answer[0] != 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "判断题答案只能是0（错误）或1（正确）"})
				return
			}
			finalAnswer = []int{req.Answer[0]}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "判断题答案不能为空"})
			return
		}
	} else {
		// 选择题：验证选项数量（最多10个）
		if len(req.Options) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("选项数量超过10个（当前：%d）", len(req.Options))})
			return
		}

		// 验证选项数量（至少2个）
		if len(req.Options) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "选择题至少需要2个选项"})
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
		finalOptions = req.Options
		finalAnswer = req.Answer
	}

	// 序列化选项和答案
	optionsJSON, err := json.Marshal(finalOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "选项序列化失败"})
		return
	}

	answerJSON, err := json.Marshal(finalAnswer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "答案序列化失败"})
		return
	}

	// 判断是否为多选题（只有选择题才可能是多选题）
	isMultiple := len(finalAnswer) > 1 && questionType == "choice"

	// 插入题目
	questionID := generateUUID()
	_, err = db.Exec("INSERT INTO questions (id, bank_id, question, options, answer, is_multiple, type, explanation) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		questionID, req.BankID, req.Question, string(optionsJSON), string(answerJSON), isMultiple, questionType, req.Explanation)
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
		Type        string   `json:"type"`                       // 题目类型：choice（选择题）或judgment（判断题）
		Explanation string   `json:"explanation"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置题目类型（默认为选择题）
	questionType := req.Type
	if questionType == "" {
		questionType = "choice"
	}

	// 处理判断题
	var finalOptions []string
	var finalAnswer []int
	if questionType == "judgment" {
		// 判断题：选项固定为["错误", "正确"]，答案：0=错误，1=正确
		finalOptions = []string{"错误", "正确"}
		if len(req.Answer) > 0 {
			// 验证答案只能是0或1
			if req.Answer[0] != 0 && req.Answer[0] != 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "判断题答案只能是0（错误）或1（正确）"})
				return
			}
			finalAnswer = []int{req.Answer[0]}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "判断题答案不能为空"})
			return
		}
	} else {
		// 选择题：验证选项数量（最多10个）
		if len(req.Options) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("选项数量超过10个（当前：%d）", len(req.Options))})
			return
		}

		// 验证选项数量（至少2个）
		if len(req.Options) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "选择题至少需要2个选项"})
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
		finalOptions = req.Options
		finalAnswer = req.Answer
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
	optionsJSON, err := json.Marshal(finalOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "选项序列化失败"})
		return
	}

	answerJSON, err := json.Marshal(finalAnswer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "答案序列化失败"})
		return
	}

	// 判断是否为多选题（只有选择题才可能是多选题）
	isMultiple := len(finalAnswer) > 1 && questionType == "choice"

	// 更新题目
	_, err = db.Exec("UPDATE questions SET question = ?, options = ?, answer = ?, is_multiple = ?, type = ?, explanation = ? WHERE id = ?",
		req.Question, string(optionsJSON), string(answerJSON), isMultiple, questionType, req.Explanation, questionID)
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

	// 获取题目，按类型排序：判断题 -> 单选题 -> 多选题
	rows, err := db.Query("SELECT id, question, options, answer, is_multiple, type, explanation FROM questions WHERE bank_id = ? ORDER BY CASE WHEN type = 'judgment' THEN 1 WHEN type = 'choice' AND is_multiple = 0 THEN 2 WHEN type = 'choice' AND is_multiple = 1 THEN 3 ELSE 4 END, id", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
		return
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		var optionsJSON, answerJSON string
		var isMultiple bool
		var questionType string
		err := rows.Scan(&q.ID, &q.Question, &optionsJSON, &answerJSON, &isMultiple, &questionType, &q.Explanation)
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

		// 解析答案JSON（支持数组）
		err = json.Unmarshal([]byte(answerJSON), &q.Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "答案解析失败"})
			return
		}

		q.IsMultiple = isMultiple
		q.Type = questionType
		if q.Type == "" {
			q.Type = "choice"
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

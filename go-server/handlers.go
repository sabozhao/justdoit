package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
	rows, err := db.Query("SELECT id, bank_id, question, options, answer, explanation FROM questions WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var q Question
		var optionsJSON string
		err := rows.Scan(&q.ID, &q.BankID, &q.Question, &optionsJSON, &q.Answer, &q.Explanation)
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

		questionID := generateUUID()
		_, err = tx.Exec("INSERT INTO questions (id, bank_id, question, options, answer, explanation) VALUES (?, ?, ?, ?, ?, ?)",
			questionID, bankID, q.Question, string(optionsJSON), q.Answer, q.Explanation)
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

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}
	defer file.Close()

	// 解析文件
	questions, err := parseUploadedFile(file, header)
	if err != nil {
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

	// 插入题目
	for i, q := range questions {
		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("第 %d 题选项序列化失败", i+1)})
			return
		}

		questionID := generateUUID()
		_, err = tx.Exec("INSERT INTO questions (id, bank_id, question, options, answer, explanation) VALUES (?, ?, ?, ?, ?, ?)",
			questionID, bankID, q.Question, string(optionsJSON), q.Answer, q.Explanation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("第 %d 题插入失败", i+1)})
			return
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

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
		"questionCount": len(questions),
		"message":       fmt.Sprintf("成功导入 %d 道题目到题库", len(questions)),
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
func parseUploadedFile(file multipart.File, header *multipart.FileHeader) ([]Question, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))

	switch ext {
	case ".json":
		return parseJSONFile(file)
	case ".xlsx", ".xls":
		return parseExcelFile(file)
	case ".csv":
		return parseCSVFile(file)
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
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
	questionCol := findColumnIndex(headers, []string{"题目", "question", "Question", "问题"})
	optionACol := findColumnIndex(headers, []string{"选项A", "A", "optionA", "选择A"})
	optionBCol := findColumnIndex(headers, []string{"选项B", "B", "optionB", "选择B"})
	optionCCol := findColumnIndex(headers, []string{"选项C", "C", "optionC", "选择C"})
	optionDCol := findColumnIndex(headers, []string{"选项D", "D", "optionD", "选择D"})
	answerCol := findColumnIndex(headers, []string{"正确答案", "answer", "Answer", "答案"})
	explanationCol := findColumnIndex(headers, []string{"解析", "explanation", "Explanation", "说明"})

	if questionCol == -1 || optionACol == -1 || optionBCol == -1 || answerCol == -1 {
		return nil, fmt.Errorf("缺少必要的列：题目、选项A、选项B、正确答案")
	}

	var questions []Question
	for i := 1; i < len(rows); i++ {
		row := rows[i]

		question := getExcelValue(row, questionCol)
		if question == "" {
			continue
		}

		optionA := getExcelValue(row, optionACol)
		optionB := getExcelValue(row, optionBCol)
		if optionA == "" || optionB == "" {
			continue
		}

		options := []string{optionA, optionB}

		// 添加可选的选项C和D
		if optionCCol != -1 {
			if optionC := getExcelValue(row, optionCCol); optionC != "" {
				options = append(options, optionC)
			}
		}
		if optionDCol != -1 {
			if optionD := getExcelValue(row, optionDCol); optionD != "" {
				options = append(options, optionD)
			}
		}

		answerStr := getExcelValue(row, answerCol)
		answer, err := parseAnswer(answerStr, len(options))
		if err != nil {
			return nil, fmt.Errorf("第 %d 行答案格式错误: %v", i+1, err)
		}

		explanation := ""
		if explanationCol != -1 {
			explanation = getExcelValue(row, explanationCol)
		}

		questions = append(questions, Question{
			Question:    question,
			Options:     options,
			Answer:      answer,
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
	questionCol := findColumnIndex(headers, []string{"题目", "question", "Question", "问题"})
	optionACol := findColumnIndex(headers, []string{"选项A", "A", "optionA", "选择A"})
	optionBCol := findColumnIndex(headers, []string{"选项B", "B", "optionB", "选择B"})
	optionCCol := findColumnIndex(headers, []string{"选项C", "C", "optionC", "选择C"})
	optionDCol := findColumnIndex(headers, []string{"选项D", "D", "optionD", "选择D"})
	answerCol := findColumnIndex(headers, []string{"正确答案", "answer", "Answer", "答案"})
	explanationCol := findColumnIndex(headers, []string{"解析", "explanation", "Explanation", "说明"})

	if questionCol == -1 || optionACol == -1 || optionBCol == -1 || answerCol == -1 {
		return nil, fmt.Errorf("缺少必要的列：题目、选项A、选项B、正确答案")
	}

	var questions []Question
	for i := 1; i < len(records); i++ {
		record := records[i]

		question := getCSVValue(record, questionCol)
		if question == "" {
			continue
		}

		optionA := getCSVValue(record, optionACol)
		optionB := getCSVValue(record, optionBCol)
		if optionA == "" || optionB == "" {
			continue
		}

		options := []string{optionA, optionB}

		// 添加可选的选项C和D
		if optionCCol != -1 {
			if optionC := getCSVValue(record, optionCCol); optionC != "" {
				options = append(options, optionC)
			}
		}
		if optionDCol != -1 {
			if optionD := getCSVValue(record, optionDCol); optionD != "" {
				options = append(options, optionD)
			}
		}

		answerStr := getCSVValue(record, answerCol)
		answer, err := parseAnswer(answerStr, len(options))
		if err != nil {
			return nil, fmt.Errorf("第 %d 行答案格式错误: %v", i+1, err)
		}

		explanation := ""
		if explanationCol != -1 {
			explanation = getCSVValue(record, explanationCol)
		}

		questions = append(questions, Question{
			Question:    question,
			Options:     options,
			Answer:      answer,
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

// 添加题目
func createQuestion(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		BankID      string   `json:"bank_id" binding:"required"`
		Question    string   `json:"question" binding:"required"`
		Options     []string `json:"options" binding:"required"`
		answer      int      `json:"answer" binding:"required"`
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

	// 验证答案范围
	if req.answer < 0 || req.answer >= len(req.Options) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "答案索引超出选项范围"})
		return
	}

	// 序列化选项
	optionsJSON, err := json.Marshal(req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "选项序列化失败"})
		return
	}

	// 插入题目
	questionID := generateUUID()
	_, err = db.Exec("INSERT INTO questions (id, bank_id, question, options, answer, explanation) VALUES (?, ?, ?, ?, ?, ?)",
		questionID, req.BankID, req.Question, string(optionsJSON), req.answer, req.Explanation)
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
		Answer      int      `json:"answer" binding:"required"`
		Explanation string   `json:"explanation"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证答案范围
	if req.Answer < 0 || req.Answer >= len(req.Options) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "答案索引超出选项范围"})
		return
	}

	// 检查题目是否属于当前用户的题库
	var bankID string
	err := db.QueryRow("SELECT q.bank_id FROM questions q JOIN question_banks qb ON q.bank_id = qb.id WHERE q.id = ? AND qb.user_id = ?",
		questionID, userID).Scan(&bankID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在或无权修改"})
		return
	}

	// 序列化选项
	optionsJSON, err := json.Marshal(req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "选项序列化失败"})
		return
	}

	// 更新题目
	_, err = db.Exec("UPDATE questions SET question = ?, options = ?, answer = ?, explanation = ? WHERE id = ?",
		req.Question, string(optionsJSON), req.Answer, req.Explanation, questionID)
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

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ==================== 共享题库管理 API ====================

// 获取所有共享题库
func getPublicQuestionBanks(c *gin.Context) {
	categoryID := c.Query("category_id") // 分类ID（可选）

	// 构建查询：只查询共享题库
	query := `
		SELECT qb.id, qb.name, qb.description, qb.category_id, 
		       c.name as category_name, qb.created_at, COUNT(q.id) as question_count
		FROM public_question_banks qb 
		LEFT JOIN public_questions q ON qb.id = q.bank_id 
		LEFT JOIN public_categories c ON qb.category_id = c.id
		WHERE 1=1
	`
	args := []interface{}{}

	// 如果指定了分类ID，添加分类筛选
	if categoryID != "" && categoryID != "null" {
		query += " AND qb.category_id = ?"
		args = append(args, categoryID)
	}

	query += " GROUP BY qb.id ORDER BY qb.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var banks []QuestionBank
	for rows.Next() {
		var bank QuestionBank
		var categoryID sql.NullString
		var categoryName sql.NullString
		err := rows.Scan(&bank.ID, &bank.Name, &bank.Description, 
			&categoryID, &categoryName, &bank.CreatedAt, &bank.QuestionCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		bank.IsPublic = true // 共享题库
		bank.UserID = ""     // 共享题库不归属任何用户
		if categoryID.Valid {
			bank.CategoryID = &categoryID.String
		}
		if categoryName.Valid {
			bank.CategoryName = categoryName.String
		}
		banks = append(banks, bank)
	}

	c.JSON(http.StatusOK, banks)
}

// 获取共享题库详情
func getPublicQuestionBankByID(c *gin.Context) {
	bankID := c.Param("id")

	// 获取共享题库信息
	var bank QuestionBank
	var categoryID sql.NullString
	var categoryName sql.NullString
	err := db.QueryRow(`
		SELECT qb.id, qb.name, qb.description, qb.category_id, 
		       c.name as category_name, qb.created_at 
		FROM public_question_banks qb
		LEFT JOIN public_categories c ON qb.category_id = c.id
		WHERE qb.id = ?
	`, bankID).Scan(&bank.ID, &bank.Name, &bank.Description, 
		&categoryID, &categoryName, &bank.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "共享题库不存在"})
		return
	}
	
	bank.IsPublic = true // 共享题库
	bank.UserID = ""     // 共享题库不归属任何用户
	
	if categoryID.Valid {
		bank.CategoryID = &categoryID.String
	}
	if categoryName.Valid {
		bank.CategoryName = categoryName.String
	}

	// 获取题目，按类型排序：判断题 -> 单选题 -> 多选题
	rows, err := db.Query("SELECT id, bank_id, question, options, answer, is_multiple, type, explanation FROM public_questions WHERE bank_id = ? ORDER BY CASE WHEN type = 'judgment' THEN 1 WHEN type = 'choice' AND is_multiple = 0 THEN 2 WHEN type = 'choice' AND is_multiple = 1 THEN 3 ELSE 4 END, id", bankID)
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

// 创建共享题库（仅管理员）
func createPublicQuestionBank(c *gin.Context) {
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以创建共享题库"})
		return
	}

	var req struct {
		Name        string     `json:"name" binding:"required"`
		Description string     `json:"description"`
		CategoryID  *string    `json:"category_id"`  // 分类ID（可选，必须是共享分类）
		Questions   []Question `json:"questions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果指定了分类，验证分类是否存在（必须是共享分类）
	if req.CategoryID != nil && *req.CategoryID != "" {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_categories WHERE id = ?)", *req.CategoryID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "指定的共享分类不存在"})
			return
		}
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// 插入共享题库
	bankID := generateUUID()
	_, err = tx.Exec("INSERT INTO public_question_banks (id, name, description, category_id) VALUES (?, ?, ?, ?)",
		bankID, req.Name, req.Description, req.CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create public question bank"})
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
		
		_, err = tx.Exec("INSERT INTO public_questions (id, bank_id, question, options, answer, is_multiple, type, explanation) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
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
		"message":     "共享题库创建成功",
	})
}

// 更新共享题库（仅管理员）
func updatePublicQuestionBank(c *gin.Context) {
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以更新共享题库"})
		return
	}

	bankID := c.Param("id")

	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		CategoryID  *string `json:"category_id"` // 分类ID（可选，必须是共享分类）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查共享题库是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_question_banks WHERE id = ?)", bankID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "共享题库不存在"})
		return
	}

	// 如果指定了分类，验证分类是否存在（必须是共享分类）
	if req.CategoryID != nil && *req.CategoryID != "" {
		var categoryExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_categories WHERE id = ?)", *req.CategoryID).Scan(&categoryExists)
		if err != nil || !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "指定的共享分类不存在"})
			return
		}
	}

	// 更新共享题库信息
	_, err = db.Exec("UPDATE public_question_banks SET name = ?, description = ?, category_id = ? WHERE id = ?",
		req.Name, req.Description, req.CategoryID, bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新共享题库失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "共享题库更新成功"})
}

// 删除共享题库（仅管理员）
func deletePublicQuestionBank(c *gin.Context) {
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以删除共享题库"})
		return
	}

	bankID := c.Param("id")

	// 检查共享题库是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_question_banks WHERE id = ?)", bankID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "共享题库不存在"})
		return
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// 删除相关的错题（共享题库的错题也在 wrong_questions 表中）
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
	_, err = tx.Exec("DELETE FROM public_questions WHERE bank_id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete questions"})
		return
	}

	// 删除共享题库
	result, err := tx.Exec("DELETE FROM public_question_banks WHERE id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete public question bank"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "共享题库不存在"})
		return
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "共享题库删除成功"})
}

// 上传共享题库文件（仅管理员）
func uploadPublicQuestionBankFile(c *gin.Context) {
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以上传共享题库"})
		return
	}

	userID := c.GetString("userID")
	bankID := c.Param("id")
	
	// 获取表单参数
	bankName := c.PostForm("bankName")
	categoryID := c.PostForm("category_id")
	
	var targetBankID string

	// 判断是上传到已有题库还是创建新题库
	if bankID == "new" || bankID == "" {
		// 创建新共享题库
		if bankName == "" {
			// 如果没有提供题库名称，使用文件名（去掉扩展名）
			filename := c.PostForm("filename")
			if filename == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请提供题库名称或上传文件"})
				return
			}
			bankName = strings.TrimSuffix(filename, filepath.Ext(filename))
		}
		
		// 如果指定了分类，验证分类是否存在（必须是共享分类）
		var categoryIDPtr *string
		if categoryID != "" {
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_categories WHERE id = ?)", categoryID).Scan(&exists)
			if err != nil || !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "指定的共享分类不存在"})
				return
			}
			categoryIDPtr = &categoryID
		}
		
		// 创建新共享题库
		targetBankID = generateUUID()
		_, err := db.Exec("INSERT INTO public_question_banks (id, name, description, category_id) VALUES (?, ?, ?, ?)",
			targetBankID, bankName, "", categoryIDPtr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建共享题库失败: " + err.Error()})
			return
		}
	} else {
		// 上传到已有共享题库
		// 检查共享题库是否存在
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_question_banks WHERE id = ?)", bankID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "共享题库不存在"})
			return
		}
		
		targetBankID = bankID
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
			// 重置文件指针，确保可以完整读取
			file.Seek(0, 0)
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
				c.JSON(http.StatusBadRequest, gin.H{"error": "Word文件解析失败: " + err.Error()})
				return
			}
		case ".pdf":
			// PDF文件必须使用AI解析
			c.JSON(http.StatusBadRequest, gin.H{"error": "PDF文件必须使用AI自动分析模式"})
			return
		case ".doc":
			// 旧版DOC格式不支持
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持旧版DOC格式，请转换为DOCX格式或使用AI自动分析"})
			return
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件格式"})
			return
		}
	}

	if len(questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件中没有找到题目"})
		return
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// 插入题目到共享题库
	for _, q := range questions {
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

		isMultiple := len(q.Answer) > 1
		questionID := generateUUID()
		questionType := q.Type
		if questionType == "" {
			questionType = "choice"
		}

		_, err = tx.Exec("INSERT INTO public_questions (id, bank_id, question, options, answer, is_multiple, type, explanation) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			questionID, targetBankID, q.Question, string(optionsJSON), string(answerJSON), isMultiple, questionType, q.Explanation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert question"})
			return
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("成功上传 %d 道题目到共享题库", len(questions)),
		"questionCount": len(questions),
		"bankId":       targetBankID,
	})
}

// 获取共享题库的题目列表
func getPublicBankQuestions(c *gin.Context) {
	bankID := c.Param("id")

	// 检查共享题库是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_question_banks WHERE id = ?)", bankID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败: " + err.Error()})
		return
	}
	
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "共享题库不存在"})
		return
	}

	// 获取题目，按类型排序：判断题 -> 单选题 -> 多选题
	rows, err := db.Query("SELECT id, question, options, answer, is_multiple, type, explanation FROM public_questions WHERE bank_id = ? ORDER BY CASE WHEN type = 'judgment' THEN 1 WHEN type = 'choice' AND is_multiple = 0 THEN 2 WHEN type = 'choice' AND is_multiple = 1 THEN 3 ELSE 4 END, id", bankID)
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

// 创建共享题库题目（仅管理员）
func createPublicQuestion(c *gin.Context) {
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以创建共享题库题目"})
		return
	}

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

	// 检查共享题库是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_question_banks WHERE id = ?)", req.BankID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "共享题库不存在"})
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
	_, err = db.Exec("INSERT INTO public_questions (id, bank_id, question, options, answer, is_multiple, type, explanation) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
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

// 更新共享题库题目（仅管理员）
func updatePublicQuestion(c *gin.Context) {
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以更新共享题库题目"})
		return
	}

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

	// 检查题目是否存在（共享题库题目）
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_questions WHERE id = ?)", questionID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
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

	// 更新题目
	_, err = db.Exec("UPDATE public_questions SET question = ?, options = ?, answer = ?, is_multiple = ?, type = ?, explanation = ? WHERE id = ?",
		req.Question, string(optionsJSON), string(answerJSON), isMultiple, questionType, req.Explanation, questionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题目失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题目更新成功"})
}

// 删除共享题库题目（仅管理员）
func deletePublicQuestion(c *gin.Context) {
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以删除共享题库题目"})
		return
	}

	questionID := c.Param("id")

	// 检查题目是否存在（共享题库题目）
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM public_questions WHERE id = ?)", questionID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}

	// 删除题目
	result, err := db.Exec("DELETE FROM public_questions WHERE id = ?", questionID)
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


package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错题相关处理函数
func getWrongQuestions(c *gin.Context) {
	userID := c.GetString("userID")

	query := `
		SELECT wq.id, wq.user_id, wq.bank_id, wq.question_id, wq.question, wq.options, wq.answer, wq.is_multiple, wq.type, wq.explanation, wq.added_at, 
		       COALESCE(qb.name, pub_qb.name) as bank_name, 
		       CASE WHEN pub_qb.id IS NOT NULL THEN 1 ELSE 0 END as is_public,
		       COALESCE(qb.category_id, pub_qb.category_id) as category_id,
		       COALESCE(c.name, pub_c.name) as category_name
		FROM wrong_questions wq 
		LEFT JOIN question_banks qb ON wq.bank_id = qb.id 
		LEFT JOIN categories c ON qb.category_id = c.id
		LEFT JOIN public_question_banks pub_qb ON wq.bank_id = pub_qb.id
		LEFT JOIN public_categories pub_c ON pub_qb.category_id = pub_c.id
		WHERE wq.user_id = ?
		ORDER BY wq.added_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var wrongQuestions []WrongQuestion
	for rows.Next() {
		var wq WrongQuestion
		var optionsJSON, answerJSON string
		var isMultiple bool
		var questionType sql.NullString
		var isPublic bool
		var categoryID sql.NullString
		var categoryName sql.NullString
		err := rows.Scan(&wq.ID, &wq.UserID, &wq.BankID, &wq.QuestionID, &wq.Question, &optionsJSON, &answerJSON, &isMultiple, &questionType, &wq.Explanation, &wq.AddedAt, &wq.BankName, &isPublic, &categoryID, &categoryName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		wq.IsPublic = isPublic
		if categoryID.Valid {
			categoryIDStr := categoryID.String
			wq.CategoryID = &categoryIDStr
		}
		if categoryName.Valid {
			wq.CategoryName = categoryName.String
		}

		// 解析选项JSON
		err = json.Unmarshal([]byte(optionsJSON), &wq.Options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse options"})
			return
		}

		// 解析答案JSON（支持数组）
		err = json.Unmarshal([]byte(answerJSON), &wq.Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse answer"})
			return
		}

		wq.IsMultiple = isMultiple
		if questionType.Valid {
			wq.Type = questionType.String
		} else {
			wq.Type = "choice" // 默认值
		}
		wrongQuestions = append(wrongQuestions, wq)
	}

	c.JSON(http.StatusOK, wrongQuestions)
}

func addWrongQuestion(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		BankID      string   `json:"bankId" binding:"required"`
		QuestionID  string   `json:"questionId" binding:"required"`
		Question    string   `json:"question" binding:"required"`
		Options     []string `json:"options" binding:"required"`
		Answer      []int    `json:"answer" binding:"required"` // 支持多选
		Explanation string   `json:"explanation"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查是否已存在相同的错题
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM wrong_questions WHERE user_id = ? AND bank_id = ? AND question_id = ?)",
		userID, req.BankID, req.QuestionID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if exists {
		c.JSON(http.StatusOK, gin.H{"message": "Wrong question already exists"})
		return
	}

	// 序列化选项和答案
	optionsJSON, err := json.Marshal(req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal options"})
		return
	}

	answerJSON, err := json.Marshal(req.Answer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal answer"})
		return
	}

	// 判断是否为多选题
	isMultiple := len(req.Answer) > 1

	// 添加新的错题
	wrongQuestionID := generateUUID()
	_, err = db.Exec(`INSERT INTO wrong_questions 
		(id, user_id, bank_id, question_id, question, options, answer, is_multiple, explanation) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		wrongQuestionID, userID, req.BankID, req.QuestionID, req.Question, string(optionsJSON), string(answerJSON), isMultiple, req.Explanation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      wrongQuestionID,
		"message": "Wrong question added successfully",
	})
}

// addWrongQuestionsBatch 批量添加错题
func addWrongQuestionsBatch(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		Questions []struct {
			BankID      string   `json:"bankId" binding:"required"`
			QuestionID  string   `json:"questionId" binding:"required"`
			Question    string   `json:"question" binding:"required"`
			Options     []string `json:"options" binding:"required"`
			Answer      []int    `json:"answer" binding:"required"`
			IsMultiple  bool     `json:"is_multiple"`
			Type        string   `json:"type"`
			Explanation string   `json:"explanation"`
		} `json:"questions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Questions array is empty"})
		return
	}

	var addedCount int
	var skippedCount int

	// 使用事务批量插入
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	for _, q := range req.Questions {
		// 检查是否已存在相同的错题
		var exists bool
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM wrong_questions WHERE user_id = ? AND bank_id = ? AND question_id = ?)",
			userID, q.BankID, q.QuestionID).Scan(&exists)
		if err != nil {
			continue // 跳过有错误的题目
		}

		if exists {
			skippedCount++
			continue
		}

		// 序列化选项和答案
		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			skippedCount++
			continue
		}

		answerJSON, err := json.Marshal(q.Answer)
		if err != nil {
			skippedCount++
			continue
		}

		// 判断是否为多选题（如果前端没有传递is_multiple，则根据答案数量判断）
		isMultiple := q.IsMultiple
		if !isMultiple {
			isMultiple = len(q.Answer) > 1
		}

		// 添加新的错题
		wrongQuestionID := generateUUID()
		_, err = tx.Exec(`INSERT INTO wrong_questions 
			(id, user_id, bank_id, question_id, question, options, answer, is_multiple, type, explanation) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			wrongQuestionID, userID, q.BankID, q.QuestionID, q.Question, string(optionsJSON), string(answerJSON), isMultiple, q.Type, q.Explanation)
		if err != nil {
			skippedCount++
			continue
		}

		addedCount++
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Wrong questions added successfully",
		"addedCount":   addedCount,
		"skippedCount": skippedCount,
		"totalCount":  len(req.Questions),
	})
}

func removeWrongQuestion(c *gin.Context) {
	userID := c.GetString("userID")
	wrongQuestionID := c.Param("id")

	result, err := db.Exec("DELETE FROM wrong_questions WHERE id = ? AND user_id = ?", wrongQuestionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wrong question not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wrong question removed successfully"})
}

func clearAllWrongQuestions(c *gin.Context) {
	userID := c.GetString("userID")

	result, err := db.Exec("DELETE FROM wrong_questions WHERE user_id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	c.JSON(http.StatusOK, gin.H{
		"message":      "All wrong questions cleared successfully",
		"deletedCount": rowsAffected,
	})
}
package main

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错题相关处理函数
func getWrongQuestions(c *gin.Context) {
	userID := c.GetString("userID")

	query := `
		SELECT wq.id, wq.user_id, wq.bank_id, wq.question_id, wq.question, wq.options, wq.answer, wq.explanation, wq.added_at, qb.name as bank_name
		FROM wrong_questions wq 
		LEFT JOIN question_banks qb ON wq.bank_id = qb.id 
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
		var optionsJSON string
		err := rows.Scan(&wq.ID, &wq.UserID, &wq.BankID, &wq.QuestionID, &wq.Question, &optionsJSON, &wq.Answer, &wq.Explanation, &wq.AddedAt, &wq.BankName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 解析选项JSON
		err = json.Unmarshal([]byte(optionsJSON), &wq.Options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse options"})
			return
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
		Answer      int      `json:"answer"`
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

	// 序列化选项
	optionsJSON, err := json.Marshal(req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal options"})
		return
	}

	// 添加新的错题
	wrongQuestionID := generateUUID()
	_, err = db.Exec(`INSERT INTO wrong_questions 
		(id, user_id, bank_id, question_id, question, options, answer, explanation) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		wrongQuestionID, userID, req.BankID, req.QuestionID, req.Question, string(optionsJSON), req.Answer, req.Explanation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      wrongQuestionID,
		"message": "Wrong question added successfully",
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
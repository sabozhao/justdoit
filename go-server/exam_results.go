package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 考试结果相关处理函数
func saveExamResult(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		BankID         string `json:"bankId" binding:"required"`
		Score          int    `json:"score"`
		CorrectCount   int    `json:"correctCount"`
		WrongCount     int    `json:"wrongCount"`
		TotalQuestions int    `json:"totalQuestions"`
		TotalTime      int    `json:"totalTime"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证bank_id是否存在 - 为错题练习添加特殊处理
	var bankExists bool
	var err error

	if req.BankID == "wrong-questions-all" || req.BankID == "wrong-questions" {
		// 错题练习使用特殊标识，跳过题库验证
		req.BankID = "wrong-questions-practice"
		bankExists = true
	} else {
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM question_banks WHERE id = ?)", req.BankID).Scan(&bankExists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		if !bankExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bank_id: question bank not found"})
			return
		}
	}

	resultID := generateUUID()
	_, err = db.Exec(`INSERT INTO exam_results 
		(id, user_id, bank_id, score, correct_count, wrong_count, total_questions, total_time) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		resultID, userID, req.BankID, req.Score, req.CorrectCount, req.WrongCount, req.TotalQuestions, req.TotalTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      resultID,
		"message": "Exam result saved successfully",
	})
}

func getExamStats(c *gin.Context) {
	userID := c.GetString("userID")

	var stats ExamStats
	err := db.QueryRow(`
		SELECT 
			COUNT(*) as total_exams,
			COALESCE(AVG(score), 0) as avg_score,
			COALESCE(MAX(score), 0) as best_score,
			COALESCE(SUM(total_questions), 0) as total_questions_answered
		FROM exam_results
		WHERE user_id = ?
	`, userID).Scan(&stats.TotalExams, &stats.AvgScore, &stats.BestScore, &stats.TotalQuestionsAnswered)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

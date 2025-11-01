package main

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupRoutes() *gin.Engine {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// CORS配置
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:5173",
		"http://localhost:5174",
		"http://localhost:5175",
		"http://localhost:5176",
		"http://localhost:5177",
		"http://119.91.68.147",
		"http://119.91.68.147:80",
		"http://119.91.68.147:5173",
		"https://examtest.top",
		"http://examtest.top",
		"*", // 允许所有域名（生产环境建议指定具体域名）
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "Cache-Control"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// API路由组
	api := r.Group("/api")

	// 认证相关路由
	auth := api.Group("/auth")
	{
		auth.POST("/register", register)
		auth.POST("/login", login)
		auth.GET("/me", authMiddleware(), getCurrentUser)
	}

	// 题库相关路由（需要认证）
	questionBanks := api.Group("/question-banks")
	questionBanks.Use(authMiddleware())
	{
		questionBanks.GET("", getQuestionBanks)
		questionBanks.GET("/:id", getQuestionBankByID)
		questionBanks.POST("", createQuestionBank)
		questionBanks.POST("/:id/upload", uploadQuestionBankFile)
		questionBanks.DELETE("/:id", deleteQuestionBank)
		questionBanks.GET("/:id/questions", getBankQuestions)
	}

	// 题目管理相关路由（需要认证）
	questions := api.Group("/questions")
	questions.Use(authMiddleware())
	{
		questions.POST("", createQuestion)
		questions.PUT("/:id", updateQuestion)
		questions.DELETE("/:id", deleteQuestion)
	}

	// 错题相关路由（需要认证）
	wrongQuestions := api.Group("/wrong-questions")
	wrongQuestions.Use(authMiddleware())
	{
		wrongQuestions.GET("", getWrongQuestions)
		wrongQuestions.POST("", addWrongQuestion)
		wrongQuestions.DELETE("/:id", removeWrongQuestion)
		wrongQuestions.DELETE("", clearAllWrongQuestions)
	}

	// 考试结果相关路由（需要认证）
	examResults := api.Group("/exam-results")
	examResults.Use(authMiddleware())
	{
		examResults.POST("", saveExamResult)
		examResults.GET("/stats", getExamStats)
	}

	// 管理员相关路由（需要管理员权限）
	admin := api.Group("/admin")
	admin.Use(authMiddleware(), adminMiddleware())
	{
		admin.GET("/users", getAllUsers)
		admin.GET("/question-banks", getAllQuestionBanks)
		admin.GET("/stats", getAdminStats)
		admin.DELETE("/users/:id", deleteUser)
		admin.DELETE("/question-banks/:id", deleteQuestionBankAdmin)
		admin.PATCH("/users/:id", updateUserAdmin)
		admin.PUT("/settings", updateSettings)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

// main函数已移到main.go文件中

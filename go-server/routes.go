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
		questionBanks.PUT("/:id", updateQuestionBank)
		questionBanks.POST("/:id/upload", uploadQuestionBankFile)
		questionBanks.DELETE("/:id", deleteQuestionBank)
		questionBanks.GET("/:id/questions", getBankQuestions)
	}

	// 示例文件下载（无需认证）
	api.GET("/demo/:type", downloadDemoFile)

	// 个人题库题目管理相关路由（需要认证）
	questions := api.Group("/questions")
	questions.Use(authMiddleware())
	{
		questions.POST("", createQuestion)
		questions.PUT("/:id", updateQuestion)
		questions.DELETE("/:id", deleteQuestion)
	}

	// 共享题库题目管理相关路由（需要认证，仅管理员可管理）
	publicQuestions := api.Group("/public-questions")
	publicQuestions.Use(authMiddleware(), adminMiddleware())
	{
		publicQuestions.POST("", createPublicQuestion)
		publicQuestions.PUT("/:id", updatePublicQuestion)
		publicQuestions.DELETE("/:id", deletePublicQuestion)
	}

	// 错题相关路由（需要认证）
	wrongQuestions := api.Group("/wrong-questions")
	wrongQuestions.Use(authMiddleware())
	{
		wrongQuestions.GET("", getWrongQuestions)
		wrongQuestions.POST("", addWrongQuestion)
		wrongQuestions.POST("/batch", addWrongQuestionsBatch)
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

	// 个人分类管理路由（需要认证）
	categories := api.Group("/categories")
	categories.Use(authMiddleware())
	{
		categories.GET("", getCategories) // 获取当前用户的个人分类
		categories.POST("", createCategory) // 创建个人分类
		categories.PUT("/:id", updateCategory) // 更新个人分类
		categories.DELETE("/:id", deleteCategory) // 删除个人分类
	}

	// 共享分类管理路由（需要认证，仅管理员可管理，所有用户可查看）
	publicCategories := api.Group("/public-categories")
	publicCategories.Use(authMiddleware())
	{
		publicCategories.GET("", getPublicCategories) // 所有用户都可以查看共享分类
	}
	
	// 共享分类管理路由（仅管理员）
	adminPublicCategories := api.Group("/public-categories")
	adminPublicCategories.Use(authMiddleware(), adminMiddleware())
	{
		adminPublicCategories.POST("", createPublicCategory) // 创建共享分类
		adminPublicCategories.PUT("/:id", updatePublicCategory) // 更新共享分类
		adminPublicCategories.DELETE("/:id", deletePublicCategory) // 删除共享分类
	}

	// 共享题库路由（需要认证，所有用户可查看，仅管理员可管理）
	publicQuestionBanks := api.Group("/public-question-banks")
	publicQuestionBanks.Use(authMiddleware())
	{
		publicQuestionBanks.GET("", getPublicQuestionBanks) // 所有用户都可以查看共享题库
		publicQuestionBanks.GET("/:id", getPublicQuestionBankByID) // 获取共享题库详情
		publicQuestionBanks.GET("/:id/questions", getPublicBankQuestions) // 获取共享题库题目列表
	}
	
	// 共享题库管理路由（仅管理员）
	adminPublicQuestionBanks := api.Group("/public-question-banks")
	adminPublicQuestionBanks.Use(authMiddleware(), adminMiddleware())
	{
		adminPublicQuestionBanks.POST("", createPublicQuestionBank) // 创建共享题库
		adminPublicQuestionBanks.PUT("/:id", updatePublicQuestionBank) // 更新共享题库
		adminPublicQuestionBanks.DELETE("/:id", deletePublicQuestionBank) // 删除共享题库
		adminPublicQuestionBanks.POST("/:id/upload", uploadPublicQuestionBankFile) // 上传共享题库文件
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
		admin.GET("/settings", getSettings)
		admin.PUT("/settings", updateSettings)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

// main函数已移到main.go文件中

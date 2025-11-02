package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// 数据结构定义
type User struct {
	ID        string    `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"-" db:"password"`
	Email     string    `json:"email" db:"email"`
	IsAdmin   bool      `json:"is_admin" db:"is_admin"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type QuestionBank struct {
	ID            string     `json:"id" db:"id"`
	UserID        string     `json:"user_id" db:"user_id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	QuestionCount int        `json:"question_count" db:"question_count"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	Questions     []Question `json:"questions,omitempty"`
}

type Question struct {
	ID          string   `json:"id" db:"id"`
	BankID      string   `json:"bank_id" db:"bank_id"`
	Question    string   `json:"question" db:"question"`
	Options     []string `json:"options" db:"options"` // 最多10个选项
	Answer      []int    `json:"answer" db:"answer"`    // 支持多选，存储答案索引数组
	IsMultiple  bool     `json:"is_multiple" db:"is_multiple"` // 是否为多选题
	Explanation string   `json:"explanation" db:"explanation"`
}

type WrongQuestion struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	BankID      string    `json:"bank_id" db:"bank_id"`
	QuestionID  string    `json:"question_id" db:"question_id"`
	Question    string    `json:"question" db:"question"`
	Options     []string  `json:"options" db:"options"` // 最多10个选项
	Answer      []int     `json:"answer" db:"answer"`    // 支持多选
	IsMultiple  bool      `json:"is_multiple" db:"is_multiple"` // 是否为多选题
	Explanation string    `json:"explanation" db:"explanation"`
	BankName    string    `json:"bank_name" db:"bank_name"`
	AddedAt     time.Time `json:"added_at" db:"added_at"`
}

type ExamResult struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"user_id" db:"user_id"`
	BankID         string    `json:"bank_id" db:"bank_id"`
	Score          int       `json:"score" db:"score"`
	CorrectCount   int       `json:"correct_count" db:"correct_count"`
	WrongCount     int       `json:"wrong_count" db:"wrong_count"`
	TotalQuestions int       `json:"total_questions" db:"total_questions"`
	TotalTime      int       `json:"total_time" db:"total_time"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type ExamStats struct {
	TotalExams             int     `json:"total_exams"`
	AvgScore               float64 `json:"avg_score"`
	BestScore              int     `json:"best_score"`
	TotalQuestionsAnswered int     `json:"total_questions_answered"`
}

// JWT Claims
type Claims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 全局变量
var (
	db        *sql.DB
	jwtSecret = []byte("your-secret-key-change-in-production")
)

// 数据库初始化
func initDB() {
	var err error

	// MySQL连接配置
	// 从环境变量获取数据库配置，如果没有则使用默认值
	dbHost := getEnv("DB_HOST", "gz-cynosdbmysql-grp-37hxchit.sql.tencentcdb.com:24127")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "exam_db")

	// 构建MySQL连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbName)

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// 测试数据库连接
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 创建表
	createTables()

	// 创建默认管理员账号
	createDefaultAdmin()

	// 初始化系统配置表
	initSystemSettings()

	log.Println("Database initialized successfully")
}

// 初始化系统配置表
func initSystemSettings() {
	// 创建系统配置表
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS system_settings (
		setting_key VARCHAR(255) PRIMARY KEY,
		setting_value TEXT,
		description VARCHAR(500),
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Printf("Warning: Failed to create system_settings table: %v", err)
		return
	}

	// 初始化默认配置（如果不存在）
	defaultSettings := map[string]string{
		"tencent_secret_id":  getEnv("TENCENT_SECRET_ID", ""),
		"tencent_secret_key": getEnv("TENCENT_SECRET_KEY", ""),
		"tencent_region":     getEnv("TENCENT_REGION", "ap-beijing"),
		"tencent_model":      getEnv("TENCENT_MODEL", "hunyuan-lite"),
		"tencent_endpoint":   "hunyuan.tencentcloudapi.com",
	}

	for key, value := range defaultSettings {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM system_settings WHERE setting_key = ?)", key).Scan(&exists)
		if err != nil {
			log.Printf("Warning: Failed to check setting %s: %v", key, err)
			continue
		}
		if !exists {
			_, err = db.Exec("INSERT INTO system_settings (setting_key, setting_value, description) VALUES (?, ?, ?)",
				key, value, getSettingDescription(key))
			if err != nil {
				log.Printf("Warning: Failed to insert default setting %s: %v", key, err)
			}
		}
	}
}

// 获取配置项的描述
func getSettingDescription(key string) string {
	descriptions := map[string]string{
		"tencent_secret_id":  "腾讯云API密钥ID（SecretId）",
		"tencent_secret_key": "腾讯云API密钥（SecretKey）",
		"tencent_region":     "腾讯云区域（如：ap-beijing, ap-guangzhou）",
		"tencent_model":      "腾讯云AI模型名称（hunyuan-lite/hunyuan-pro/hunyuan-standard）",
		"tencent_endpoint":   "腾讯云API端点地址",
	}
	return descriptions[key]
}

// 获取系统配置
func getSystemSetting(key string) string {
	var value string
	err := db.QueryRow("SELECT setting_value FROM system_settings WHERE setting_key = ?", key).Scan(&value)
	if err != nil {
		// 如果从数据库获取失败，尝试从环境变量获取
		envKey := strings.ToUpper(strings.ReplaceAll(key, "_", "_"))
		envKey = "TENCENT_" + envKey
		return getEnv(envKey, "")
	}
	return value
}

// 更新系统配置
func updateSystemSetting(key, value string) error {
	_, err := db.Exec(`INSERT INTO system_settings (setting_key, setting_value, description) 
		VALUES (?, ?, ?) 
		ON DUPLICATE KEY UPDATE setting_value = ?, description = ?`,
		key, value, getSettingDescription(key), value, getSettingDescription(key))
	return err
}

// 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 创建默认管理员账号
func createDefaultAdmin() {
	// 检查管理员账号是否已存在
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if err != nil {
		log.Printf("Warning: Failed to check admin user existence: %v", err)
		return
	}

	if count > 0 {
		log.Println("Admin user already exists")
		return
	}

	// 创建管理员账号
	adminID := generateUUID()
	hashedPassword, err := hashPassword("123456")
	if err != nil {
		log.Printf("Warning: Failed to hash admin password: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO users (id, username, password, email, is_admin) VALUES (?, ?, ?, ?, ?)",
		adminID, "admin", hashedPassword, "admin@example.com", true)
	if err != nil {
		log.Printf("Warning: Failed to create admin user: %v", err)
		return
	}

	log.Println("Default admin user created successfully")
}

func createTables() {
	// 用户表
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(255) PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		email VARCHAR(255),
		is_admin BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal("Failed to create users table:", err)
	}

	// 检查并添加is_admin字段（如果不存在）- MySQL兼容版本
	var columnExists bool
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'is_admin'").Scan(&columnExists)
	if err != nil {
		log.Printf("Warning: Failed to check if is_admin column exists: %v", err)
	} else if !columnExists {
		_, err = db.Exec("ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT 0")
		if err != nil {
			log.Printf("Warning: Failed to add is_admin column: %v", err)
		} else {
			log.Println("Successfully added is_admin column to users table")
		}
	}

	// 题库表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS question_banks (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	)`)
	if err != nil {
		log.Fatal("Failed to create question_banks table:", err)
	}

	// 题目表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS questions (
		id VARCHAR(255) PRIMARY KEY,
		bank_id VARCHAR(255) NOT NULL,
		question TEXT NOT NULL,
		options JSON NOT NULL,
		answer JSON NOT NULL,
		is_multiple BOOLEAN DEFAULT 0,
		explanation TEXT,
		FOREIGN KEY (bank_id) REFERENCES question_banks (id) ON DELETE CASCADE
	)`)
	if err != nil {
		log.Fatal("Failed to create questions table:", err)
	}

	// 检查并升级answer字段为JSON类型（如果还是INT类型）
	var answerColumnType string
	err = db.QueryRow("SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'questions' AND COLUMN_NAME = 'answer'").Scan(&answerColumnType)
	if err == nil && answerColumnType == "int" {
		log.Println("检测到answer字段为INT类型，正在升级为JSON类型...")
		// 先备份数据，然后删除旧字段，创建新字段
		// 注意：这会导致数据丢失，但这是必要的升级步骤
		_, err = db.Exec("ALTER TABLE questions MODIFY COLUMN answer JSON NOT NULL")
		if err != nil {
			log.Printf("警告: 升级answer字段失败: %v\n", err)
			log.Println("如果升级失败，请手动执行: ALTER TABLE questions MODIFY COLUMN answer JSON NOT NULL")
		} else {
			log.Println("成功升级answer字段为JSON类型")
		}
	}

	// 检查并添加is_multiple字段（如果不存在）
	var isMultipleExists bool
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'questions' AND COLUMN_NAME = 'is_multiple'").Scan(&isMultipleExists)
	if err != nil {
		log.Printf("Warning: Failed to check if is_multiple column exists: %v", err)
	} else if !isMultipleExists {
		_, err = db.Exec("ALTER TABLE questions ADD COLUMN is_multiple BOOLEAN DEFAULT 0")
		if err != nil {
			log.Printf("Warning: Failed to add is_multiple column: %v", err)
		} else {
			log.Println("Successfully added is_multiple column to questions table")
		}
	}

	// 错题表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wrong_questions (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		bank_id VARCHAR(255) NOT NULL,
		question_id VARCHAR(255) NOT NULL,
		question TEXT NOT NULL,
		options JSON NOT NULL,
		answer JSON NOT NULL,
		is_multiple BOOLEAN DEFAULT 0,
		explanation TEXT,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (bank_id) REFERENCES question_banks (id) ON DELETE CASCADE
	)`)
	if err != nil {
		log.Fatal("Failed to create wrong_questions table:", err)
	}

	// 检查并升级wrong_questions表的answer字段为JSON类型
	err = db.QueryRow("SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wrong_questions' AND COLUMN_NAME = 'answer'").Scan(&answerColumnType)
	if err == nil && answerColumnType == "int" {
		log.Println("检测到wrong_questions表的answer字段为INT类型，正在升级为JSON类型...")
		_, err = db.Exec("ALTER TABLE wrong_questions MODIFY COLUMN answer JSON NOT NULL")
		if err != nil {
			log.Printf("警告: 升级wrong_questions表的answer字段失败: %v\n", err)
		} else {
			log.Println("成功升级wrong_questions表的answer字段为JSON类型")
		}
	}

	// 检查并添加is_multiple字段（如果不存在）
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wrong_questions' AND COLUMN_NAME = 'is_multiple'").Scan(&isMultipleExists)
	if err != nil {
		log.Printf("Warning: Failed to check if is_multiple column exists in wrong_questions: %v", err)
	} else if !isMultipleExists {
		_, err = db.Exec("ALTER TABLE wrong_questions ADD COLUMN is_multiple BOOLEAN DEFAULT 0")
		if err != nil {
			log.Printf("Warning: Failed to add is_multiple column to wrong_questions: %v", err)
		} else {
			log.Println("Successfully added is_multiple column to wrong_questions table")
		}
	}

	// 考试结果表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS exam_results (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		bank_id VARCHAR(255) NOT NULL,
		score INT NOT NULL,
		correct_count INT NOT NULL,
		wrong_count INT NOT NULL,
		total_questions INT NOT NULL,
		total_time INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (bank_id) REFERENCES question_banks (id) ON DELETE CASCADE
	)`)
	if err != nil {
		log.Fatal("Failed to create exam_results table:", err)
	}
}

// 工具函数
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	fmt.Printf("Debug: Comparing password '%s' with hash '%s'\n", password, hash)
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		fmt.Printf("Debug: Password comparison failed: %v\n", err)
		return false
	}
	return true
}

func generateToken(userID, username string) (string, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access token required"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// 管理员中间件
func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")

		var isAdmin bool
		err := db.QueryRow("SELECT is_admin FROM users WHERE id = ?", userID).Scan(&isAdmin)
		if err != nil || !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// 认证相关处理函数
func register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", req.Username).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	// 加密密码
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	fmt.Printf("Debug: Password '%s' hashed to '%s'\n", req.Password, hashedPassword)

	// 创建用户
	userID := generateUUID()
	_, err = db.Exec("INSERT INTO users (id, username, password, email) VALUES (?, ?, ?, ?)",
		userID, req.Username, hashedPassword, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// 生成JWT token
	token, err := generateToken(userID, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User registered successfully",
		"token":   token,
		"user": gin.H{
			"id":       userID,
			"username": req.Username,
			"email":    req.Email,
		},
	})
}

func login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找用户
	var user User
	var email sql.NullString
	err := db.QueryRow("SELECT id, username, password, email, is_admin FROM users WHERE username = ?", req.Username).
		Scan(&user.ID, &user.Username, &user.Password, &email, &user.IsAdmin)
	if err != nil {
		log.Printf("Database query error for user '%s': %v", req.Username, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 处理可能为NULL的email
	if email.Valid {
		user.Email = email.String
	} else {
		user.Email = ""
	}

	// 验证密码
	if !checkPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 生成JWT token
	token, err := generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

func getCurrentUser(c *gin.Context) {
	userID := c.GetString("userID")

	var user User
	var email sql.NullString
	err := db.QueryRow("SELECT id, username, email, is_admin, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Username, &email, &user.IsAdmin, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 处理可能为NULL的email
	if email.Valid {
		user.Email = email.String
	} else {
		user.Email = ""
	}

	c.JSON(http.StatusOK, user)
}

// 管理员功能处理函数
func getAllUsers(c *gin.Context) {
	rows, err := db.Query("SELECT id, username, email, is_admin, created_at FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.IsAdmin, &user.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan user"})
			return
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, users)
}

func getAllQuestionBanks(c *gin.Context) {
	rows, err := db.Query("SELECT id, user_id, name, description, created_at FROM question_banks")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var banks []QuestionBank
	for rows.Next() {
		var bank QuestionBank
		err := rows.Scan(&bank.ID, &bank.UserID, &bank.Name, &bank.Description, &bank.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan question bank"})
			return
		}
		banks = append(banks, bank)
	}

	c.JSON(http.StatusOK, banks)
}

func deleteUser(c *gin.Context) {
	userID := c.Param("id")

	_, err := db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func deleteQuestionBankAdmin(c *gin.Context) {
	bankID := c.Param("id")

	_, err := db.Exec("DELETE FROM question_banks WHERE id = ?", bankID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete question bank"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question bank deleted successfully"})
}

func updateUserAdmin(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		IsAdmin bool `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE users SET is_admin = ? WHERE id = ?", req.IsAdmin, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// 管理员统计信息
func getAdminStats(c *gin.Context) {
	var stats struct {
		TotalUsers          int `json:"total_users"`
		TotalQuestionBanks  int `json:"total_question_banks"`
		TotalQuestions      int `json:"total_questions"`
		TotalExamResults    int `json:"total_exam_results"`
		TotalWrongQuestions int `json:"total_wrong_questions"`
	}

	// 获取用户总数
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user count"})
		return
	}

	// 获取题库总数
	err = db.QueryRow("SELECT COUNT(*) FROM question_banks").Scan(&stats.TotalQuestionBanks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get question bank count"})
		return
	}

	// 获取题目总数
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&stats.TotalQuestions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get question count"})
		return
	}

	// 获取考试结果总数
	err = db.QueryRow("SELECT COUNT(*) FROM exam_results").Scan(&stats.TotalExamResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exam result count"})
		return
	}

	// 获取错题总数
	err = db.QueryRow("SELECT COUNT(*) FROM wrong_questions").Scan(&stats.TotalWrongQuestions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wrong question count"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// 获取系统设置
func getSettings(c *gin.Context) {
	settings := make(map[string]string)
	
	rows, err := db.Query("SELECT setting_key, setting_value FROM system_settings")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设置失败"})
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		settings[key] = value
	}
	
	// 对敏感信息进行掩码处理
	if secretKey, ok := settings["tencent_secret_key"]; ok && secretKey != "" {
		settings["tencent_secret_key"] = maskSecretKey(secretKey)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}

// 更新系统设置
func updateSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 保存设置到数据库
	for key, value := range settings {
		valueStr := fmt.Sprintf("%v", value)
		
		// 对于敏感信息（secret_key），如果值是掩码格式（包含***），则不更新
		if key == "tencent_secret_key" {
			if valueStr == "" || strings.Contains(valueStr, "***") || len(valueStr) <= 6 {
				continue // 跳过，不更新密钥
			}
		}
		
		err := updateSystemSetting(key, valueStr)
		if err != nil {
			log.Printf("Failed to update setting %s: %v", key, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新设置 %s 失败", key)})
			return
		}
	}

	// 如果更新了腾讯云配置，重新加载配置
	tencentKeys := []string{"tencent_secret_id", "tencent_secret_key", "tencent_region", "tencent_model", "tencent_endpoint"}
	needsReload := false
	for _, key := range tencentKeys {
		if _, ok := settings[key]; ok {
			needsReload = true
			break
		}
	}
	
	if needsReload {
		reloadTencentCloudConfig()
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "设置保存成功",
	})
}


func handleOptions(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
	c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Status(http.StatusNoContent)
}

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using default values")
	}

	// 检查是否是调试模式（不初始化数据库，直接从环境变量读取配置）
	if len(os.Args) > 1 && os.Args[1] == "debug-api" {
		debugTencentAPI()
		return
	}

	// 初始化数据库
	initDB()
	defer db.Close()

	// 初始化腾讯云AI服务（如果配置了SecretId和SecretKey）
	initTencentCloudAI()
	if tencentCloudSecretId != "" && tencentCloudSecretKey != "" {
		log.Println("AI服务已初始化（腾讯云混元大模型）")
	} else {
		log.Println("Warning: 腾讯云SecretId和SecretKey未配置，AI识别功能将被禁用")
	}

	// 使用setupRoutes()函数设置路由
	r := setupRoutes()

	// 启动服务器
	port := ":3005"
	log.Printf("Server is running on http://localhost%s", port)
	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

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
	Options     []string `json:"options" db:"options"`
	Answer      int      `json:"answer" db:"answer"`
	Explanation string   `json:"explanation" db:"explanation"`
}

type WrongQuestion struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	BankID      string    `json:"bank_id" db:"bank_id"`
	QuestionID  string    `json:"question_id" db:"question_id"`
	Question    string    `json:"question" db:"question"`
	Options     []string  `json:"options" db:"options"`
	Answer      int       `json:"answer" db:"answer"`
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

	log.Println("Database initialized successfully")
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
		answer INT NOT NULL,
		explanation TEXT,
		FOREIGN KEY (bank_id) REFERENCES question_banks (id) ON DELETE CASCADE
	)`)
	if err != nil {
		log.Fatal("Failed to create questions table:", err)
	}

	// 错题表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wrong_questions (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		bank_id VARCHAR(255) NOT NULL,
		question_id VARCHAR(255) NOT NULL,
		question TEXT NOT NULL,
		options JSON NOT NULL,
		answer INT NOT NULL,
		explanation TEXT,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (bank_id) REFERENCES question_banks (id) ON DELETE CASCADE
	)`)
	if err != nil {
		log.Fatal("Failed to create wrong_questions table:", err)
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

// 更新系统设置
func updateSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 这里可以将设置保存到数据库或配置文件中
	// 当前简单实现：只返回成功
	// 后续可以添加数据库存储功能
	
	c.JSON(http.StatusOK, gin.H{
		"message": "设置保存成功",
		"settings": settings,
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

	// 初始化数据库
	initDB()
	defer db.Close()

	// 使用setupRoutes()函数设置路由
	r := setupRoutes()

	// 启动服务器
	port := ":3005"
	log.Printf("Server is running on http://localhost%s", port)
	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

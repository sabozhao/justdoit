package main

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ==================== 个人分类管理 API ====================

// 获取当前用户的所有个人分类（树形结构）
func getCategories(c *gin.Context) {
	userID := c.GetString("userID")

	// 查询当前用户的所有个人分类
	rows, err := db.Query("SELECT id, parent_id, name, description, sort_order, created_at, updated_at FROM categories WHERE user_id = ? ORDER BY sort_order ASC, created_at ASC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var allCategories []Category
	for rows.Next() {
		var cat Category
		var parentID sql.NullString
		err := rows.Scan(&cat.ID, &parentID, &cat.Name, &cat.Description, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if parentID.Valid {
			cat.ParentID = &parentID.String
		}
		allCategories = append(allCategories, cat)
	}

	// 构建树形结构（支持多层级）
	categoryMap := make(map[string]*Category)

	// 第一遍：创建所有分类的映射（使用指针），并初始化Children切片
	for i := range allCategories {
		allCategories[i].Children = []Category{}
		categoryMap[allCategories[i].ID] = &allCategories[i]
	}

	// 第二遍：递归构建所有父子关系（支持多层级）
	var buildCategoryTree func(catID string) Category
	buildCategoryTree = func(catID string) Category {
		cat := categoryMap[catID]
		if cat == nil {
			return Category{}
		}
		
		result := Category{
			ID:          cat.ID,
			ParentID:    cat.ParentID,
			Name:        cat.Name,
			Description: cat.Description,
			SortOrder:   cat.SortOrder,
			CreatedAt:   cat.CreatedAt,
			UpdatedAt:   cat.UpdatedAt,
			Children:    []Category{},
		}
		
		// 递归查找并构建所有直接子分类
		for i := range allCategories {
			if allCategories[i].ParentID != nil && *allCategories[i].ParentID == catID {
				// 递归构建子分类的完整树结构
				childTree := buildCategoryTree(allCategories[i].ID)
				result.Children = append(result.Children, childTree)
			}
		}
		
		return result
	}

	// 构建rootCategories（只包含根分类，但包含完整的子分类树）
	var rootCategories []Category
	for i := range allCategories {
		if allCategories[i].ParentID == nil || *allCategories[i].ParentID == "" {
			rootCategories = append(rootCategories, buildCategoryTree(allCategories[i].ID))
		}
	}

	c.JSON(http.StatusOK, rootCategories)
}

// 创建个人分类
func createCategory(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		ParentID    *string `json:"parent_id"`    // 父分类ID（可选）
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		SortOrder   int     `json:"sort_order"`    // 排序顺序
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果指定了父分类，验证父分类是否存在且属于当前用户
	if req.ParentID != nil && *req.ParentID != "" {
		var parentUserID string
		err := db.QueryRow("SELECT user_id FROM categories WHERE id = ?", *req.ParentID).Scan(&parentUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "指定的父分类不存在"})
			return
		}
		if parentUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该父分类"})
			return
		}
	}

	// 插入个人分类
	categoryID := generateUUID()
	_, err := db.Exec("INSERT INTO categories (id, user_id, parent_id, name, description, sort_order) VALUES (?, ?, ?, ?, ?, ?)",
		categoryID, userID, req.ParentID, req.Name, req.Description, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分类失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          categoryID,
		"name":        req.Name,
		"description": req.Description,
		"message":     "分类创建成功",
	})
}

// 更新个人分类
func updateCategory(c *gin.Context) {
	userID := c.GetString("userID")
	categoryID := c.Param("id")

	var req struct {
		ParentID    *string `json:"parent_id"`
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		SortOrder   int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查分类是否存在且属于当前用户
	var currentUserID string
	err := db.QueryRow("SELECT user_id FROM categories WHERE id = ?", categoryID).Scan(&currentUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
		return
	}
	if currentUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权更新此分类"})
		return
	}

	// 如果指定了父分类，验证父分类是否存在且不是自己，且属于当前用户
	if req.ParentID != nil && *req.ParentID != "" {
		if *req.ParentID == categoryID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能将自己设置为父分类"})
			return
		}
		var parentUserID string
		err := db.QueryRow("SELECT user_id FROM categories WHERE id = ?", *req.ParentID).Scan(&parentUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "指定的父分类不存在"})
			return
		}
		if parentUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该父分类"})
			return
		}
	}

	// 更新个人分类
	_, err = db.Exec("UPDATE categories SET parent_id = ?, name = ?, description = ?, sort_order = ? WHERE id = ?",
		req.ParentID, req.Name, req.Description, req.SortOrder, categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新分类失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分类更新成功"})
}

// 删除个人分类
func deleteCategory(c *gin.Context) {
	userID := c.GetString("userID")
	categoryID := c.Param("id")

	// 检查分类是否存在且属于当前用户
	var currentUserID string
	err := db.QueryRow("SELECT user_id FROM categories WHERE id = ?", categoryID).Scan(&currentUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
		return
	}
	if currentUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此分类"})
		return
	}

	// 检查是否有子分类
	var childCount int
	err = db.QueryRow("SELECT COUNT(*) FROM categories WHERE parent_id = ? AND user_id = ?", categoryID, userID).Scan(&childCount)
	if err == nil && childCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该分类下还有子分类，无法删除"})
		return
	}

	// 检查是否有题库使用该分类
	var bankCount int
	err = db.QueryRow("SELECT COUNT(*) FROM question_banks WHERE category_id = ?", categoryID).Scan(&bankCount)
	if err == nil && bankCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该分类下还有题库，无法删除"})
		return
	}

	// 删除个人分类
	_, err = db.Exec("DELETE FROM categories WHERE id = ?", categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除分类失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分类删除成功"})
}

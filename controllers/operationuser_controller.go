package controllers

import (
	crand "crypto/rand" // 别名crand，避免与math/rand冲突
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	mrand "math/rand" // 别名mrand，区分crypto/rand
	"net/http"
	"regexp"
	"strings"
	"time"

	"django_to_go/db"
	"django_to_go/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// OperationUserController 运营用户控制器
type OperationUserController struct{}

// AddServiceUser 添加客服用户 - 与Django的add_service_user函数完全匹配
func (ouc *OperationUserController) AddServiceUser(c *gin.Context) {
	// 生成唯一请求ID用于追踪
	requestID := generateRequestID()
	log.Printf("add_service_user request received, request_id=%s", requestID)

	// 检查请求方法
	if c.Request.Method != "POST" {
		log.Printf("add_service_user received non-POST request, request_id=%s", requestID)
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Method not allowed",
		})
		return
	}

	// 解析请求体JSON数据 - 移到重试循环外，确保只读取一次请求体
	var requestData struct {
		Nickname string `json:"nickname" binding:"required"`
		Mobile   string `json:"mobile" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Printf("Invalid JSON format in add_service_user request, request_id=%s, error=%v", requestID, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON format",
		})
		return
	}

	log.Printf("add_service_user parameters: nickname=%s, mobile=%s, request_id=%s",
		requestData.Nickname, requestData.Mobile, requestID)

	// 验证必填字段
	if requestData.Nickname == "" || requestData.Mobile == "" || requestData.Password == "" {
		log.Printf("Missing required fields in add_service_user, request_id=%s", requestID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "Missing required fields",
			"required": []string{"nickname", "mobile", "password"},
		})
		return
	}

	maxRetries := 3
	retryCount := 0
	for retryCount < maxRetries {
		tryAgain := false
		retryCount++

		// 检查手机号是否已存在
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM Customer_service_user WHERE mobile = ?)"
		err := db.DB.Raw(query, requestData.Mobile).Scan(&exists).Error
		if err != nil {
			log.Printf("Database error checking mobile existence, request_id=%s, error=%v", requestID, err)
			tryAgain = true
		} else if exists {
			log.Printf("Mobile number already exists: %s, request_id=%s", requestData.Mobile, requestID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Mobile number already exists",
			})
			return
		}

		// 如果需要重试，跳过剩余步骤
		if tryAgain {
			continue
		}

		// 密码哈希处理
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Password hashing failed, request_id=%s, error=%v", requestID, err)
			tryAgain = true
		}

		// 如果需要重试，跳过剩余步骤
		if tryAgain {
			continue
		}

		// 创建客服用户
		user := models.DjangoCustomerServiceUser{
			Nickname: requestData.Nickname,
			Mobile:   requestData.Mobile,
			Password: string(hashedPassword),
		}

		// 获取数据库连接
		sqlDB, err := db.DB.DB()
		if err != nil {
			log.Printf("获取数据库连接失败: %v, request_id=%s", err, requestID)
			tryAgain = true
		} else {
			// 开始原生SQL事务
			sqlTx, err := sqlDB.Begin()
			if err != nil {
				log.Printf("Failed to begin sql transaction, request_id=%s, error=%v", requestID, err)
				tryAgain = true
			} else {
				// 标记事务已开始，需要在错误处理中回滚
				txStarted := true
				defer func() {
					// 确保如果事务已开始但未提交，会被回滚
					if txStarted {
						sqlTx.Rollback()
					}
				}()

				// 调用BeforeSave钩子函数生成user_id
				if err := user.BeforeSave(sqlTx); err != nil {
					log.Printf("Failed to generate user_id, request_id=%s, error=%v", requestID, err)
					tryAgain = true
				} else {
					// 插入用户数据
					_, err = sqlTx.Exec("INSERT INTO Customer_service_user (user_id, nickname, mobile, password) VALUES (?, ?, ?, ?)",
						user.UserID, user.Nickname, user.Mobile, user.Password)
					if err != nil {
						log.Printf("Insert user failed, request_id=%s, error=%v", requestID, err)
						if strings.Contains(err.Error(), "duplicate key") {
							log.Printf("Integrity error in add_service_user, request_id=%s, error=%v", requestID, err)
							c.JSON(http.StatusBadRequest, gin.H{
								"error": "Mobile number or nickname already exists",
							})
							return
						}
						tryAgain = true
					} else {
						// 提交事务
						if err := sqlTx.Commit(); err != nil {
							log.Printf("Failed to commit transaction, request_id=%s, error=%v", requestID, err)
							tryAgain = true
						} else {
							// 标记事务已成功提交
							txStarted = false
							// 成功，返回响应
							log.Printf("Customer service user created successfully: %s, request_id=%s", user.UserID, requestID)
							c.JSON(http.StatusCreated, gin.H{
								"success": true,
								"message": "Customer service user created successfully",
								"user_id": user.UserID,
							})
							return
						}
					}
				}
			}
		}

		// 指数退避
		if tryAgain && retryCount < maxRetries {
			// 确保位移操作在整数上下文中进行
			shiftValue := 1 << uint(retryCount-1)
			sleepTime := 0.5 * float64(shiftValue) // 指数退避
			log.Printf("Retrying add_service_user after %.2f seconds, request_id=%s, attempt=%d/%d",
				sleepTime, requestID, retryCount, maxRetries)
			time.Sleep(time.Duration(sleepTime * float64(time.Second)))
		}
	}

	// 所有重试都失败
	log.Printf("All attempts failed in add_service_user, request_id=%s", requestID)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Server error occurred after multiple attempts",
	})
}

// AddOperationUser 添加运营用户 - 与Django的add_operation_user函数完全匹配
func (ouc *OperationUserController) AddOperationUser(c *gin.Context) {
	// 生成唯一请求ID用于追踪
	requestID := generateRequestID()
	log.Printf("add_operation_user request received, request_id=%s", requestID)

	// 检查请求方法
	if c.Request.Method != "POST" {
		log.Printf("add_operation_user received non-POST request, request_id=%s", requestID)
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Method not allowed",
		})
		return
	}

	// 解析请求体JSON数据 - 移到重试循环外，确保只读取一次请求体
	// 使用接口类型处理Level字段，支持string和int类型
	var requestData struct {
		Nickname string      `json:"nickname" binding:"required"`
		Mobile   string      `json:"mobile" binding:"required"`
		Password string      `json:"password" binding:"required"`
		Level    interface{} `json:"level" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Printf("Invalid JSON format in add_operation_user request, request_id=%s, error=%v", requestID, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON format",
		})
		return
	}

	// 处理Level字段，转换为int类型
	var level int
	switch v := requestData.Level.(type) {
	case int:
		level = v
	case float64: // JSON数字默认解析为float64
		level = int(v)
	case string:
		// 尝试将字符串转换为整数
		if _, err := fmt.Sscanf(v, "%d", &level); err != nil {
			log.Printf("Invalid level format, must be a number, request_id=%s", requestID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid level format, must be a number",
			})
			return
		}
	default:
		log.Printf("Invalid level type, request_id=%s", requestID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid level type",
		})
		return
	}

	log.Printf("add_operation_user parameters: nickname=%s, mobile=%s, level=%d, request_id=%s",
		requestData.Nickname, requestData.Mobile, level, requestID)

	// 验证必填字段
	if requestData.Nickname == "" || requestData.Mobile == "" || requestData.Password == "" || level == 0 {
		log.Printf("Missing required fields in add_operation_user, request_id=%s", requestID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "Missing required fields",
			"required": []string{"nickname", "mobile", "password", "level"},
		})
		return
	}

	maxRetries := 3
	retryCount := 0
	for retryCount < maxRetries {
		tryAgain := false
		retryCount++

		// 检查手机号是否已存在
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM Operation_user WHERE mobile = ?)"
		err := db.DB.Raw(query, requestData.Mobile).Scan(&exists).Error
		if err != nil {
			log.Printf("Database error checking mobile existence, request_id=%s, error=%v", requestID, err)
			tryAgain = true
		} else if exists {
			log.Printf("Mobile number already exists: %s, request_id=%s", requestData.Mobile, requestID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Mobile number already exists",
			})
			return
		}

		// 如果需要重试，跳过剩余步骤
		if tryAgain {
			continue
		}

		// 密码哈希处理
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Password hashing failed, request_id=%s, error=%v", requestID, err)
			tryAgain = true
		}

		// 如果需要重试，跳过剩余步骤
		if tryAgain {
			continue
		}

		// 创建运营用户
		user := models.DjangoOperationUser{
			Nickname: requestData.Nickname,
			Mobile:   requestData.Mobile,
			Password: string(hashedPassword),
			Level:    level,
		}

		// 获取数据库连接
		sqlDB, err := db.DB.DB()
		if err != nil {
			log.Printf("获取数据库连接失败: %v, request_id=%s", err, requestID)
			tryAgain = true
		} else {
			// 开始原生SQL事务
			sqlTx, err := sqlDB.Begin()
			if err != nil {
				log.Printf("Failed to begin sql transaction, request_id=%s, error=%v", requestID, err)
				tryAgain = true
			} else {
				// 标记事务已开始，需要在错误处理中回滚
				txStarted := true
				defer func() {
					// 确保如果事务已开始但未提交，会被回滚
					if txStarted {
						sqlTx.Rollback()
					}
				}()

				// 调用BeforeSave钩子函数生成user_id
				if err := user.BeforeSave(sqlTx); err != nil {
					log.Printf("Failed to generate user_id, request_id=%s, error=%v", requestID, err)
					tryAgain = true
				} else {
					// 插入用户数据
					_, err = sqlTx.Exec("INSERT INTO Operation_user (user_id, nickname, mobile, password, level) VALUES (?, ?, ?, ?, ?)",
						user.UserID, user.Nickname, user.Mobile, user.Password, user.Level)
					if err != nil {
						log.Printf("Insert user failed, request_id=%s, error=%v", requestID, err)
						if strings.Contains(err.Error(), "duplicate key") {
							log.Printf("Integrity error in add_operation_user, request_id=%s, error=%v", requestID, err)
							c.JSON(http.StatusBadRequest, gin.H{
								"error": "Mobile number or nickname already exists",
							})
							return
						}
						tryAgain = true
					} else {
						// 提交事务
						if err := sqlTx.Commit(); err != nil {
							log.Printf("Failed to commit transaction, request_id=%s, error=%v", requestID, err)
							tryAgain = true
						} else {
							// 标记事务已成功提交
							txStarted = false
							// 成功，返回响应
							log.Printf("Operation user created successfully: %s, request_id=%s", user.UserID, requestID)
							c.JSON(http.StatusCreated, gin.H{
								"success": true,
								"message": "Operation user created successfully",
								"user_id": user.UserID,
							})
							return
						}
					}
				}
			}
		}

		// 指数退避
		if tryAgain && retryCount < maxRetries {
			// 确保位移操作在整数上下文中进行
			shiftValue := 1 << uint(retryCount-1)
			sleepTime := 0.5 * float64(shiftValue) // 指数退避
			log.Printf("Retrying add_operation_user after %.2f seconds, request_id=%s, attempt=%d/%d",
				sleepTime, requestID, retryCount, maxRetries)
			time.Sleep(time.Duration(sleepTime * float64(time.Second)))
		}
	}

	// 所有重试都失败
	log.Printf("All attempts failed in add_operation_user, request_id=%s", requestID)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Server error occurred after multiple attempts",
	})
}

// VerificationStatus 验证登录状态 - 与Django的verification_status函数完全匹配
func (ouc *OperationUserController) VerificationStatus(c *gin.Context) {
	// 解析请求体数据
	var requestData struct {
		Mobile    string `json:"mobile" binding:"required"`
		Password  string `json:"password" binding:"required"`
		ObjectNum string `json:"object_num" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的JSON格式",
		})
		return
	}

	// 验证必填字段
	if requestData.Mobile == "" || requestData.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "缺少必要参数",
			"required": []string{"mobile", "password"},
		})
		return
	}

	// 验证手机号格式
	mobileRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !mobileRegex.MatchString(requestData.Mobile) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "手机号格式错误",
		})
		return
	}

	// 确定用户类型
	var tableName string
	if requestData.ObjectNum == "1" {
		tableName = "Operation_user"
	} else if requestData.ObjectNum == "2" {
		tableName = "Customer_service_user"
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "object_num参数错误",
		})
		return
	}

	// 查询用户
	maxRetries := 3
	retryCount := 0
	var userID, nickname, password string
	for retryCount < maxRetries {
		retryCount++
		log.Printf("开始查询用户(尝试%d/%d): mobile=%s", retryCount, maxRetries, requestData.Mobile)

		// 查询用户信息
		query := fmt.Sprintf("SELECT user_id, nickname, password FROM %s WHERE mobile = ?", tableName)
		var err error
		if db.DB != nil {
			sqlDB, err := db.DB.DB()
			if err != nil {
				log.Printf("获取数据库连接失败: %v", err)
				err = fmt.Errorf("database connection error")
			} else {
				err = sqlDB.QueryRow(query, requestData.Mobile).Scan(&userID, &nickname, &password)
			}
		} else {
			err = fmt.Errorf("database instance not initialized")
		}
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("手机号未注册: %s", requestData.Mobile)
				c.JSON(http.StatusNotFound, gin.H{
					"error": "手机号未注册",
				})
				return
			}
			log.Printf("用户查询异常(尝试%d/%d): %v", retryCount, maxRetries, err)
			if retryCount >= maxRetries {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "用户信息查询失败，请稍后重试",
				})
				return
			}
			// 指数退避
			// 确保位移操作在整数上下文中进行
			shiftValue := 1 << uint(retryCount-1)
			sleepTime := 0.5 * float64(shiftValue) // 指数退避
			log.Printf("等待%.2f秒后重试...", sleepTime)
			time.Sleep(time.Duration(sleepTime * float64(time.Second)))
			continue
		}
		break
	}

	// 确保用户对象已正确获取
	if userID == "" {
		log.Printf("所有重试均失败，用户对象未获取")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "系统繁忙，请稍后再试",
		})
		return
	}

	// 验证密码
	log.Printf("开始密码验证: user_id=%s", userID)
	err := bcrypt.CompareHashAndPassword([]byte(password), []byte(requestData.Password))
	if err != nil {
		log.Printf("密码验证失败: user_id=%s", userID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "账号存在但密码错误",
		})
		return
	}
	log.Printf("密码验证成功: user_id=%s", userID)

	// 验证成功
	responseData := gin.H{
		"success":  true,
		"message":  "登录状态验证成功",
		"user_id":  userID,
		"nickname": nickname,
	}
	log.Printf("生成成功响应: %v", responseData)
	c.JSON(http.StatusOK, responseData)
}

// ChangePassword 修改密码 - 与Django的change_password函数完全匹配
func (ouc *OperationUserController) ChangePassword(c *gin.Context) {
	// 生成唯一请求ID用于追踪
	requestID := generateRequestID()
	log.Printf("change_password request received, request_id=%s", requestID)

	// 检查请求方法
	if c.Request.Method != "POST" {
		log.Printf("change_password received non-POST request, request_id=%s", requestID)
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Method not allowed",
		})
		return
	}

	maxRetries := 3
	retryCount := 0
	for retryCount < maxRetries {
		tryAgain := false
		retryCount++

		// 解析请求体JSON数据
		var requestData struct {
			ObjectNum   int    `json:"object_num" binding:"required"`
			Mobile      string `json:"mobile" binding:"required"`
			OldPassword string `json:"old_password" binding:"required"`
			NewPassword string `json:"new_password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&requestData); err != nil {
			log.Printf("Invalid JSON format, request_id=%s, error=%v", requestID, err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JSON format",
			})
			return
		}

		// 验证必填字段
		if requestData.ObjectNum == 0 || requestData.Mobile == "" || requestData.OldPassword == "" || requestData.NewPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":    "Missing required fields",
				"required": []string{"object_num", "mobile", "old_password", "new_password"},
			})
			return
		}

		// 验证object_num
		if requestData.ObjectNum != 1 && requestData.ObjectNum != 2 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "object_num must be 1 or 2",
			})
			return
		}

		// 验证手机号格式
		mobileRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
		if !mobileRegex.MatchString(requestData.Mobile) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid mobile format",
			})
			return
		}

		// 确定用户表
		var tableName string
		if requestData.ObjectNum == 1 {
			tableName = "Operation_user"
		} else {
			tableName = "Customer_service_user"
		}

		// 查询用户
		var userID, currentPassword string
		var err error
		query := fmt.Sprintf("SELECT user_id, password FROM %s WHERE mobile = ?", tableName)
		if db.DB != nil {
			sqlDB, err := db.DB.DB()
			if err != nil {
				log.Printf("获取数据库连接失败: %v", err)
				err = fmt.Errorf("database connection error")
			} else {
				err = sqlDB.QueryRow(query, requestData.Mobile).Scan(&userID, &currentPassword)
			}
		} else {
			err = fmt.Errorf("database instance not initialized")
		}
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "User not found",
				})
				return
			}
			log.Printf("Database error querying user, request_id=%s, error=%v", requestID, err)
			tryAgain = true
		} else {
			// 验证旧密码
			err = bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(requestData.OldPassword))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Old password is incorrect",
				})
				return
			}

			// 如果需要重试，跳过剩余步骤
			if tryAgain {
				continue
			}

			// 哈希新密码
			newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.NewPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("Password hashing failed, request_id=%s, error=%v", requestID, err)
				tryAgain = true
			}

			// 如果需要重试，跳过剩余步骤
			if tryAgain {
				continue
			}

			// 更新密码
			tx := db.DB.Begin()
			if tx.Error != nil {
				log.Printf("Failed to begin transaction, request_id=%s, error=%v", requestID, tx.Error)
				tryAgain = true
			} else {
				updateQuery := fmt.Sprintf("UPDATE %s SET password = ? WHERE user_id = ?", tableName)
				err = tx.Exec(updateQuery, string(newHashedPassword), userID).Error
				if err != nil {
					log.Printf("Failed to update password, request_id=%s, error=%v", requestID, err)
					tx.Rollback()
					tryAgain = true
				} else {
					// 提交事务
					if err := tx.Commit(); err != nil {
						log.Printf("Failed to commit transaction, request_id=%s, error=%v", requestID, err)
						tx.Rollback()
						tryAgain = true
					} else {
						log.Printf("Password updated successfully for user: %s, request_id=%s", userID, requestID)
						c.JSON(http.StatusOK, gin.H{
							"success": true,
							"message": "Password updated successfully",
						})
						return
					}
				}
			}
		}

		// 指数退避
		if tryAgain && retryCount < maxRetries {
			// 确保位移操作在整数上下文中进行
			shiftValue := 1 << uint(retryCount-1)
			sleepTime := 0.5 * float64(shiftValue) // 指数退避
			log.Printf("Retrying change_password after %.2f seconds, request_id=%s, attempt=%d/%d",
				sleepTime, requestID, retryCount, maxRetries)
			time.Sleep(time.Duration(sleepTime * float64(time.Second)))
		}
	}

	// 所有重试都失败
	log.Printf("All attempts failed in change_password, request_id=%s", requestID)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Server error after multiple attempts",
	})
}

// generateRequestID 生成唯一的请求ID用于日志追踪
func generateRequestID() string {
	bytes := make([]byte, 16)
	// 使用crypto/rand生成安全随机数（优先方案）
	_, err := crand.Read(bytes)
	if err != nil {
		// 降级方案：初始化math/rand种子（避免随机数重复）
		mrand.Seed(time.Now().UnixNano())
		// 生成8位十六进制随机数（匹配%08x格式）
		randomNum := mrand.Intn(0x100000000) // 0x100000000 = 2^32，对应8位十六进制
		return fmt.Sprintf("%d%08x", time.Now().UnixNano(), randomNum)
	}
	// crypto/rand成功时，返回32位十六进制字符串（16字节=32位十六进制）
	return hex.EncodeToString(bytes)
}

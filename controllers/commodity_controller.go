package controllers

import (
	"django_to_go/db"
	"django_to_go/models"
	"django_to_go/utils"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CommodityController 商品控制器

type CommodityController struct{}

// CommodityList 获取商品列表
func (cc *CommodityController) CommodityList(c *gin.Context) {
	// 获取查询参数
	pageNum, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	category := c.Query("category")
	keyword := c.Query("keyword")

	// 计算偏移量
	offset := (pageNum - 1) * pageSize

	// 构建查询
	var commodities []models.Commodity
	query := db.DB.Model(&models.Commodity{})

	// 添加分类筛选
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// 添加关键词搜索
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 获取总数
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取商品总数失败",
		})
		return
	}

	// 执行分页查询
	if err := query.Offset(offset).Limit(pageSize).Find(&commodities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取商品列表失败",
		})
		return
	}

	// 格式化返回结果
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"items":     commodities,
			"total":     totalCount,
			"page":      pageNum,
			"page_size": pageSize,
		},
	})
}

// SearchStyleCodes 搜索款式编码名称

func (cc *CommodityController) SearchStyleCodes(c *gin.Context) {
	var requestData struct {
		Shopname      string `json:"shopname" binding:"required"`
		SearchKeyword string `json:"search_keyword" binding:"required"`
		Page          int    `json:"page" binding:"required,min=1"`
		PageSize      int    `json:"page_size" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据",
		})
		return
	}

	// 验证店铺名称
	if requestData.Shopname != "youlan_kids" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的店铺名称",
		})
		return
	}

	// 计算偏移量
	offset := (requestData.Page - 1) * requestData.PageSize

	// 查询款式编码数据
	var styleCodeDataList []models.StyleCodeData
	var totalCount int64

	query := db.DB.Model(&models.StyleCodeData{}).
		Where("name LIKE ?", "%"+requestData.SearchKeyword+"%").
		Order("style_code")

	// 获取总条数
	if err := query.Count(&totalCount).Error; err != nil {
		log.Printf("获取款式编码总数失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器内部错误",
		})
		return
	}

	// 获取分页数据
	if err := query.Offset(offset).Limit(requestData.PageSize).Find(&styleCodeDataList).Error; err != nil {
		log.Printf("搜索款式编码失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器内部错误",
		})
		return
	}

	// 构建响应数据
	result := make([]map[string]interface{}, 0, len(styleCodeDataList))
	// 获取请求的协议，考虑反向代理环境
	proto := utils.GetRequestProto(c)
	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)

	for _, styleData := range styleCodeDataList {
		styleInfo := make(map[string]interface{})
		styleInfo["style_code"] = styleData.StyleCode
		styleInfo["name"] = styleData.Name
		styleInfo["category"] = styleData.Category
		styleInfo["category_detail"] = styleData.CategoryDetail
		styleInfo["price"] = styleData.Price

		// 处理图片URL
		if styleData.Image != "" {
			styleInfo["image"] = utils.BuildFullImageURL(baseURL, styleData.Image, "media")
		} else {
			styleInfo["image"] = nil
		}

		// 格式化时间
		styleInfo["created_at"] = styleData.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05")
		styleInfo["updated_at"] = styleData.UpdatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05")

		result = append(result, styleInfo)
	}

	// 计算总页数
	totalPages := (totalCount + int64(requestData.PageSize) - 1) / int64(requestData.PageSize)

	c.JSON(http.StatusOK, gin.H{
		"code":         200,
		"message":      "查询成功",
		"data":         result,
		"total_count":  totalCount,
		"total_pages":  totalPages,
		"current_page": requestData.Page,
		"page_size":    requestData.PageSize,
	})
}

// AddGoods 增加商品并自动生成相关模型内容
func (cc *CommodityController) AddGoods(c *gin.Context) {
	var requestData struct {
		CommodityID string  `form:"commodity_id" binding:"required"`
		Name        string  `form:"name" binding:"required"`
		Price       float64 `form:"price" binding:"required,gt=0"`
		Category    string  `form:"category" binding:"required"`
		StyleCode   string  `form:"style_code"`
		Size        string  `form:"size"`
		Notes       string  `form:"notes"`
	}

	// 处理表单数据
	if err := c.ShouldBind(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 检查商品ID是否已存在
	var existingCommodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", requestData.CommodityID).First(&existingCommodity).Error; err == nil {
		// 商品ID已存在
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "商品ID已存在，请使用其他商品ID",
		})
		return
	}

	// 开始事务
	begin := db.DB.Begin()
	if begin.Error != nil {
		log.Printf("开启事务失败: %v", begin.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	// 如果提供了StyleCode，检查并创建StyleCodeData记录
	if requestData.StyleCode != "" {
		// 检查StyleCodeData是否已存在
		var styleCodeData models.StyleCodeData
		if err := begin.Where("style_code = ?", requestData.StyleCode).First(&styleCodeData).Error; err != nil {
			// StyleCodeData不存在，创建新记录
			styleCodeData = models.StyleCodeData{
				StyleCode:       requestData.StyleCode,
				Name:            requestData.Name,
				Category:        requestData.Category,
				Price:           requestData.Price,
				Image:           "",   // 暂时为空，实际应从上传文件中获取
				DisplayPictures: "{}", // 初始化为空JSON对象，符合MySQL JSON类型要求
			}

			if err := begin.Create(&styleCodeData).Error; err != nil {
				begin.Rollback()
				log.Printf("创建StyleCodeData失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "添加商品失败: 创建款式数据失败",
				})
				return
			}

			// 创建StyleCodeSituation记录
			styleCodeSituation := models.StyleCodeSituation{
				StyleCode: requestData.StyleCode,
				Status:    "pending", // 默认状态为待审核
			}

			if err := begin.Create(&styleCodeSituation).Error; err != nil {
				begin.Rollback()
				log.Printf("创建StyleCodeSituation失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "添加商品失败: 创建款式状态失败",
				})
				return
			}
		}
	}

	// 创建商品对象
	commodity := models.Commodity{
		CommodityID: requestData.CommodityID,
		Name:        requestData.Name,
		Price:       requestData.Price,
		Category:    requestData.Category,
		StyleCode:   requestData.StyleCode,
		Size:        requestData.Size,
		Notes:       requestData.Notes,
		// 确保image字段不为空
		Image: "",
	}

	// 处理文件上传
	var mainImageURL string
	var numImages int = 0
	mainImageFile, err := c.FormFile("image")
	if err == nil && mainImageFile != nil {
		// 获取协议
		proto := utils.GetRequestProto(c)
		// 构建基础URL
		baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)

		// 为每个商品ID创建独立的子文件夹
		commodityDir := filepath.Join("commodities", requestData.CommodityID)
		// 保存上传的文件
		imagePath, err := utils.SaveUploadedFile(c, mainImageFile, commodityDir, "commodity_main_")
		if err != nil {
			begin.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "上传主图失败: " + err.Error(),
			})
			return
		}
		// 实际保存文件到指定路径
		fullPath := filepath.Join("./media", imagePath)
		if err := c.SaveUploadedFile(mainImageFile, fullPath); err != nil {
			begin.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "保存主图失败: " + err.Error(),
			})
			return
		}
		// 构建完整的图片URL
		mainImageURL = utils.BuildFullImageURL(baseURL, imagePath, "media")
		// 更新商品的图片URL
		commodity.Image = mainImageURL
		// 设置图片数量
		numImages = 1

		// 如果提供了StyleCode，同时更新StyleCodeData的图片
		if requestData.StyleCode != "" {
			var styleCodeData models.StyleCodeData
			if err := begin.Where("style_code = ?", requestData.StyleCode).First(&styleCodeData).Error; err == nil {
				styleCodeData.Image = mainImageURL
				if err := begin.Save(&styleCodeData).Error; err != nil {
					begin.Rollback()
					log.Printf("更新款式图片失败: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"code":    500,
						"message": "添加商品失败: 更新款式图片失败",
					})
					return
				}
			}
		}
	}

	// 创建商品
	if err := begin.Create(&commodity).Error; err != nil {
		begin.Rollback()
		log.Printf("创建商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "添加商品失败: " + err.Error(),
		})
		return
	}

	// 创建商品状态记录
	commoditySituation := models.CommoditySituation{
		CommodityID: requestData.CommodityID,
		Status:      "pending",
	}

	if err := begin.Create(&commoditySituation).Error; err != nil {
		begin.Rollback()
		log.Printf("创建商品状态记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "添加商品失败: " + err.Error(),
		})
		return
	}

	// 提交事务
	if err := begin.Commit().Error; err != nil {
		begin.Rollback()
		log.Printf("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "添加商品失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "添加成功",
		"data": gin.H{
			"commodity_id": requestData.CommodityID,
			"image_count":  numImages,
		},
	})
}

// DeleteGoods 删除商品
func (cc *CommodityController) DeleteGoods(c *gin.Context) {
	var requestData struct {
		CommodityID interface{} `json:"commodity_id" form:"commodity_id" binding:"required"`
	}

	// 尝试从JSON和查询参数中绑定数据
	if err := c.ShouldBind(&requestData); err != nil {
		// 检查URL查询参数中是否有commodity_id
		commodityID := c.Query("commodity_id")
		if commodityID == "" {
			// 如果URL查询参数中也没有，返回详细错误信息
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "缺少commodity_id参数，请在请求体或URL查询参数中提供",
				"details": err.Error(),
			})
			return
		}
		// 使用URL查询参数中的commodity_id
		requestData.CommodityID = commodityID
	}

	// 将CommodityID转换为字符串
	commodityIDStr := ""
	log.Printf("接收到的commodity_id参数: %v, 类型: %T", requestData.CommodityID, requestData.CommodityID)
	switch v := requestData.CommodityID.(type) {
	case string:
		commodityIDStr = v
		log.Printf("commodity_id参数是字符串类型: %s", commodityIDStr)
	case int, int64:
		// 整数类型直接转换为字符串
		commodityIDStr = fmt.Sprintf("%d", v)
		log.Printf("commodity_id参数是整数类型，转换为字符串: %s", commodityIDStr)
	case float64:
		// 浮点数类型，判断是否为整数
		if v == float64(int64(v)) {
			// 如果是整数，转换为整数格式的字符串
			commodityIDStr = fmt.Sprintf("%.0f", v)
			log.Printf("commodity_id参数是浮点整数类型，转换为整数格式字符串: %s", commodityIDStr)
		} else {
			// 非整数浮点数，保持原样
			commodityIDStr = fmt.Sprintf("%v", v)
			log.Printf("commodity_id参数是浮点数类型，转换为字符串: %s", commodityIDStr)
		}
	default:
		log.Printf("commodity_id参数格式不正确，类型: %T", v)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "commodity_id参数格式不正确",
			"details": fmt.Sprintf("参数类型: %T", v),
		})
		return
	}

	// 开始事务
	begin := db.DB.Begin()
	if begin.Error != nil {
		log.Printf("开启事务失败: %v", begin.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "服务器处理失败",
			"details": begin.Error.Error(),
		})
		return
	}

	// 查询商品
	var commodity models.Commodity
	log.Printf("准备查询商品，commodity_id: %s", commodityIDStr)
	if err := begin.Where("commodity_id = ?", commodityIDStr).First(&commodity).Error; err != nil {
		begin.Rollback()
		log.Printf("查询商品失败，commodity_id: %s, 错误: %v", commodityIDStr, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "商品不存在",
			"details": fmt.Sprintf("查询ID: %s", commodityIDStr),
		})
		return
	}
	log.Printf("查询到商品: %+v", commodity)

	// 删除商品图片
	var commodityImages []models.CommodityImage
	begin.Where("commodity_id = ?", commodityIDStr).Find(&commodityImages)
	// 实际项目中应该删除文件

	// 删除数据库记录
	if err := begin.Where("commodity_id = ?", commodityIDStr).Delete(&models.CommodityImage{}).Error; err != nil {
		begin.Rollback()
		log.Printf("删除商品图片失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	if err := begin.Where("commodity_id = ?", commodityIDStr).Delete(&models.CommoditySituation{}).Error; err != nil {
		begin.Rollback()
		log.Printf("删除商品状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	if err := begin.Delete(&commodity).Error; err != nil {
		begin.Rollback()
		log.Printf("删除商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	// 提交事务
	if err := begin.Commit().Error; err != nil {
		begin.Rollback()
		log.Printf("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

// SearchCommodityData 查询商品信息
func (cc *CommodityController) SearchCommodityData(c *gin.Context) {
	var requestData struct {
		CommodityID interface{} `json:"commodity_id" form:"commodity_id" binding:"required"`
		DataList    []string    `json:"data_list"`
	}

	// 尝试从JSON和查询参数中绑定数据
	if err := c.ShouldBind(&requestData); err != nil {
		// 检查URL查询参数中是否有commodity_id
		commodityID := c.Query("commodity_id")
		if commodityID == "" {
			// 如果URL查询参数中也没有，返回详细错误信息
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "缺少commodity_id参数，请在请求体或URL查询参数中提供",
				"details": err.Error(),
			})
			return
		}
		// 使用URL查询参数中的commodity_id
		requestData.CommodityID = commodityID
	}

	// 将CommodityID转换为字符串
	commodityIDStr := ""
	log.Printf("接收到的commodity_id参数: %v, 类型: %T", requestData.CommodityID, requestData.CommodityID)
	switch v := requestData.CommodityID.(type) {
	case string:
		commodityIDStr = v
		log.Printf("commodity_id参数是字符串类型: %s", commodityIDStr)
	case int, int64:
		// 整数类型直接转换为字符串
		commodityIDStr = fmt.Sprintf("%d", v)
		log.Printf("commodity_id参数是整数类型，转换为字符串: %s", commodityIDStr)
	case float64:
		// 浮点数类型，判断是否为整数
		if v == float64(int64(v)) {
			// 如果是整数，转换为整数格式的字符串
			commodityIDStr = fmt.Sprintf("%.0f", v)
			log.Printf("commodity_id参数是浮点整数类型，转换为整数格式字符串: %s", commodityIDStr)
		} else {
			// 非整数浮点数，保持原样
			commodityIDStr = fmt.Sprintf("%v", v)
			log.Printf("commodity_id参数是浮点数类型，转换为字符串: %s", commodityIDStr)
		}
	default:
		log.Printf("commodity_id参数格式不正确，类型: %T", v)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "commodity_id参数格式不正确",
			"details": fmt.Sprintf("参数类型: %T", v),
		})
		return
	}

	// 查询商品
	var commodity models.Commodity
	log.Printf("准备查询商品，commodity_id: %s", commodityIDStr)
	if err := db.DB.Where("commodity_id = ?", commodityIDStr).First(&commodity).Error; err != nil {
		log.Printf("查询商品失败，commodity_id: %s, 错误: %v", commodityIDStr, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "商品不存在",
			"details": fmt.Sprintf("查询ID: %s", commodityIDStr),
		})
		return
	}
	log.Printf("查询到商品: %+v", commodity)

	// 构建响应数据
	result := make(map[string]interface{})
	// 获取请求的协议，考虑反向代理环境
	proto := utils.GetRequestProto(c)
	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)

	// 如果指定了字段列表，只返回指定的字段
	if len(requestData.DataList) > 0 {
		for _, field := range requestData.DataList {
			switch field {
			case "commodity_id":
				result[field] = commodity.CommodityID
			case "name":
				result[field] = commodity.Name
			case "style_code":
				result[field] = commodity.StyleCode
			case "category":
				result[field] = commodity.Category
			case "price":
				result[field] = commodity.Price
			case "size":
				result[field] = commodity.Size
			case "color":
				result[field] = commodity.Color
			case "image":
				if commodity.Image != "" {
					result[field] = utils.BuildFullImageURL(baseURL, commodity.Image, "media")
				} else {
					result[field] = nil
				}
			case "promo_image":
				if commodity.PromoImage != "" {
					result[field] = utils.BuildFullImageURL(baseURL, commodity.PromoImage, "media")
				} else {
					result[field] = nil
				}
			case "created_at":
				result[field] = commodity.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05")
			default:
				// 忽略未定义的字段
			}
		}
	}

	// 查询商品图片
	var commodityImages []models.CommodityImage
	if err := db.DB.Where("commodity_id = ?", requestData.CommodityID).Find(&commodityImages).Error; err != nil {
		log.Printf("获取商品图片失败: %v", err)
	}

	// 构建图片信息
	images := make([]map[string]interface{}, 0, len(commodityImages))
	var mainImage map[string]interface{}
	otherImages := make([]map[string]interface{}, 0)

	for _, img := range commodityImages {
		imgInfo := make(map[string]interface{})
		imgInfo["id"] = img.ID
		imgInfo["url"] = utils.BuildFullImageURL(baseURL, img.Image, "media")
		imgInfo["is_main"] = img.IsMain
		imgInfo["created_at"] = img.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05")

		images = append(images, imgInfo)

		if img.IsMain {
			mainImage = imgInfo
		} else {
			otherImages = append(otherImages, imgInfo)
		}
	}

	// 添加图片信息
	result["images"] = images
	result["main_image"] = mainImage
	result["other_images"] = otherImages

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GoodsQuery 商品查询
func (cc *CommodityController) GoodsQuery(c *gin.Context) {
	var requestData struct {
		Shopname  string      `json:"shopname" binding:"required"`
		Demand    string      `json:"demand"`
		StyleCode string      `json:"style_code"`
		Category  interface{} `json:"category"`
		Status    string      `json:"status"`
		Page      int         `json:"page"`
		PageSize  int         `json:"page_size"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的JSON格式",
		})
		return
	}

	// 验证店铺名称
	if requestData.Shopname != "youlan_kids" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的店铺名称",
		})
		return
	}

	// 设置默认分页参数
	if requestData.Page <= 0 {
		requestData.Page = 1
	}
	if requestData.PageSize <= 0 {
		requestData.PageSize = 20
	} else if requestData.PageSize > 50 {
		requestData.PageSize = 50
	}

	// 构建查询
	var commodities []models.Commodity
	var commoditiesPage []models.Commodity
	var total int64
	var totalPages int64
	query := db.DB.Model(&models.Commodity{}).Order("-created_at")

	// 处理特定需求
	if requestData.Demand == "style_code" || requestData.Demand == "goods" {
		// 获取在线的款式代码
		var onlineStyleCodes []string
		db.DB.Model(&models.StyleCodeSituation{}).
			Where("status = ?", "online").
			Pluck("style_code", &onlineStyleCodes)

		query = query.Where("style_code IN ?", onlineStyleCodes)

		// 根据style_code过滤
		if requestData.Demand != "style_code" && requestData.StyleCode != "" {
			query = query.Where("style_code = ?", requestData.StyleCode)
		}

		// 根据category过滤
		if requestData.Category != nil {
			if categoryList, ok := requestData.Category.([]interface{}); ok {
				stringList := make([]string, 0, len(categoryList))
				for _, cat := range categoryList {
					if strCat, ok := cat.(string); ok {
						stringList = append(stringList, strCat)
					}
				}
				query = query.Where("category IN ?", stringList)
			} else if strCat, ok := requestData.Category.(string); ok {
				query = query.Where("category = ?", strCat)
			}
		}
	}

	// 处理状态过滤
	if requestData.Status != "" {
		// 验证状态值（在实际项目中应该定义一个状态列表）
		validStatuses := []string{"online", "offline", "pending"}
		isValidStatus := false
		for _, s := range validStatuses {
			if s == requestData.Status {
				isValidStatus = true
				break
			}
		}

		if !isValidStatus {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "无效的状态值",
			})
			return
		}

		// 获取符合状态的商品ID列表
		var situationIDs []string
		db.DB.Model(&models.CommoditySituation{}).
			Where("status = ?", requestData.Status).
			Pluck("commodity_id", &situationIDs)

		query = query.Where("commodity_id IN ?", situationIDs)
	}

	// 根据category过滤（无论何种模式都应用）
	if requestData.Category != nil {
		if categoryList, ok := requestData.Category.([]interface{}); ok {
			stringList := make([]string, 0, len(categoryList))
			for _, cat := range categoryList {
				if strCat, ok := cat.(string); ok {
					stringList = append(stringList, strCat)
				}
			}
			query = query.Where("category IN ?", stringList)
		} else if strCat, ok := requestData.Category.(string); ok {
			query = query.Where("category = ?", strCat)
		}
	}

	// 根据模式选择不同的查询和分页策略
	if requestData.Demand == "style_code" {
		// style_code模式：相同style_code只返回一条记录
		// 首先获取所有符合条件的商品
		query.Find(&commodities)

		// 使用map去重，确保每个style_code只保留一条最新记录
		uniqueCommodities := []models.Commodity{}
		seenStyleCodes := make(map[string]bool)

		for _, commodity := range commodities {
			if !seenStyleCodes[commodity.StyleCode] {
				seenStyleCodes[commodity.StyleCode] = true
				uniqueCommodities = append(uniqueCommodities, commodity)
			}
		}

		// 手动处理分页
		total = int64(len(uniqueCommodities))
		start := (requestData.Page - 1) * requestData.PageSize
		end := start + requestData.PageSize

		// 确保end不越界
		if end > len(uniqueCommodities) {
			end = len(uniqueCommodities)
		}

		// 获取当前页数据
		if start < len(uniqueCommodities) {
			commoditiesPage = uniqueCommodities[start:end]
		} else {
			commoditiesPage = []models.Commodity{}
		}

		totalPages = (total + int64(requestData.PageSize) - 1) / int64(requestData.PageSize)
	} else {
		// 其他模式：标准分页处理
		query.Count(&total)
		offset := (requestData.Page - 1) * requestData.PageSize
		query.Offset(offset).Limit(requestData.PageSize).Find(&commoditiesPage)
		totalPages = (total + int64(requestData.PageSize) - 1) / int64(requestData.PageSize)
	}

	// 构建响应数据
	result := make([]map[string]interface{}, 0, len(commoditiesPage))
	// 获取请求的协议，考虑反向代理环境
	proto := utils.GetRequestProto(c)
	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)

	for _, commodity := range commoditiesPage {
		var goodsData map[string]interface{}

		// 根据demand参数决定返回数据格式
		if requestData.Demand == "style_code" || requestData.Demand == "goods" {
			// 对于style_code或goods需求，只返回指定字段
			goodsData = make(map[string]interface{})

			// 构建promo_image_url，当为空时使用image的值
			var promoImageURL string
			if commodity.PromoImage != "" {
				promoImageURL = utils.BuildFullImageURL(baseURL, commodity.PromoImage, "media")
			} else if commodity.Image != "" {
				promoImageURL = utils.BuildFullImageURL(baseURL, commodity.Image, "media")
			} else {
				// 查找第一个主图作为备用
				var mainImage models.CommodityImage
				db.DB.Where("commodity_id = ? AND is_main = ?", commodity.CommodityID, true).First(&mainImage)
				if mainImage.ID > 0 {
					promoImageURL = utils.BuildFullImageURL(baseURL, mainImage.Image, "media")
				} else {
					promoImageURL = ""
				}
			}

			goodsData["promo_image_url"] = promoImageURL
			goodsData["price"] = commodity.Price
			goodsData["name"] = commodity.Name
			goodsData["style_code"] = commodity.StyleCode
		} else {
			// 原始逻辑：返回所有字段
			goodsData = make(map[string]interface{})
			goodsData["commodity_id"] = commodity.CommodityID
			goodsData["name"] = commodity.Name
			goodsData["style"] = commodity.StyleCode
			goodsData["category"] = commodity.Category
			goodsData["price"] = commodity.Price

			// 构建图片URL
			if commodity.PromoImage != "" {
				goodsData["promo_image_url"] = utils.BuildFullImageURL(baseURL, commodity.PromoImage, "media")
			} else {
				goodsData["promo_image_url"] = nil
			}

			// 格式化时间
			goodsData["created_at"] = commodity.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05")

			// 构建图片列表
			var commodityImages []models.CommodityImage
			db.DB.Where("commodity_id = ?", commodity.CommodityID).Find(&commodityImages)

			imageURLs := make([]map[string]interface{}, 0, len(commodityImages))
			var mainImage map[string]interface{}
			otherImages := make([]map[string]interface{}, 0)

			for _, img := range commodityImages {
				imgInfo := make(map[string]interface{})
				imgInfo["id"] = img.ID
				imgInfo["url"] = utils.BuildFullImageURL(baseURL, img.Image, "media")
				imgInfo["is_main"] = img.IsMain
				imgInfo["created_at"] = img.CreatedAt.Format("2006-01-02 15:04:05")

				imageURLs = append(imageURLs, imgInfo)

				if img.IsMain {
					mainImage = imgInfo
				} else {
					otherImages = append(otherImages, imgInfo)
				}
			}

			goodsData["images"] = imageURLs
			goodsData["main_image"] = mainImage
			goodsData["other_images"] = otherImages
		}

		result = append(result, goodsData)
	}

	// 返回与Django版本完全一致的格式
	response := gin.H{
		"status": "success",
		"data":   result,
		"pagination": gin.H{
			"total":     total,
			"page":      requestData.Page,
			"page_size": requestData.PageSize,
			"pages":     totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// ChangeCommodityData 修改商品信息
func (cc *CommodityController) ChangeCommodityData(c *gin.Context) {
	var requestData struct {
		CommodityID  interface{}            `json:"commodity_id" form:"commodity_id" binding:"required"`
		UpdateFields map[string]interface{} `json:"update_fields"`
	}

	// 尝试从JSON和查询参数中绑定数据
	if err := c.ShouldBind(&requestData); err != nil {
		// 检查URL查询参数中是否有commodity_id
		commodityID := c.Query("commodity_id")
		if commodityID == "" {
			// 如果URL查询参数中也没有，返回详细错误信息
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "缺少commodity_id参数，请在请求体或URL查询参数中提供",
				"details": err.Error(),
			})
			return
		}
		// 使用URL查询参数中的commodity_id
		requestData.CommodityID = commodityID
	}

	// 将CommodityID转换为字符串
	commodityIDStr := ""
	log.Printf("接收到的commodity_id参数: %v, 类型: %T", requestData.CommodityID, requestData.CommodityID)
	switch v := requestData.CommodityID.(type) {
	case string:
		commodityIDStr = v
		log.Printf("commodity_id参数是字符串类型: %s", commodityIDStr)
	case int, int64:
		// 整数类型直接转换为字符串
		commodityIDStr = fmt.Sprintf("%d", v)
		log.Printf("commodity_id参数是整数类型，转换为字符串: %s", commodityIDStr)
	case float64:
		// 浮点数类型，判断是否为整数
		if v == float64(int64(v)) {
			// 如果是整数，转换为整数格式的字符串
			commodityIDStr = fmt.Sprintf("%.0f", v)
			log.Printf("commodity_id参数是浮点整数类型，转换为整数格式字符串: %s", commodityIDStr)
		} else {
			// 非整数浮点数，保持原样
			commodityIDStr = fmt.Sprintf("%v", v)
			log.Printf("commodity_id参数是浮点数类型，转换为字符串: %s", commodityIDStr)
		}
	default:
		log.Printf("commodity_id参数格式不正确，类型: %T", v)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "commodity_id参数格式不正确",
			"details": fmt.Sprintf("参数类型: %T", v),
		})
		return
	}

	// 查询商品
	var commodity models.Commodity
	log.Printf("准备查询商品，commodity_id: %s", commodityIDStr)
	if err := db.DB.Where("commodity_id = ?", commodityIDStr).First(&commodity).Error; err != nil {
		log.Printf("查询商品失败，commodity_id: %s, 错误: %v", commodityIDStr, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "商品不存在",
			"details": fmt.Sprintf("查询ID: %s", commodityIDStr),
		})
		return
	}
	log.Printf("查询到商品: %+v", commodity)

	// 更新允许修改的字段
	updatedFields := make([]string, 0)

	for field, value := range requestData.UpdateFields {
		switch field {
		case "name":
			if strValue, ok := value.(string); ok {
				commodity.Name = strValue
				updatedFields = append(updatedFields, field)
			}
		case "category":
			if strValue, ok := value.(string); ok {
				commodity.Category = strValue
				updatedFields = append(updatedFields, field)
			}
		case "price":
			if floatValue, ok := value.(float64); ok && floatValue > 0 {
				commodity.Price = floatValue
				updatedFields = append(updatedFields, field)
			}
		case "size":
			if strValue, ok := value.(string); ok {
				commodity.Size = strValue
				updatedFields = append(updatedFields, field)
			}
		case "color":
			if strValue, ok := value.(string); ok {
				commodity.Color = strValue
				updatedFields = append(updatedFields, field)
			}
		case "notes":
			if strValue, ok := value.(string); ok {
				commodity.Notes = strValue
				updatedFields = append(updatedFields, field)
			}
		case "style_code":
			if strValue, ok := value.(string); ok {
				commodity.StyleCode = strValue
				updatedFields = append(updatedFields, field)
			}
			// 可以添加更多可更新的字段
		}
	}

	// 保存更新
	if err := db.DB.Save(&commodity).Error; err != nil {
		log.Printf("更新商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"updated_fields": updatedFields,
	})
}

// ChangeCommodityStatusOnline 商品上线
func (cc *CommodityController) ChangeCommodityStatusOnline(c *gin.Context) {
	var requestData struct {
		CommodityID interface{} `json:"commodity_id" form:"commodity_id" binding:"required"`
	}

	// 尝试从JSON和查询参数中绑定数据
	if err := c.ShouldBind(&requestData); err != nil {
		// 检查URL查询参数中是否有commodity_id
		commodityID := c.Query("commodity_id")
		if commodityID == "" {
			// 如果URL查询参数中也没有，返回详细错误信息
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "缺少commodity_id参数，请在请求体或URL查询参数中提供",
				"details": err.Error(),
			})
			return
		}
		// 使用URL查询参数中的commodity_id
		requestData.CommodityID = commodityID
	}

	// 将CommodityID转换为字符串
	commodityIDStr := ""
	log.Printf("接收到的commodity_id参数: %v, 类型: %T", requestData.CommodityID, requestData.CommodityID)
	switch v := requestData.CommodityID.(type) {
	case string:
		commodityIDStr = v
		log.Printf("commodity_id参数是字符串类型: %s", commodityIDStr)
	case int, int64:
		// 整数类型直接转换为字符串
		commodityIDStr = fmt.Sprintf("%d", v)
		log.Printf("commodity_id参数是整数类型，转换为字符串: %s", commodityIDStr)
	case float64:
		// 浮点数类型，判断是否为整数
		if v == float64(int64(v)) {
			// 如果是整数，转换为整数格式的字符串
			commodityIDStr = fmt.Sprintf("%.0f", v)
			log.Printf("commodity_id参数是浮点整数类型，转换为整数格式字符串: %s", commodityIDStr)
		} else {
			// 非整数浮点数，保持原样
			commodityIDStr = fmt.Sprintf("%v", v)
			log.Printf("commodity_id参数是浮点数类型，转换为字符串: %s", commodityIDStr)
		}
	default:
		log.Printf("commodity_id参数格式不正确，类型: %T", v)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "commodity_id参数格式不正确",
			"details": fmt.Sprintf("参数类型: %T", v),
		})
		return
	}

	// 查询商品状态
	var commoditySituation models.CommoditySituation
	log.Printf("准备查询商品，commodity_id: %s", commodityIDStr)
	if err := db.DB.Where("commodity_id = ?", commodityIDStr).First(&commoditySituation).Error; err != nil {
		log.Printf("查询商品失败，commodity_id: %s, 错误: %v", commodityIDStr, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "商品不存在",
			"details": fmt.Sprintf("查询ID: %s", commodityIDStr),
		})
		return
	}
	log.Printf("查询到商品: %+v", commoditySituation)

	// 更新状态
	commoditySituation.Status = "online"
	commoditySituation.OnlineTime = time.Now()

	if err := db.DB.Save(&commoditySituation).Error; err != nil {
		log.Printf("更新商品状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	// 格式化上线时间为中国时区
	formattedTime := commoditySituation.OnlineTime.Add(8 * time.Hour).Format("2006-01-02 15:04:05")

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"online_time": formattedTime,
	})
}

// ChangeCommodityStatusOffline 商品下线
func (cc *CommodityController) ChangeCommodityStatusOffline(c *gin.Context) {
	var requestData struct {
		CommodityID interface{} `json:"commodity_id" form:"commodity_id" binding:"required"`
	}

	// 尝试从JSON和查询参数中绑定数据
	if err := c.ShouldBind(&requestData); err != nil {
		// 检查URL查询参数中是否有commodity_id
		commodityID := c.Query("commodity_id")
		if commodityID == "" {
			// 如果URL查询参数中也没有，返回详细错误信息
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "缺少commodity_id参数，请在请求体或URL查询参数中提供",
				"details": err.Error(),
			})
			return
		}
		// 使用URL查询参数中的commodity_id
		requestData.CommodityID = commodityID
	}

	// 将CommodityID转换为字符串
	commodityIDStr := ""
	log.Printf("接收到的commodity_id参数: %v, 类型: %T", requestData.CommodityID, requestData.CommodityID)
	switch v := requestData.CommodityID.(type) {
	case string:
		commodityIDStr = v
		log.Printf("commodity_id参数是字符串类型: %s", commodityIDStr)
	case int, int64:
		// 整数类型直接转换为字符串
		commodityIDStr = fmt.Sprintf("%d", v)
		log.Printf("commodity_id参数是整数类型，转换为字符串: %s", commodityIDStr)
	case float64:
		// 浮点数类型，判断是否为整数
		if v == float64(int64(v)) {
			// 如果是整数，转换为整数格式的字符串
			commodityIDStr = fmt.Sprintf("%.0f", v)
			log.Printf("commodity_id参数是浮点整数类型，转换为整数格式字符串: %s", commodityIDStr)
		} else {
			// 非整数浮点数，保持原样
			commodityIDStr = fmt.Sprintf("%v", v)
			log.Printf("commodity_id参数是浮点数类型，转换为字符串: %s", commodityIDStr)
		}
	default:
		log.Printf("commodity_id参数格式不正确，类型: %T", v)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "commodity_id参数格式不正确",
			"details": fmt.Sprintf("参数类型: %T", v),
		})
		return
	}

	// 查询商品状态
	var commoditySituation models.CommoditySituation
	log.Printf("准备查询商品，commodity_id: %s", commodityIDStr)
	if err := db.DB.Where("commodity_id = ?", commodityIDStr).First(&commoditySituation).Error; err != nil {
		log.Printf("查询商品失败，commodity_id: %s, 错误: %v", commodityIDStr, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "商品不存在",
			"details": fmt.Sprintf("查询ID: %s", commodityIDStr),
		})
		return
	}
	log.Printf("查询到商品: %+v", commoditySituation)

	// 更新状态
	commoditySituation.Status = "offline"
	commoditySituation.OfflineTime = time.Now()

	if err := db.DB.Save(&commoditySituation).Error; err != nil {
		log.Printf("更新商品状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	// 格式化下线时间为中国时区
	formattedTime := commoditySituation.OfflineTime.Add(8 * time.Hour).Format("2006-01-02 15:04:05")

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"offline_time": formattedTime,
	})
}

// GetCommodityStatus 获取商品状态
func (cc *CommodityController) GetCommodityStatus(c *gin.Context) {
	var requestData struct {
		CommodityID interface{} `json:"commodity_id" form:"commodity_id" binding:"required"`
	}

	// 尝试从JSON和查询参数中绑定数据
	if err := c.ShouldBind(&requestData); err != nil {
		// 检查URL查询参数中是否有commodity_id
		commodityID := c.Query("commodity_id")
		if commodityID == "" {
			// 如果URL查询参数中也没有，返回详细错误信息
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "缺少commodity_id参数，请在请求体或URL查询参数中提供",
				"details": err.Error(),
			})
			return
		}
		// 使用URL查询参数中的commodity_id
		requestData.CommodityID = commodityID
	}

	// 将CommodityID转换为字符串
	commodityIDStr := ""
	log.Printf("接收到的commodity_id参数: %v, 类型: %T", requestData.CommodityID, requestData.CommodityID)
	switch v := requestData.CommodityID.(type) {
	case string:
		commodityIDStr = v
		log.Printf("commodity_id参数是字符串类型: %s", commodityIDStr)
	case int, int64:
		// 整数类型直接转换为字符串
		commodityIDStr = fmt.Sprintf("%d", v)
		log.Printf("commodity_id参数是整数类型，转换为字符串: %s", commodityIDStr)
	case float64:
		// 浮点数类型，判断是否为整数
		if v == float64(int64(v)) {
			// 如果是整数，转换为整数格式的字符串
			commodityIDStr = fmt.Sprintf("%.0f", v)
			log.Printf("commodity_id参数是浮点整数类型，转换为整数格式字符串: %s", commodityIDStr)
		} else {
			// 非整数浮点数，保持原样
			commodityIDStr = fmt.Sprintf("%v", v)
			log.Printf("commodity_id参数是浮点数类型，转换为字符串: %s", commodityIDStr)
		}
	default:
		log.Printf("commodity_id参数格式不正确，类型: %T", v)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "commodity_id参数格式不正确",
			"details": fmt.Sprintf("参数类型: %T", v),
		})
		return
	}

	// 查询商品状态
	var commoditySituation models.CommoditySituation
	log.Printf("准备查询商品，commodity_id: %s", commodityIDStr)
	if err := db.DB.Where("commodity_id = ?", commodityIDStr).First(&commoditySituation).Error; err != nil {
		log.Printf("查询商品失败，commodity_id: %s, 错误: %v", commodityIDStr, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "商品不存在",
			"details": fmt.Sprintf("查询ID: %s", commodityIDStr),
		})
		return
	}
	log.Printf("查询到商品: %+v", commoditySituation)

	// 构建响应数据
	responseData := gin.H{
		"status": commoditySituation.Status,
	}

	// 根据状态返回对应时间
	if commoditySituation.Status == "online" && !commoditySituation.OnlineTime.IsZero() {
		responseData["online_time"] = commoditySituation.OnlineTime.Add(8 * time.Hour).Format("2006-01-02 15:04:05")
		responseData["offline_time"] = ""
	} else if commoditySituation.Status == "offline" && !commoditySituation.OfflineTime.IsZero() {
		responseData["online_time"] = ""
		responseData["offline_time"] = commoditySituation.OfflineTime.Add(8 * time.Hour).Format("2006-01-02 15:04:05")
	} else {
		responseData["online_time"] = ""
		responseData["offline_time"] = ""
	}

	c.JSON(http.StatusOK, responseData)
}

// CommodityDetail 获取商品详情
func (cc *CommodityController) CommodityDetail(c *gin.Context) {
	commodityIDStr := c.Param("id")
	commodityID, err := strconv.Atoi(commodityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}

	// 查询商品
	var commodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", commodityID).First(&commodity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品不存在"})
		return
	}

	// 查询商品图片
	var commodityImages []models.CommodityImage
	if err := db.DB.Where("commodity_id = ?", commodityID).Find(&commodityImages).Error; err != nil {
		log.Printf("获取商品图片失败: %v", err)
	}

	// 查询商品状态
	var commoditySituation models.CommoditySituation
	if err := db.DB.Where("commodity_id = ?", strconv.Itoa(commodityID)).First(&commoditySituation).Error; err != nil {
		// 如果没有状态记录，创建一个默认的
		commoditySituation = models.CommoditySituation{
			CommodityID: strconv.Itoa(commodityID),
			Status:      "online",
			SalesVolume: 0,
			StyleCode:   commodity.StyleCode,
		}
		if err := db.DB.Create(&commoditySituation).Error; err != nil {
			log.Printf("创建商品状态记录失败: %v", err)
		}
	}

	// 准备响应数据
	detailMap := convertCommodityToMap(commodity, c)
	detailMap["images"] = convertImagesToMap(commodityImages, c)
	detailMap["status"] = commoditySituation.Status
	detailMap["sales_volume"] = commoditySituation.SalesVolume

	responseData := gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    detailMap,
	}

	c.JSON(http.StatusOK, responseData)
}

// CommodityCreate 创建商品
func (cc *CommodityController) CommodityCreate(c *gin.Context) {
	var requestData models.Commodity
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	// 验证必要字段
	if requestData.Name == "" || requestData.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少必要的商品信息"})
		return
	}

	// 创建商品
	commodity := models.Commodity{
		Name:      requestData.Name,
		StyleCode: requestData.StyleCode,
		Category:  requestData.Category,
		Price:     requestData.Price,
		Size:      requestData.Size,
		Color:     requestData.Color,
		Image:     "default.png", // 默认图片路径
	}

	if err := db.DB.Create(&commodity).Error; err != nil {
		log.Printf("创建商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 创建商品状态
	commoditySituation := models.CommoditySituation{
		CommodityID: commodity.CommodityID,
		Status:      "online", // 设置为在线状态
	}

	if err := db.DB.Create(&commoditySituation).Error; err != nil {
		log.Printf("创建商品状态失败: %v", err)
	}

	// 如果有款式代码，处理款式相关数据
	if requestData.StyleCode != "" {
		// 查找或创建款式数据
		var styleCodeData models.StyleCodeData
		if err := db.DB.Where("style_code = ?", requestData.StyleCode).First(&styleCodeData).Error; err != nil {
			styleCodeData = models.StyleCodeData{
				StyleCode:       requestData.StyleCode,
				Name:            "",
				Category:        "",
				Price:           0,
				Image:           "",
				DisplayPictures: "{}", // 初始化为空JSON对象，符合MySQL JSON类型要求
			}
			if err := db.DB.Create(&styleCodeData).Error; err != nil {
				log.Printf("创建款式数据失败: %v", err)
			}
		}

		// 查找或创建款式状态
		var styleCodeSituation models.StyleCodeSituation
		if err := db.DB.Where("style_code = ?", requestData.StyleCode).First(&styleCodeSituation).Error; err != nil {
			styleCodeSituation = models.StyleCodeSituation{
				StyleCode: requestData.StyleCode,
				Status:    "online",
			}
			if err := db.DB.Create(&styleCodeSituation).Error; err != nil {
				log.Printf("创建款式状态失败: %v", err)
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "商品创建成功",
		"data":    gin.H{"commodity_id": commodity.CommodityID},
	})
}

// CommodityUpdate 更新商品
func (cc *CommodityController) CommodityUpdate(c *gin.Context) {
	commodityIDStr := c.Param("id")
	commodityID, err := strconv.Atoi(commodityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}

	// 查询商品
	var commodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", commodityID).First(&commodity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品不存在"})
		return
	}

	// 绑定请求数据
	var updateData models.Commodity
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON格式"})
		return
	}

	// 更新字段
	if updateData.Name != "" {
		commodity.Name = updateData.Name
	}
	if updateData.StyleCode != "" {
		commodity.StyleCode = updateData.StyleCode
	}
	if updateData.Category != "" {
		commodity.Category = updateData.Category
	}
	if updateData.Price > 0 {
		commodity.Price = updateData.Price
	}
	if updateData.Size != "" {
		commodity.Size = updateData.Size
	}
	if updateData.Color != "" {
		commodity.Color = updateData.Color
	}
	if updateData.CategoryDetail != "" {
		commodity.CategoryDetail = updateData.CategoryDetail
	}
	if updateData.Height != "" {
		commodity.Height = updateData.Height
	}
	if updateData.SpecCode != "" {
		commodity.SpecCode = updateData.SpecCode
	}
	if updateData.Notes != "" {
		commodity.Notes = updateData.Notes
	}

	// 保存更新
	if err := db.DB.Save(&commodity).Error; err != nil {
		log.Printf("更新商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "商品更新成功",
	})
}

// CommodityDelete 删除商品
func (cc *CommodityController) CommodityDelete(c *gin.Context) {
	commodityIDStr := c.Param("id")
	commodityID, err := strconv.Atoi(commodityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品ID"})
		return
	}

	// 查询商品
	var commodity models.Commodity
	if err := db.DB.Where("commodity_id = ?", strconv.Itoa(commodityID)).First(&commodity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品不存在"})
		return
	}

	// 删除商品（软删除）
	if err := db.DB.Delete(&commodity).Error; err != nil {
		log.Printf("删除商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "商品删除成功",
	})
}

// 工具函数：将商品对象转换为map
func convertCommodityToMap(commodity models.Commodity, c *gin.Context) map[string]interface{} {
	result := make(map[string]interface{})
	result["commodity_id"] = commodity.CommodityID
	result["name"] = commodity.Name
	result["style_code"] = commodity.StyleCode
	result["category"] = commodity.Category
	result["category_detail"] = commodity.CategoryDetail
	result["price"] = commodity.Price
	result["size"] = commodity.Size
	result["color"] = commodity.Color
	result["height"] = commodity.Height
	result["spec_code"] = commodity.SpecCode
	result["notes"] = commodity.Notes
	result["created_at"] = commodity.CreatedAt.Format("2006-01-02 15:04:05")

	// 处理图片URL
	if commodity.Image != "" {
		// 获取请求的协议，考虑反向代理环境
		proto := utils.GetRequestProto(c)
		baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)
		result["image"] = utils.BuildFullImageURL(baseURL, commodity.Image, "media")
	}

	return result
}

// 工具函数：将商品列表转换为map数组
func convertCommoditiesToMap(commodities []models.Commodity, c *gin.Context) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(commodities))
	for _, commodity := range commodities {
		result = append(result, convertCommodityToMap(commodity, c))
	}
	return result
}

// 工具函数：将图片列表转换为map数组
func convertImagesToMap(images []models.CommodityImage, c *gin.Context) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(images))
	// 获取请求的协议，考虑反向代理环境
	proto := utils.GetRequestProto(c)
	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)

	for _, image := range images {
		imgMap := make(map[string]interface{})
		imgMap["id"] = image.ID
		// 使用BuildFullImageURL函数构建完整URL
		imgMap["url"] = utils.BuildFullImageURL(baseURL, image.Image, "media")
		imgMap["is_main"] = image.IsMain
		result = append(result, imgMap)
	}

	return result
}

// SearchProductsByName 根据名称搜索商品
func (cc *CommodityController) SearchProductsByName(c *gin.Context) {
	var requestData struct {
		SearchStr string `json:"search_str" binding:"required"`
		Page      int    `json:"page" binding:"required,min=1"`
		PageSize  int    `json:"page_size" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Printf("绑定请求数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据",
			"details": err.Error(),
		})
		return
	}

	// 限制每页最大数量
	if requestData.PageSize > 100 {
		requestData.PageSize = 100
	}

	// 计算偏移量
	offset := (requestData.Page - 1) * requestData.PageSize

	// 构建查询
	var commodities []models.Commodity
	query := db.DB.Model(&models.Commodity{}).
		Where("name LIKE ?", "%"+requestData.SearchStr+"%")

	// 获取总数
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		log.Printf("获取商品总数失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	// 执行分页查询
	if err := query.Offset(offset).Limit(requestData.PageSize).Find(&commodities).Error; err != nil {
		log.Printf("搜索商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	// 转换为map数组
	result := convertCommoditiesToMap(commodities, c)

	// 计算总页数
	totalPages := (totalCount + int64(requestData.PageSize) - 1) / int64(requestData.PageSize)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"items":     result,
			"total":     totalCount,
			"page":      requestData.Page,
			"page_size": requestData.PageSize,
			"pages":     totalPages,
		},
	})
}

// BatchGetProductsByIDs 批量获取商品信息
func (cc *CommodityController) BatchGetProductsByIDs(c *gin.Context) {
	var requestData struct {
		CommodityIDs []string `json:"commodity_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Printf("绑定请求数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求数据无效",
			"details": err.Error(),
		})
		return
	}

	log.Printf("接收到的commodity_ids参数: %v", requestData.CommodityIDs)

	var commodities []models.Commodity
	if err := db.DB.Where("commodity_id IN (?)", requestData.CommodityIDs).Find(&commodities).Error; err != nil {
		log.Printf("批量获取商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	// 转换为map数组
	result := convertCommoditiesToMap(commodities, c)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    result,
		"count":   len(commodities),
	})
}

// ChangeStyleCodeStatusOnline 设置款式代码为在线状态
func (cc *CommodityController) ChangeStyleCodeStatusOnline(c *gin.Context) {
	var requestData struct {
		StyleCode string `json:"style_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Printf("绑定请求数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "款式代码不能为空",
			"details": err.Error(),
		})
		return
	}

	// 获取当前时间
	currentTime := time.Now()

	// 查找或创建款式状态
	var styleCodeSituation models.StyleCodeSituation
	if err := db.DB.Where("style_code = ?", requestData.StyleCode).First(&styleCodeSituation).Error; err != nil {
		styleCodeSituation = models.StyleCodeSituation{
			StyleCode:  requestData.StyleCode,
			Status:     "online",
			OnlineTime: currentTime,
		}
		if err := db.DB.Create(&styleCodeSituation).Error; err != nil {
			log.Printf("创建款式状态失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "服务器内部错误",
			})
			return
		}
	} else {
		// 更新状态为在线
		styleCodeSituation.Status = "online"
		styleCodeSituation.OnlineTime = currentTime
		if err := db.DB.Save(&styleCodeSituation).Error; err != nil {
			log.Printf("更新款式状态失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "服务器内部错误",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "款式状态设置为在线成功",
	})
}

// ChangeStyleCodeStatusOffline 设置款式代码为离线状态
func (cc *CommodityController) ChangeStyleCodeStatusOffline(c *gin.Context) {
	var requestData struct {
		StyleCode string `json:"style_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Printf("绑定请求数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "款式代码不能为空",
			"details": err.Error(),
		})
		return
	}

	// 获取当前时间
	currentTime := time.Now()

	// 查找或创建款式状态
	var styleCodeSituation models.StyleCodeSituation
	if err := db.DB.Where("style_code = ?", requestData.StyleCode).First(&styleCodeSituation).Error; err != nil {
		styleCodeSituation = models.StyleCodeSituation{
			StyleCode:   requestData.StyleCode,
			Status:      "offline",
			OfflineTime: currentTime,
		}
		if err := db.DB.Create(&styleCodeSituation).Error; err != nil {
			log.Printf("创建款式状态失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "服务器内部错误",
			})
			return
		}
	} else {
		// 更新状态为离线
		styleCodeSituation.Status = "offline"
		styleCodeSituation.OfflineTime = currentTime
		if err := db.DB.Save(&styleCodeSituation).Error; err != nil {
			log.Printf("更新款式状态失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "服务器内部错误",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "款式状态设置为离线成功",
	})
}

// GetCommoditiesByStyleCode 根据款式代码获取商品列表
func (cc *CommodityController) GetCommoditiesByStyleCode(c *gin.Context) {
	var requestData struct {
		Shopname  string `json:"shopname" binding:"required"`
		StyleCode string `json:"style_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Printf("绑定请求数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的JSON格式",
		})
		return
	}

	// 验证店铺名称
	if requestData.Shopname != "youlan_kids" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的店铺名称",
		})
		return
	}

	// 查询该style_code下的所有商品
	var commodities []models.Commodity
	if err := db.DB.Where("style_code = ?", requestData.StyleCode).Find(&commodities).Error; err != nil {
		log.Printf("根据款式代码获取商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	// 查询商品状态，只返回上线商品
	var onlineCommodityIDs []string
	if err := db.DB.Model(&models.CommoditySituation{}).
		Where("status = ?", "online").
		Where("commodity_id IN ?", extractCommodityIDs(commodities)).
		Pluck("commodity_id", &onlineCommodityIDs).Error; err != nil {
		log.Printf("获取上线商品ID失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	// 过滤出上线的商品
	var onlineCommodities []models.Commodity
	if err := db.DB.Where("commodity_id IN ?", onlineCommodityIDs).Find(&onlineCommodities).Error; err != nil {
		log.Printf("获取上线商品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "服务器处理失败",
		})
		return
	}

	// 获取请求的协议，考虑反向代理环境
	proto := utils.GetRequestProto(c)
	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)

	// 构建响应数据
	result := map[string]interface{}{
		"name":             "",
		"price":            0.0,
		"items":            []map[string]interface{}{},
		"images":           []map[string]interface{}{},
		"main_image":       nil,
		"other_images":     []map[string]interface{}{},
		"display_pictures": make(map[string]string), // 初始化为空map
	}

	// 查询StyleCodeData表，获取display_pictures
	var styleCodeData models.StyleCodeData
	if err := db.DB.Where("style_code = ?", requestData.StyleCode).First(&styleCodeData).Error; err == nil {
		// 如果找到了对应的款式数据，并且display_pictures不为空
		if styleCodeData.DisplayPictures != "" && styleCodeData.DisplayPictures != "{}" {
			var displayPicturesMap map[string]string
			if err := json.Unmarshal([]byte(styleCodeData.DisplayPictures), &displayPicturesMap); err == nil {
				// 处理display_pictures中的图片URL，确保它们是完整的URL
				processedPictures := make(map[string]string)
				for key, imagePath := range displayPicturesMap {
					// 处理JSON中已转义的反斜杠，先替换双反斜杠为单反斜杠，再替换为正斜杠
					cleanPath := strings.ReplaceAll(imagePath, "\\\\", "\\")
					cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")
					// 使用BuildFullImageURL函数处理图片路径
					processedPictures[key] = utils.BuildFullImageURL(baseURL, cleanPath, "media")
				}
				result["display_pictures"] = processedPictures
			} else {
				log.Printf("解析display_pictures失败: %v", err)
			}
		}
	}

	// 创建颜色分组的字典
	colorGroups := make(map[string]map[string]interface{})

	for i, commodity := range onlineCommodities {
		// 设置name和price（使用第一款商品的名称和价格）
		if i == 0 {
			result["name"] = commodity.Name
			result["price"] = commodity.Price

			// 获取该商品的所有图片信息
			var commodityImages []models.CommodityImage
			if err := db.DB.Where("commodity_id = ?", commodity.CommodityID).Find(&commodityImages).Error; err != nil {
				log.Printf("获取商品图片失败: %v", err)
			}

			images := []map[string]interface{}{}
			for _, img := range commodityImages {
				imageInfo := map[string]interface{}{
					"id":         img.ID,
					"url":        utils.BuildFullImageURL(baseURL, img.Image, "media"),
					"is_main":    img.IsMain,
					"created_at": img.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05"),
				}
				images = append(images, imageInfo)
			}

			// 如果没有从CommodityImage获取到图片，使用商品本身的图片作为备选
			if len(images) == 0 {
				// 优先使用image字段
				if commodity.Image != "" {
					mainImageURL := utils.BuildFullImageURL(baseURL, commodity.Image, "media")
					images = append(images, map[string]interface{}{
						"id":         nil,
						"url":        mainImageURL,
						"is_main":    true,
						"created_at": commodity.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05"),
					})
					// 再检查promo_image字段
				} else if commodity.PromoImage != "" {
					promoImageURL := utils.BuildFullImageURL(baseURL, commodity.PromoImage, "media")
					images = append(images, map[string]interface{}{
						"id":         nil,
						"url":        promoImageURL,
						"is_main":    true,
						"created_at": commodity.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05"),
					})
				}
			}

			result["images"] = images

			// 添加主图和其他图片分类
			var mainImage map[string]interface{}
			var otherImages []map[string]interface{}
			for _, img := range images {
				if isMain, ok := img["is_main"].(bool); ok && isMain {
					mainImage = img
				} else {
					otherImages = append(otherImages, img)
				}
			}
			result["main_image"] = mainImage
			result["other_images"] = otherImages
		}

		// 获取color_image，如果为空则使用image
		var colorImage string
		if commodity.ColorImage != "" {
			colorImage = utils.BuildFullImageURL(baseURL, commodity.ColorImage, "media")
		} else if commodity.Image != "" {
			colorImage = utils.BuildFullImageURL(baseURL, commodity.Image, "media")
		} else {
			colorImage = ""
		}

		// 按颜色分组
		color := commodity.Color
		if _, exists := colorGroups[color]; !exists {
			// 新颜色，创建颜色组
			colorGroups[color] = map[string]interface{}{
				"color":       color,
				"color_image": colorImage,
				"sizes":       []map[string]interface{}{},
			}
		}

		// 添加尺码信息到颜色组
		colorGroups[color]["sizes"] = append(colorGroups[color]["sizes"].([]map[string]interface{}), map[string]interface{}{
			"commodity_id": commodity.CommodityID,
			"size":         commodity.Size,
		})
	}

	// 将颜色分组字典转换为列表格式
	items := make([]map[string]interface{}, 0, len(colorGroups))
	for _, colorInfo := range colorGroups {
		items = append(items, colorInfo)
	}
	result["items"] = items

	// 返回与Django版本完全一致的格式
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// 工具函数：从商品列表中提取商品ID
func extractCommodityIDs(commodities []models.Commodity) []string {
	ids := make([]string, 0, len(commodities))
	for _, commodity := range commodities {
		ids = append(ids, commodity.CommodityID)
	}
	return ids
}

// GetAllCategories 获取所有商品类别
func (cc *CommodityController) GetAllCategories(c *gin.Context) {
	var requestData struct {
		Shopname string `json:"shopname" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据",
		})
		return
	}

	// 验证店铺名称
	if requestData.Shopname != "youlan_kids" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的店铺名称",
		})
		return
	}

	// 查询所有状态为online的商品ID
	var onlineCommodityIDs []string
	if err := db.DB.Model(&models.CommoditySituation{}).
		Where("status = ?", "online").
		Pluck("commodity_id", &onlineCommodityIDs).Error; err != nil {
		log.Printf("查询在线商品ID失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	// 查询所有不重复的商品类别，只包含在线商品
	var categories []string
	if err := db.DB.Model(&models.Commodity{}).
		Where("commodity_id IN (?)", onlineCommodityIDs).
		Distinct("category").
		Order("category").
		Pluck("category", &categories).Error; err != nil {
		log.Printf("查询商品类别失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    categories,
	})
}

// handleDisplayPicture 处理单个展示图片的上传和URL构建
func handleDisplayPicture(c *gin.Context, file *multipart.FileHeader, styleCode string, baseURL string, position string, displayPictures map[string]string) {
	// 为每个款式编码创建独立的子文件夹
	displayDir := filepath.Join("styles", styleCode, "display")
	log.Printf("准备保存到目录: %s", displayDir)
	// 保存上传的文件
	imagePath, err := utils.SaveUploadedFile(c, file, displayDir, "style_display_")
	if err != nil {
		log.Printf("SaveUploadedFile失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("上传展示图片(位置%s)失败: %v", position, err),
		})
		return
	}
	log.Printf("SaveUploadedFile成功, 返回路径: %s", imagePath)
	// 实际保存文件到指定路径
	fullPath := filepath.Join("./media", imagePath)
	log.Printf("准备保存文件到完整路径: %s", fullPath)
	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		log.Printf("c.SaveUploadedFile失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("保存展示图片(位置%s)失败: %v", position, err),
		})
		return
	}
	log.Printf("文件保存成功: %s", fullPath)
	// 只保存相对路径到数据库
	log.Printf("保存的相对路径: %s", imagePath)
	// 使用指定位置作为key
	displayPictures[position] = imagePath
	log.Printf("添加到displayPictures映射, key: %s, value: %s", position, imagePath)
}

// UpdateStyleCodeInfo 修改款式信息(支持multipart/form-data格式)
func (cc *CommodityController) UpdateStyleCodeInfo(c *gin.Context) {
	var requestData struct {
		StyleCode      string  `form:"style_code" binding:"required"` // 款式编码(必填)
		Name           string  `form:"name"`                          // 款式名称(选填)
		Category       string  `form:"category"`                      // 分类(选填)
		CategoryDetail string  `form:"category_detail"`               // 详细分类(选填)
		Price          float64 `form:"price"`                         // 价格(选填)
	}

	// 解析请求体(支持multipart/form-data)
	if err := c.ShouldBind(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 获取协议
	proto := utils.GetRequestProto(c)
	// 构建基础URL
	baseURL := fmt.Sprintf("%s://%s", proto, c.Request.Host)

	// 处理主图上传
	var mainImageURL string
	mainImageFile, err := c.FormFile("image")
	if err == nil && mainImageFile != nil {
		// 为每个款式编码创建独立的子文件夹
		styleDir := filepath.Join("styles", requestData.StyleCode)
		// 保存上传的文件
		imagePath, err := utils.SaveUploadedFile(c, mainImageFile, styleDir, "style_main_")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "上传主图失败: " + err.Error(),
			})
			return
		}
		// 实际保存文件到指定路径
		fullPath := filepath.Join("./media", imagePath)
		if err := c.SaveUploadedFile(mainImageFile, fullPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "保存主图失败: " + err.Error(),
			})
			return
		}
		// 只保存相对路径到数据库
		mainImageURL = imagePath
	}

	// 处理展示图片上传
	displayPictures := make(map[string]string)
	// 先从数据库中查询款式数据，获取现有展示图片
	var tempStyleData models.StyleCodeData
	err = db.DB.Where("style_code = ?", requestData.StyleCode).First(&tempStyleData).Error
	if err == nil {
		// 解析现有的display_pictures JSON字符串到map中
		if tempStyleData.DisplayPictures != "" && tempStyleData.DisplayPictures != "{}" {
			if err := json.Unmarshal([]byte(tempStyleData.DisplayPictures), &displayPictures); err != nil {
				log.Printf("解析现有展示图片数据失败: %v", err)
				displayPictures = make(map[string]string)
			}
		}
	}
	// 获取form中的display_pictures_json参数(如果有)
	displayPicturesJSON := c.PostForm("display_pictures_json")
	if displayPicturesJSON != "" {
		if err := json.Unmarshal([]byte(displayPicturesJSON), &displayPictures); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "无效的display_pictures_json格式: " + err.Error(),
			})
			return
		}
	}

	// 处理上传的多张展示图片
	form, err := c.MultipartForm()
	log.Printf("尝试获取MultipartForm, err: %v", err)
	if err != nil {
		log.Printf("获取MultipartForm失败: %v", err)
	} else if form != nil {
		log.Printf("成功获取MultipartForm, 包含文件字段数: %d", len(form.File))
		// 记录所有文件字段名和内容
		for fieldName := range form.File {
			log.Printf("Form中包含文件字段: %s, 文件数量: %d", fieldName, len(form.File[fieldName]))
			for i, file := range form.File[fieldName] {
				log.Printf("  - 文件%d: %s, 大小: %d字节, 头信息: %v", i+1, file.Filename, file.Size, file.Header)
			}
		}

		// 支持标准上传方式：
		// 1. 使用display_pictures[]批量上传（按顺序分配位置）
		if files, ok := form.File["display_pictures[]"]; ok {
			log.Printf("找到display_pictures[]字段, 文件数量: %d", len(files))
			for i, file := range files {
				position := i + 1
				log.Printf("处理第%d个文件: %s, 大小: %d字节, 自动分配位置: %d", i+1, file.Filename, file.Size, position)
				handleDisplayPicture(c, file, requestData.StyleCode, baseURL, fmt.Sprintf("%d", position), displayPictures)
			}
		}

		// 2. 处理display_pictures[N]单独指定位置上传
		for fieldName, files := range form.File {
			if len(files) > 0 {
				// 检查字段名是否符合display_pictures[N]格式或display_pictures[1]格式
				if strings.HasPrefix(fieldName, "display_pictures[") && strings.HasSuffix(fieldName, "]") {
					// 提取位置编号
					positionStr := fieldName[17 : len(fieldName)-1] // 提取display_pictures[和]之间的部分
					position, err := strconv.Atoi(positionStr)
					if err == nil && position > 0 {
						// 每个位置字段只处理一个文件
						file := files[0]
						log.Printf("处理指定位置的文件: %s, 大小: %d字节, 指定位置: %d", file.Filename, file.Size, position)
						handleDisplayPicture(c, file, requestData.StyleCode, baseURL, positionStr, displayPictures)
					}
				}
			}
		}
	} else {
		log.Printf("form为nil，无法获取文件。尝试直接获取单个文件...")
		// 尝试直接获取单个文件
		file, err := c.FormFile("display_pictures[1]")
		if err == nil {
			log.Printf("成功直接获取display_pictures[1]: %s, 大小: %d字节", file.Filename, file.Size)
			handleDisplayPicture(c, file, requestData.StyleCode, baseURL, "1", displayPictures)
		} else {
			log.Printf("直接获取文件失败: %v", err)
		}
	}

	log.Printf("处理完图片后，displayPictures的长度: %d", len(displayPictures))

	// 声明款式数据变量
	var styleCodeData models.StyleCodeData
	// 开始事务
	begin := db.DB.Begin()
	if begin.Error != nil {
		log.Printf("开启事务失败: %v", begin.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	// 查询款式数据
	err = begin.Where("style_code = ?", requestData.StyleCode).First(&styleCodeData).Error
	if err != nil {
		begin.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "款式编码不存在",
		})
		return
	}

	// 更新款式数据
	if requestData.Name != "" {
		styleCodeData.Name = requestData.Name
	}
	if requestData.Category != "" {
		styleCodeData.Category = requestData.Category
	}
	if requestData.CategoryDetail != "" {
		styleCodeData.CategoryDetail = requestData.CategoryDetail
	}
	if requestData.Price > 0 {
		styleCodeData.Price = requestData.Price
	}
	// 如果上传了主图，更新图片路径（使用相对路径）
	if mainImageURL != "" {
		styleCodeData.Image = mainImageURL
	}
	// 处理展示图片
	// 将map转换为JSON字符串

	displayPicturesJSON, err = utils.StringMapToJSONString(displayPictures)
	if err != nil {
		begin.Rollback()
		log.Printf("转换展示图片数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新款式信息失败",
		})
		return
	}
	styleCodeData.DisplayPictures = displayPicturesJSON

	// 保存更新
	if err := begin.Save(&styleCodeData).Error; err != nil {
		begin.Rollback()
		log.Printf("更新款式数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新款式信息失败",
		})
		return
	}

	// 提交事务
	if err := begin.Commit().Error; err != nil {
		begin.Rollback()
		log.Printf("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新款式信息失败",
		})
		return
	}

	// 准备响应数据，确保返回完整URL
	displayPicturesMap := make(map[string]string)
	if styleCodeData.DisplayPictures != "" && styleCodeData.DisplayPictures != "{}" {
		if err := json.Unmarshal([]byte(styleCodeData.DisplayPictures), &displayPicturesMap); err == nil {
			// 处理display_pictures中的图片URL，确保它们是完整的URL
			processedPictures := make(map[string]string)
			for key, imagePath := range displayPicturesMap {
				// 处理JSON中已转义的反斜杠，先替换双反斜杠为单反斜杠，再替换为正斜杠
				cleanPath := strings.ReplaceAll(imagePath, "\\\\", "\\")
				cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")
				// 使用BuildFullImageURL函数处理图片路径
				processedPictures[key] = utils.BuildFullImageURL(baseURL, cleanPath, "media")
			}
			// 将处理后的display_pictures转换回JSON字符串
			processedPicturesJSON, err := utils.StringMapToJSONString(processedPictures)
			if err == nil {
				styleCodeData.DisplayPictures = processedPicturesJSON
			}
		}
	}

	// 确保main image也是完整URL
	if styleCodeData.Image != "" && !strings.HasPrefix(styleCodeData.Image, "http://") && !strings.HasPrefix(styleCodeData.Image, "https://") {
		styleCodeData.Image = utils.BuildFullImageURL(baseURL, styleCodeData.Image, "media")
	}

	// 准备响应数据
	responseData := gin.H{
		"style_code":       styleCodeData.StyleCode,
		"name":             styleCodeData.Name,
		"category":         styleCodeData.Category,
		"category_detail":  styleCodeData.CategoryDetail,
		"price":            styleCodeData.Price,
		"image":            styleCodeData.Image,
		"display_pictures": styleCodeData.DisplayPictures,
		"updated_at":       styleCodeData.UpdatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    responseData,
	})
}

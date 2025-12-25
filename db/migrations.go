package db

import (
	"django_to_go/models"
	"fmt"
	"log"
	"strings"
)

// RunMigrations 运行数据库迁移
// 此函数用于同步所有模型的数据库结构
func RunMigrations() {
	log.Println("开始运行数据库迁移...")

	// 移除SnowUser表中mobile字段的唯一索引
	log.Println("开始移除SnowUser表中mobile字段的唯一索引...")
	// 直接尝试删除索引，如果索引不存在会报错，但我们可以忽略这个错误
	if err := DB.Exec("ALTER TABLE snow_uesr DROP INDEX idx_snow_uesr_mobile").Error; err != nil {
		// 检查错误是否是索引不存在的错误
		if !strings.Contains(err.Error(), "doesn't exist") && !strings.Contains(err.Error(), "Duplicate column name") {
			log.Printf("移除mobile字段唯一索引失败: %v", err)
		} else {
			log.Println("mobile字段唯一索引不存在，跳过删除操作")
		}
	} else {
		log.Println("mobile字段唯一索引移除成功")
	}

	// 同步StyleCodeData模型（重点关注）
	log.Println("开始同步StyleCodeData模型...")
	if err := DB.AutoMigrate(&models.StyleCodeData{}); err != nil {
		log.Printf("同步StyleCodeData模型结构失败: %v", err)
	} else {
		log.Println("StyleCodeData模型结构同步成功，包括DisplayPictures字段")
	}

	// 同步所有其他模型
	modelsToMigrate := []interface{}{
		&models.User{},
		&models.UserData{},
		&models.Address{},
		&models.Commodity{},
		&models.CommodityImage{},
		&models.CommoditySituation{},
		&models.StyleCodeSituation{},
		&models.Order{},
		&models.ReturnOrder{},
		&models.Cart{},
		&models.ActivityImage{},
		&models.Product{},
		&models.AccessToken{},
		&models.DjangoCustomerServiceUser{},
		&models.DjangoOperationUser{},
		&models.SnowUser{},
		&models.SnowSuccessUser{},
		&models.SnowLotteryDraw{},
		&models.SnowOrderData{},
		&models.SnowLotteryDraw{},
	}

	// 循环同步每个模型
	for _, model := range modelsToMigrate {
		modelName := fmt.Sprintf("%T", model)
		if err := DB.AutoMigrate(model); err != nil {
			log.Printf("同步%v模型结构失败: %v", modelName, err)
		} else {
			log.Printf("%v 模型结构同步成功", modelName)
		}
	}

	log.Println("数据库迁移完成！")
}

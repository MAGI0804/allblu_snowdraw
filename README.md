# 优蓝童装 Go 后端项目

这是一个基于 Go 语言的童装定制后端项目，通过 Gin 框架和 GORM 库实现，旨在替代原有的 Django 后端。

## 项目结构

```
django_to_go/
├── config/            # 配置相关
│   └── config.go      # 配置文件
├── controllers/       # 控制器
│   ├── user_controller.go     # 用户控制器
│   ├── commodity_controller.go # 商品控制器
│   └── order_controller.go    # 订单控制器
├── db/                # 数据库相关
│   └── db.go          # 数据库连接
├── middleware/        # 中间件
│   └── auth_middleware.go     # 认证中间件
├── models/            # 数据模型
│   ├── user.go        # 用户模型
│   ├── commodity.go   # 商品模型
│   ├── order.go       # 订单模型
│   ├── address.go     # 地址模型
│   └── cart.go        # 购物车模型
├── routes/            # 路由配置
│   └── routes.go      # 路由注册
├── utils/             # 工具函数
│   └── utils.go       # 通用工具函数
├── main.go            # 项目入口
├── go.mod             # Go模块定义
└── README.md          # 项目说明
```

## 依赖项

- Gin: Web框架
- GORM: ORM库
- MySQL驱动: 数据库连接
- JWT: 认证

## 配置说明

配置信息存储在 `config/config.go` 文件中，可以通过环境变量或默认值设置。

主要配置项包括：
- 数据库连接信息
- JWT密钥和过期时间
- 微信小程序配置

## 如何运行

1. 确保已安装 Go 1.21 或更高版本
2. 安装依赖：`go mod download`
3. 确保 MySQL 数据库已配置正确
4. 运行项目：`go run main.go`

默认情况下，服务将在 http://localhost:8080 启动。

## 主要功能

1. **用户管理**
   - 用户注册
   - 用户查询
   - 用户信息修改
   - 微信小程序登录
   - 登录状态验证

2. **商品管理**
   - 商品列表
   - 商品详情
   - 商品创建
   - 商品更新
   - 商品删除

3. **订单管理**
   - 订单列表
   - 订单详情
   - 订单创建
   - 订单更新
   - 订单取消
   - 订单支付
   - 订单发货

## API 路由

API 路由配置在 `routes/routes.go` 文件中，所有路由都以 `/api/` 开头。

## 注意事项

- 项目使用 GORM 作为 ORM 库，支持数据库迁移
- 所有 API 响应都采用统一的 JSON 格式
- 认证使用 JWT 令牌机制
- 支持跨域请求
- 提供了静态文件服务

## 开发建议

1. 在添加新功能时，请先创建对应的模型和控制器
2. 使用中间件进行权限验证和日志记录
3. 保持代码风格一致
4. 添加适当的错误处理
5. 定期进行代码审查

## 部署说明

### 本地开发环境

1. 确保安装了Go 1.21或更高版本
2. 确保MySQL数据库已配置并运行
3. 使用启动脚本运行：`./run.sh`
   或者手动执行：
   ```bash
   go mod tidy
   go build -o youlan_kids_go main.go
   ./youlan_kids_go
   ```

### Docker部署

1. 确保已安装Docker和docker-compose
2. 在项目根目录运行：
   ```bash
   docker-compose up --build -d
   ```
3. 应用将在8080端口启动

### 环境变量配置

应用支持通过环境变量覆盖默认配置：

- `DB_HOST`: 数据库主机地址
- `DB_PORT`: 数据库端口
- `DB_NAME`: 数据库名称
- `DB_USER`: 数据库用户名
- `DB_PASSWORD`: 数据库密码
- `SECRET_KEY`: JWT密钥
- `JWT_ACCESS_TOKEN_LIFETIME`: JWT访问令牌有效期（分钟）
- `JWT_REFRESH_TOKEN_LIFETIME`: JWT刷新令牌有效期（分钟）

## License

本项目使用 MIT 许可证。
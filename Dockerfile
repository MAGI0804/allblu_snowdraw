# 构建阶段
FROM golang:1.24-alpine AS builder

# 1. 重置环境变量（启用模块模式）
ENV GO111MODULE=on \
    GOPATH="" \
    GOMODCACHE="/go/pkg/mod"

# 2. 设置工作目录（与你的 django_to_go 根目录对应）
WORKDIR /app

# 3. 安装工具 + 修复权限
RUN apk add --no-cache git && \
    git config --global http.sslVerify false && \
    chmod -R 755 /app

# 4. 复制所有文件（你的目录结构：config、db、main.go 等都在根目录）
COPY . .

# 5. 验证目录和模块（修正注释位置，避免命令断裂）
RUN echo "=== 验证根目录文件 ===" && \
    ls -la /app && \
    echo "=== 验证 go.mod 内容 ===" && \
    cat /app/go.mod && \
    echo "=== 验证子包目录 ===" && \
    ls -la /app/config && \
    ls -la /app/db && \
    echo "=== 验证模块信息 ===" && \
    go list -m django_to_go && \
    echo "=== 验证 config 子包 ===" && \
    go list django_to_go/config || echo "子包未识别（检查目录和 package 声明）"

# 6. 国内代理
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off

# 7. 整理依赖
RUN go mod tidy

# 8. 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o youlan_kids_go main.go

# 运行阶段
FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/youlan_kids_go .
# 复制media目录到容器中
COPY --from=builder /app/media ./media
# 创建logs目录并设置权限
RUN mkdir -p ./logs && chmod -R 777 ./logs
# 设置media目录权限以允许文件写入
RUN chmod -R 777 ./media
# 添加执行权限
RUN chmod +x ./youlan_kids_go
EXPOSE 8080
ENV TZ=Asia/Shanghai
CMD ["./youlan_kids_go"]
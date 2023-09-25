# 使用官方的Golang基础镜像
FROM golang:1.18 AS builder

# 设置工作目录
WORKDIR /sealdice-core-app

# 复制项目文件到容器中
COPY . .

# 设置一些环境变量
#ENV CGO_ENABLED=0
#ENV GOOS=linux
ENV GOPROXY=https://goproxy.cn,direct
#ENV GOSUMDB=off

# 下载依赖并构建应用程序
RUN go mod download
RUN go mod tidy
RUN go generate ./...
RUN go install github.com/pointlander/peg@v1.0.1
RUN go build -o sealdice-core-app


# 使用一个轻量的Alpine Linux基础镜像作为最终的容器
FROM ubuntu

# 设置工作目录
WORKDIR /sealdice-core-app

# 复制构建好的二进制文件到最终的容器中
COPY --from=builder /sealdice-core-app/sealdice-core-app .

# 暴露容器内部的3211端口
EXPOSE 3211
RUN ls
RUN pwd
# 运行你的应用程序
CMD ["./sealdice-core-app"]

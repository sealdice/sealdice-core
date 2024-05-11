FROM alpine:3.19.1

# 指定默认时区
RUN apk add --no-cache tzdata
ENV TZ Asia/Shanghai

# 移入对应文件
VOLUME ["/backups", "/data"]
ARG TARGETARCH
COPY ./${TARGETARCH}/data   /data
COPY --chmod=0755 ./${TARGETARCH}/sealdice-core  .

# 启动
EXPOSE 3211
ENTRYPOINT ["./sealdice-core", "--container-mode"]
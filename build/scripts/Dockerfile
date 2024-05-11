FROM alpine:3.19.1

# 指定默认时区
RUN apk add --no-cache tzdata
ENV TZ Asia/Shanghai

# 移入对应文件
ARG TARGETARCH

COPY ./${TARGETARCH}  .
COPY --chmod=0755 ./${TARGETARCH}/sealdice-core  .

EXPOSE 3211

ENTRYPOINT ["./sealdice-core", "--container-mode"]
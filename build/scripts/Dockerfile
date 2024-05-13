FROM alpine:3.19.1

# 指定默认时区
RUN apk add --no-cache tzdata
ENV TZ Asia/Shanghai

# 移入对应文件
VOLUME ["/backups", "/data"]
ARG TARGETARCH
COPY ./${TARGETARCH}/data /buildin-data
COPY --chmod=0755 ./${TARGETARCH}/sealdice-core  .

RUN echo "#!/bin/sh" > ./entrypoint.sh
RUN echo "cp -r ./buildin-data/* ./data/" >> ./entrypoint.sh
RUN echo "./sealdice-core --container-mode" >> ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

# 启动
EXPOSE 3211
ENTRYPOINT [ "./entrypoint.sh" ]

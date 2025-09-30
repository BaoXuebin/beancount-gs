# 构建 beancount-gs
FROM golang:1.17.3 AS go_builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GIN_MODE=release \
    CGO_ENABLED=0 \
    PORT=80

WORKDIR /app
COPY . .
COPY public/icons ./public/default_icons
RUN go build .

# 镜像
FROM frankwuzp/beancount-gs:v1.2.2

WORKDIR /app
COPY --from=go_builder /app/beancount-gs ./
COPY --from=go_builder /app/template ./template
COPY --from=go_builder /app/config ./config
COPY --from=go_builder /app/public ./public
COPY --from=go_builder /app/logs ./logs

EXPOSE 80

ENTRYPOINT [ "/bin/sh", "-c", "cp -rn /app/public/default_icons/* /app/public/icons && /app/beancount-gs -p 80" ]
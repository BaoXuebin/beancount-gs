# 构建 beancount-gs
FROM golang:1.17.3-alpine AS go_builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GIN_MODE=release \
    CGO_ENABLED=0 \
    PORT=80

WORKDIR /build
COPY . .
COPY public/icons /build/public/default_icons
RUN go build .

# 镜像
FROM python:3.11.9-alpine3.19

WORKDIR /app

RUN echo "https://mirrors.aliyun.com/alpine/v3.16/main/" > /etc/apk/repositories  \
    && echo "https://mirrors.aliyun.com/alpine/v3.16/community/" >> /etc/apk/repositories \
    && set -x \
    && apk update \
    && apk add --no-cache gcc musl-dev \
    && python3 -mvenv /app/beancount \
    && /app/beancount/bin/pip install --no-cache-dir beanquery -i https://mirrors.aliyun.com/pypi/simple/ \
    && apk del gcc musl-dev

COPY --from=go_builder /build/beancount-gs /app
COPY --from=go_builder /build/template /app/template
COPY --from=go_builder /build/config /app/config
COPY --from=go_builder /build/public /app/public
COPY --from=go_builder /build/logs /app/logs

ENV LANG=C.UTF-8 \
    SHELL=/bin/bash \
    PS1="\u@\h:\w \$ " \
    PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/app/bin:/app/beancount/bin"

EXPOSE 80

ENTRYPOINT [ "/bin/sh", "-c", "cp -rn /app/public/default_icons/* /app/public/icons && /app/beancount-gs -p 80" ]
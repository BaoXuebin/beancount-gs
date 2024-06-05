# 构建 beancount
FROM python:3.7.16 as beancount_builder
WORKDIR /build
ENV PATH "/app/bin:$PATH"
RUN python3 -mvenv /app
RUN wget https://github.com/beancount/beancount/archive/refs/tags/2.3.5.tar.gz
RUN tar -zxvf 2.3.5.tar.gz
RUN python3 -m pip install ./beancount-2.3.5 -i https://mirrors.aliyun.com/pypi/simple/
RUN find /app -name __pycache__ -exec rm -rf -v {} +

# 构建 beancount-gs
FROM golang:1.17.3 AS go_builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GIN_MODE=release \
    CGO_ENABLED=0 \
    PORT=80

WORKDIR /build
COPY . .
COPY public/icons ./public/default_icons
RUN go build .

# 镜像
FROM python:3.7.16-alpine

COPY --from=beancount_builder /app /app

WORKDIR /app
COPY --from=go_builder /build/beancount-gs ./
COPY --from=go_builder /build/template ./template
COPY --from=go_builder /build/config ./config
COPY --from=go_builder /build/public ./public
COPY --from=go_builder /build/logs ./logs

ENV PATH "/app/bin:$PATH"
EXPOSE 80
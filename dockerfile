ARG BEANCOUNT_VERSION=2.3.5
ARG GOLANG_VERSION=1.17.3

# 构建 beancount
FROM python:latest as beancount_builder
WORKDIR /build
ENV PATH "/app/bin:$PATH"
RUN python3 -mvenv /app
RUN git clone -b ${BEANCOUNT_VERSION} https://github.com/beancount/beancount.git
RUN python3 -m pip install ./beancount -i https://mirrors.aliyun.com/pypi/simple/
RUN find /app -name __pycache__ -exec rm -rf -v {} +

# 构建 beancount-gs
FROM golang:${GOLANG_VERSION} AS go_builder

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
FROM python:3.10-alpine

COPY --from=beancount_builder /app /app

WORKDIR /app
COPY --from=go_builder /build/beancount-gs ./
COPY --from=go_builder /build/template ./template
COPY --from=go_builder /build/config ./config
COPY --from=go_builder /build/public ./public
COPY --from=go_builder /build/logs ./logs

ENV PATH "/app/bin:$PATH"
EXPOSE 80
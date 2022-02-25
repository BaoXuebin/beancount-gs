# syntax=docker/dockerfile:1
FROM golang:latest AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GIN_MODE=release \
    PORT=80

WORKDIR /builder
COPY . .
COPY public/icons ./public/default_icons
RUN go build .

FROM python:latest
RUN pip3 install beancount -i https://pypi.tuna.tsinghua.edu.cn/simple

WORKDIR /app
COPY --from=builder ./builder/public ./public
COPY --from=builder ./builder/config ./config
COPY --from=builder ./builder/template ./template
COPY --from=builder ./builder/beancount-gs* ./

EXPOSE 80
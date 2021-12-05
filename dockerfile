# syntax=docker/dockerfile:1
FROM golang:1.17.3

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GIN_MODE=release \
    PORT=80

# install beancount
RUN apt-get update || : && apt-get install python3.5 python3-pip -y
RUN pip3 install beancount -i https://pypi.tuna.tsinghua.edu.cn/simple

WORKDIR /app
COPY . .
COPY public/icons ./public/default_icons
RUN go build .

EXPOSE 80
# syntax=docker/dockerfile:1
FROM golang:1.17.3 AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GIN_MODE=release \
    PORT=80

WORKDIR /builder

COPY . .
COPY public/icons ./public/default_icons
RUN go build .

FROM python:3.8
RUN python3 -m pip install -U pip setuptools wheel -i https://pypi.tuna.tsinghua.edu.cn/simple

COPY ./beancount-2.3.4-cp38-cp38-linux_x86_64.whl /tmp
RUN pip3 install /tmp/beancount-2.3.4-cp38-cp38-linux_x86_64.whl -i https://pypi.tuna.tsinghua.edu.cn/simple

COPY ./fava-1.18-py3-none-any.whl /tmp
RUN pip3 install /tmp/fava-1.18-py3-none-any.whl -i https://pypi.tuna.tsinghua.edu.cn/simple


WORKDIR /app
COPY --from=builder ./builder/public ./public
COPY --from=builder ./builder/config ./config
COPY --from=builder ./builder/template ./template
COPY --from=builder ./builder/beancount-gs* ./

EXPOSE 80
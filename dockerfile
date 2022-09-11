ARG BEANCOUNT_VERSION=2.3.5
ARG GOLANG_VERSION=1.17.3

FROM golang:${GOLANG_VERSION} AS go_build_env

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GIN_MODE=release \
    CGO_ENABLED=0 \
    PORT=80

WORKDIR /tmp/build
RUN git clone https://github.com/frankwuzp/beancount-gs.git

WORKDIR /tmp/build/beancount-gs
RUN mkdir -p public/default_icons && cp -rn public/icons/* public/default_icons

RUN go build .

FROM python:latest as build_env
ARG BEANCOUNT_VERSION

ENV PATH "/app/bin:$PATH"
RUN python3 -mvenv /app

WORKDIR /tmp/build
RUN git clone https://github.com/beancount/beancount

WORKDIR /tmp/build/beancount
RUN git checkout ${BEANCOUNT_VERSION}

RUN CFLAGS=-s pip3 install -U /tmp/build/beancount

RUN pip3 uninstall -y pip

RUN find /app -name __pycache__ -exec rm -rf -v {} +

FROM python:3.10-alpine

COPY --from=build_env /app /app

WORKDIR /app
COPY --from=go_build_env /tmp/build/beancount-gs /app

# volumes 挂载目录会导 /app/public/icons 中的图标被覆盖，这里将默认图标在挂载后重新拷贝图标
RUN cp -rn /app/public/default_icons/* /app/public/icons

ENV PATH "/app/bin:$PATH"

EXPOSE 80

CMD ["/app/beancount-gs", "-p", "80"]

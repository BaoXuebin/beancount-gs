# 第一阶段：构建阶段
FROM python:3.11.9-alpine3.19 as builder

# 设置环境变量，防止 Python 创建 .pyc 文件
ENV PYTHONUNBUFFERED=1

# 替换为阿里云的镜像源，并安装必要的依赖
RUN echo "https://mirrors.aliyun.com/alpine/v3.15/main/" > /etc/apk/repositories && \
    echo "https://mirrors.aliyun.com/alpine/v3.15/community/" >> /etc/apk/repositories && \
    apk update && \
    apk add --no-cache --virtual .build-deps \
        gcc \
        g++ \
        musl-dev

# 设置工作目录
WORKDIR /app

# 创建虚拟环境
RUN python3 -m venv /app/venv

# 将 Beancount 源码压缩包复制到容器中
COPY beancount-2.3.6.tar.gz /app

# 解压 Beancount 源码到 /beancount 目录
RUN mkdir /beancount && \
    tar -xzf /app/beancount-2.3.6.tar.gz -C /beancount --strip-components=1

# 激活虚拟环境并安装 Beancount
RUN /app/venv/bin/pip install --upgrade pip -i https://mirrors.aliyun.com/pypi/simple/ && \
    /app/venv/bin/pip install /beancount -i https://mirrors.aliyun.com/pypi/simple/ && \
    # 清理不必要的文件
    rm -rf /app/beancount-2.3.6.tar.gz && \
    find /app -name __pycache__ -exec rm -rf -v {} +

# 第二阶段：运行阶段
FROM python:3.11.9-alpine3.19

# 设置环境变量，防止 Python 创建 .pyc 文件
ENV PYTHONUNBUFFERED=1

# 设置工作目录
WORKDIR /app

# 从构建阶段复制虚拟环境到当前镜像
COPY --from=builder /app/venv /app/venv

# 将虚拟环境的 bin 目录添加到 PATH 环境变量
ENV PATH="/app/venv/bin:$PATH"

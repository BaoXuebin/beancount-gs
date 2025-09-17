#!/bin/bash

###
 # @Author: liangzai450
 # @Date: 2025-09-06 13:18:36
 # @LastEditors: liangzai450
 # @LastEditTime: 2025-09-12 23:28:12
 # @FilePath: \\beancount-gs-wu\\start_dev.sh
 # @Description: Beancount v3 开发环境自动配置脚本
 # Copyright (c) 2025 by ${git_name_email}, All Rights Reserved. 
 # ==============================================
### 

echo "[Beancount v3 开发环境自动配置脚本]"
echo "========================================"

# 检查是否在Git Bash中运行
if [[ "$OSTYPE" != "msys" ]]; then
    echo "警告: 此脚本专为Git Bash设计，当前环境: $OSTYPE"
    read -p "是否继续? (y/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 基础路径设置
WINDOWS_DRIVE="H:"
GITBASH_DRIVE="/h"
DEV_DIR="dev"

# 路径转换函数
gitbash_to_win() {
    echo "$1" | sed 's#^/\([a-z]\)#\1:#' | sed 's#/#\\#g'
}

win_to_gitbash() {
    echo "$1" | sed 's#\([a-zA-Z]\):#/\1#' | sed 's#\\#/#g'
}

# 设置路径
DEV_ROOT="$GITBASH_DRIVE/$DEV_DIR"
WIN_DEV_ROOT="$WINDOWS_DRIVE\\$DEV_DIR"
export DEV_ROOT
export WIN_DEV_ROOT

# 添加工具到PATH
export PATH="$DEV_ROOT/dev_tools/nodejs:$DEV_ROOT/dev_tools/git/bin:$PATH"

# Python路径
PYTHON_GITBASH_PATH="$DEV_ROOT/dev_envs/winpython-3.12.10-dot/WPy64-312101/python"
PYTHON_WIN_PATH=$(gitbash_to_win "$PYTHON_GITBASH_PATH")

if [ -f "$PYTHON_GITBASH_PATH/python.exe" ]; then
    export PATH="$PYTHON_GITBASH_PATH:$PYTHON_GITBASH_PATH/Scripts:$PATH"
    echo "✓ Python已添加到PATH"
else
    echo "✗ Python未找到: $PYTHON_GITBASH_PATH/python.exe"
    exit 1
fi

# 设置别名
alias python="winpty $PYTHON_WIN_PATH/python.exe"
alias pip="winpty $PYTHON_WIN_PATH/python.exe -m pip"

echo ""
echo "环境设置完成！"
echo "========================================"

# 进入项目目录,# 进入脚本所在目录
cd "$(dirname "$0")"   || { echo "无法进入项目目录"; exit 1; }

# 检查Python版本
echo "检查Python版本..."
python --version

# 定义虚拟环境名称和依赖文件
VENV_NAME=".env_beancount-v3"
REQUIREMENTS_FILE="requirements-beancount-v3.txt"

# 函数：创建虚拟环境
create_venv() {
    echo "创建虚拟环境: $VENV_NAME..."
    python -m venv "$VENV_NAME"
    
    if [ $? -eq 0 ]; then
        echo "✓ 虚拟环境创建成功"
        return 0
    else
        echo "✗ 虚拟环境创建失败"
        return 1
    fi
}

# 函数：激活虚拟环境
activate_venv() {
    echo "激活虚拟环境..."
    source "$VENV_NAME/Scripts/activate"
    
    if [ $? -eq 0 ]; then
        echo "✓ 虚拟环境激活成功"
        python --version
        return 0
    else
        echo "✗ 虚拟环境激活失败"
        return 1
    fi
}

# 函数：安装依赖
install_dependencies() {
    echo "安装Beancount v3依赖..."
    
    # 升级pip
    pip install --upgrade pip
    
    # 安装固定版本的依赖
    echo "安装固定版本的依赖..."
    pip install \
        beancount==3.1.0 \  
        beanquery==0.2.0 \  
        fava==1.30.5 \
        beangulp==0.2.0 \   
        dateparser==1.2.2 \
        debugpy==1.8.16 \
        pytest==8.4.2 \
        Pygments==2.19.2 \
        pyzipper==0.3.6

    if [ $? -eq 0 ]; then
        echo "✓ 依赖安装成功"
        return 0
    else
        echo "✗ 依赖安装失败"
        return 1
    fi

    # # 安装固定版本的依赖
    # echo "安装fava..."
    # pip install fava==1.30.5
    
    # echo "安装dateparser..."
    # pip install dateparser==1.2.2
    
    # echo "安装debugpy..."
    # pip install debugpy==1.8.16
    
    # echo "安装pytest..."
    # pip install pytest==8.4.2
    
    # echo "安装Pygments..."
    # pip install Pygments==2.19.2
    
    # echo "安装pyzipper..."
    # pip install pyzipper==0.3.6
    
    # # 如果需要安装其他beancount相关包，可以取消注释并添加版本号
    # # echo "安装beancount..."
    # # pip install beancount==3.0.0
    
    # # echo "安装beanquery..."
    # # pip install beanquery==1.0.0
    
    # # echo "安装beangulp..."
    # # pip install beangulp==0.5.0
    
    # # echo "安装bean-price..."
    # # pip install bean-price==1.0.0
    
    # 生成requirements文件
    pip freeze > "$REQUIREMENTS_FILE"
    echo "✓ 依赖已保存到 $REQUIREMENTS_FILE"
}

# 函数：验证安装
verify_installation() {
    echo "验证安装..."
    
    echo "1. 检查fava版本:"
    python -c "import fava; print(f'Fava版本: {fava.__version__}')"
    
    echo "2. 检查dateparser版本:"
    python -c "import dateparser; print(f'Dateparser版本: {dateparser.__version__}')"
    
    echo "3. 检查debugpy版本:"
    python -c "import debugpy; print(f'Debugpy版本: {debugpy.__version__}')"
    
    echo "4. 检查pytest版本:"
    python -c "import pytest; print(f'Pytest版本: {pytest.__version__}')"
    
    echo "5. 检查Pygments版本:"
    python -c "import pygments; print(f'Pygments版本: {pygments.__version__}')"
    
    echo "6. 检查pyzipper:"
    python -c "import pyzipper; print(f'Pyzipper版本: {pyzipper.__version__}')" 2>/dev/null || python -c "import pyzipper; print('Pyzipper导入成功')"
    
    # 如果需要验证其他beancount相关包，可以取消注释
    echo "7. 检查beancount版本:"
    python -c "import beancount; print(f'Beancount版本: {beancount.__version__}')"
    
    echo "8. 检查beanquery:"
    python -c "import beanquery; print('Beanquery导入成功')"
    
    echo "✓ 所有依赖安装验证完成"
}

# 主执行流程
main() {
    # 检查虚拟环境是否存在
    if [ ! -d "$VENV_NAME" ]; then
        echo "虚拟环境不存在，开始创建..."
        create_venv || exit 1
    else
        echo "虚拟环境已存在，跳过创建"
        echo "当前路径为: $(pwd)"
    fi
    
    # 激活虚拟环境
    activate_venv || exit 1
    
    # 检查是否已安装beancount
    if ! python -c "import beancount" 2>/dev/null; then
        echo "Beancount未安装，开始安装依赖..."
        install_dependencies
    else
        echo "Beancount已安装，跳过依赖安装"
    fi
    
    # 验证安装
    verify_installation
    
    echo ""
    echo "========================================"
    echo "🎉 Beancount v3 开发环境配置完成！"
    echo "虚拟环境: $VENV_NAME"
    echo "项目目录: $(pwd)"
    echo ""
    echo "可用命令:"
    echo "  source ./.env_beancount-v3/Scripts/activate # 激活虚拟环境"
    echo "  bean-check your_file.bean    # 检查语法"
    echo "  bean-query your_file.bean    # 执行查询"
    echo "  fava your_file.bean          # 启动Web界面"
    echo "========================================"
    
    # 保持脚本运行在激活的虚拟环境中
    source ./.env_beancount-v3/Scripts/activate
    # 保持在激活的虚拟环境中
    exec bash
}

# 执行主函数
main
# Beancount-gs Script

```bash

在本地安装python3环境，

# 创建虚拟环境

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
    source "$VENV_NAME/bin/activate"
    
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
    
    echo "7. 检查beancount版本:"
    python -c "import beancount; print(f'Beancount版本: {beancount.__version__}')"
    
    echo "8. 检查beanquery:"
    python -c "import beanquery; print('Beanquery导入成功')"
    
    echo "✓ 所有依赖安装验证完成"
}

# 函数：为Beancount添加中文账户名支持
patch_beancount_for_chinese() {
    echo ""
    echo "为Beancount添加中文账户名支持..."
    
    # 设置颜色输出
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    RED='\033[0;31m'
    NC='\033[0m' # No Color
    
    # 直接定位account.py文件
    ACCOUNT_FILE="$VENV_NAME/lib/python"*"/site-packages/beancount/core/account.py"
    
    # 使用通配符展开找到确切的文件路径
    ACCOUNT_FILES=$(ls $ACCOUNT_FILE 2>/dev/null)
    
    if [ -z "$ACCOUNT_FILES" ]; then
        echo -e "${RED}错误: 未找到beancount库的account.py文件${NC}"
        
        # 非CNB环境中询问用户
        echo -e "${YELLOW}是否继续执行脚本？(跳过中文支持补丁) [Y/n]${NC}"
        read -r response
        if [[ "$response" =~ ^([nN][oO]|[nN])$ ]]; then
            echo "用户选择中断脚本执行"
            return 1
        else
            echo "跳过中文支持补丁但继续执行脚本其他内容"
            return 0
        fi

    fi
    
    echo -e "${GREEN}找到account.py文件:${NC}"
    echo "$ACCOUNT_FILES"
    
    # 为每个找到的文件创建备份并进行替换
    for FILE in $ACCOUNT_FILES; do
        echo -e "${YELLOW}处理文件: $FILE${NC}"
        
        # 创建备份
        BACKUP="${FILE}.bak"
        cp "$FILE" "$BACKUP"
        echo -e "${GREEN}已创建备份: $BACKUP${NC}"
        
        # 使用grep查找包含"Component separator for account names"的行号
        START_LINE=$(grep -n "Component separator for account names" "$FILE" | cut -d: -f1)
        
        if [ -z "$START_LINE" ]; then
            echo -e "${RED}错误: 无法找到起始行标记${NC}"
            echo -e "${YELLOW}尝试使用固定行号替换...${NC}"
            START_LINE=28
        fi
        
        # 使用grep查找包含"TYPE = \"<AccountDummy>\""的行号
        END_LINE=$(grep -n "TYPE = \"<AccountDummy>\"" "$FILE" | cut -d: -f1)
        
        if [ -z "$END_LINE" ]; then
            echo -e "${RED}错误: 无法找到结束行标记${NC}"
            echo -e "${YELLOW}尝试使用固定行号替换...${NC}"
            END_LINE=41
        fi
        
        echo -e "${GREEN}找到替换范围: 第${START_LINE}行到第${END_LINE}行${NC}"
        
        # 使用sed进行替换，使用动态行号
        sed -i.tmp "${START_LINE},${END_LINE}c\\
# Component separator for account names.\\
sep = \":\"\\
\\
\\
# Regular expression string that matches valid account name components.\\
# Categories are:\\
#   Lu: Uppercase letters.\\
#   L: All letters.\\
#   Nd: Decimal numbers.\\
ACC_COMP_TYPE_RE = (\\
    r\"[\\\\p{Lu}][\\\\p{L}\\\\p{Nd}\\\\-]*\"  # Root account type (e.g. Assets or Income)\\
)\\
ACC_COMP_NAME_RE = r\"[\\\\p{Han}\\\\p{Lu}][\\\\p{Han}\\\\p{L}\\\\p{Nd}\\\\-]*\"  # Account name components (e.g. Cash or 现金)\\
# ACC_COMP_NAME_RE = r\"[\\\\p{Han}\\\\p{Lu}\\\\p{Nd}][\\\\p{Han}\\\\p{Lu}\\\\p{Nd}\\\\-（）()·—、，,\\\\.]*\" # Account name components (e.g. Cash or 现—金)\\
\\
# Regular expression string that matches a valid account. {5672c7270e1e}\\
ACCOUNT_RE = r\"(?:{})(?:{}{})+\".format(ACC_COMP_TYPE_RE, sep, ACC_COMP_NAME_RE)\\
\\
\\
# A dummy object which stands for the account type. Values in custom directives\\
# use this to disambiguate between string objects and account names.\\
TYPE = \"<AccountDummy>\"\\
" "$FILE"
        
        # 检查替换是否成功
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}文件已成功更新${NC}"
            # 删除临时文件
            rm -f "${FILE}.tmp"
        else
            echo -e "${RED}更新失败，正在恢复备份${NC}"
            cp "$BACKUP" "$FILE"
        fi
    done
    
    echo -e "${GREEN}中文账户名支持已添加${NC}"
    echo -e "${YELLOW}现在可以使用如下中文账户名:${NC}"
    echo -e "  Assets:银行:工商银行:储蓄卡"
    echo -e "  Income:工资:基本工资"
    echo -e "  Expenses:食品:早餐"
}


# 函数：构建beancount-gs程序
build_beancount_gs() {
    echo "构建beancount-gs程序..."
    
    # 检查是否在虚拟环境中
    if [ -z "$VIRTUAL_ENV" ]; then
        echo "不在虚拟环境中，尝试激活..."
        source "$VENV_NAME/bin/activate"
    fi
    
    # 构建程序
    go build -o beancount-gs .
    
    if [ $? -eq 0 ] && [ -f "beancount-gs" ]; then
        echo "✓ beancount-gs构建成功"
        chmod +x beancount-gs
    else
        echo "✗ beancount-gs构建失败"
        return 1
    fi
}

# 函数：清理端口占用
cleanup_port() {
    local port=$1
    echo "检查端口 $port 占用情况..."
    
    # 检查端口是否被占用
    if lsof -i :$port >/dev/null 2>&1; then
        echo "端口 $port 被占用，正在清理..."
        
        # 获取占用端口的进程ID并杀死
        pids=$(lsof -ti :$port)
        if [ -n "$pids" ]; then
            echo "杀死占用端口的进程: $pids"
            kill -9 $pids 2>/dev/null
            sleep 2  # 等待进程完全终止
            
            # 再次检查是否清理成功
            if lsof -i :$port >/dev/null 2>&1; then
                echo "警告: 无法完全清理端口 $port 的占用"
                return 1
            else
                echo "✓ 端口 $port 已清理完成"
                return 0
            fi
        fi
    else
        echo "✓ 端口 $port 空闲可用"
        return 0
    fi
}

# 函数：启动beancount-gs web界面
start_beancount_gs() {
    echo "启动beancount-gs web界面..."
    
    local port=10000
    local secret="B8nK2dL7qR4tY9"
    local debug = false
    
    # 清理端口占用
    if ! cleanup_port $port; then
        echo "✗ 无法清理端口 $port，启动失败"
        return 1
    fi
    
    if [ -f "beancount-gs" ]; then
        echo "启动beancount-gs web界面..."
        echo "当前debug模式: $debugFlag"
        # 启动程序并指定端口（如果程序支持端口参数）
        # ./beancount-gs --p $port -secret $secret -debug $debug &
        ./beancount-gs --p $port -secret $secret &
        local pid=$!
        
        # 等待程序启动
        sleep 3
        
        # 检查程序是否正常运行
        if ps -p $pid >/dev/null 2>&1; then
            echo "✓ beancount-gs已启动 (PID: $pid, 端口: $port, 密钥: $secret)"
            echo "Web界面地址: http://localhost:$port"
            return 0
        else
            echo "✗ beancount-gs启动失败"
            return 1
        fi
    else
        echo "✗ beancount-gs可执行文件不存在，请先构建"
        return 1
    fi
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
    
    # 添加中文账户名支持
    patch_beancount_for_chinese
    echo "现在可以使用中文账户名了！"

    # 构建beancount-gs程序
    build_beancount_gs
 
    # 启动beancount-gs web界面
    start_beancount_gs
    
    
    echo ""
    echo "========================================"
    echo "🎉 Beancount v3 开发环境配置完成！"
    echo "========================================"
}

# 执行主函数
main "$@"

```

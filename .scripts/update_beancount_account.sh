#!/bin/bash

# 脚本用于查找并更新beancount库中的account.py文件
# 添加对中文账户名的支持

# 设置颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}开始查找beancount库的account.py文件...${NC}"

# 查找所有可能的account.py文件
ACCOUNT_FILES=$(find / -path "*/site-packages/beancount/core/account.py" 2>/dev/null)

if [ -z "$ACCOUNT_FILES" ]; then
    echo -e "${RED}错误: 未找到beancount库的account.py文件${NC}"
    echo "请确保beancount已安装，或者尝试以root权限运行此脚本"
    exit 1
fi

echo -e "${GREEN}找到以下account.py文件:${NC}"
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

echo -e "${GREEN}所有文件处理完成${NC}"
echo -e "${YELLOW}现在beancount支持中文账户名了!${NC}"
echo -e "例如: Assets:银行:工商银行:储蓄卡"
echo -e "      Income:工资:基本工资"
echo -e "      Expenses:食品:早餐"

exit 0
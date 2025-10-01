package service

import (
	"fmt"

	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
)

type Tags struct {
	Value string `bql:"distinct tags" json:"value"`
}

func QueryTags(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)
	tags := make([]Tags, 0)

	// 使用 script.QueryParams 构建查询条件
	queryParams := &script.QueryParams{
		Where:      true,   // 启用 WHERE 子句
		Tag:        "tags", // tag in tags 条件
		TagNotNull: "true", // tag 非空条件（值可以是任意非空字符串）
	}

	err := script.BQLQueryList(ledgerConfig, queryParams, &tags)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	result := make([]string, 0)
	for _, t := range tags {
		if t.Value != "" {
			result = append(result, t.Value)
			// 记录每个标签值
			script.LogDebugDetailed(ledgerConfig.Mail, "TagValue", "标签值: %s", t.Value)
		}
	}

	// 记录最终返回结果
	script.LogDebug(ledgerConfig.Mail, fmt.Sprintf("返回 %d 个标签", len(result)))

	OK(c, result)
}

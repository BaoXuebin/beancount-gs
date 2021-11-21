package service

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"os/exec"
	"strings"
)

func MonthsList(c *gin.Context) {
	months := make([]string, 0)

	ledgerConfig := script.GetLedgerConfigFromContext(c)
	beanFilePath := ledgerConfig.DataPath + "/index.bean"
	bql := "SELECT distinct year(date), month(date)"
	cmd := exec.Command("bean-query", beanFilePath, bql)
	output, err := cmd.Output()
	if err != nil {
		InternalError(c, "Failed to exec bql")
		return
	}
	execResult := string(output)
	months = make([]string, 0)
	for _, line := range strings.Split(execResult, "\n")[2:] {
		if line != "" {
			yearMonth := strings.Fields(line)
			months = append(months, yearMonth[0]+"-"+yearMonth[1])
		}
	}
	OK(c, months)
}

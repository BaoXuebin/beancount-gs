package service

import (
	"github.com/beancount-gs/script"
	"github.com/gin-gonic/gin"
	"os/exec"
)

func MonthsList(c *gin.Context) {
	ledgerConfig := script.GetLedgerConfigFromContext(c)

	beanFilePath := ledgerConfig.DataPath + "/index.bean"
	bql := "SELECT distinct year(date), month(date)"
	cmd := exec.Command("bean-query " + beanFilePath + " \"" + bql + "\"")
	output, err := cmd.CombinedOutput()
	if err != nil {
		InternalError(c, "Failed to exec bql")
		return
	}

	script.LogInfo(string(output))
	OK(c, "")
}

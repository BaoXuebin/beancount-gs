package script

import (
	"fmt"
	"time"
)

func LogInfo(ledgerName string, message string) {
	fmt.Printf("[Info] [%s] [%s]: %s\n", time.Now().Format("2006-01-02 15:04:05"), ledgerName, message)
}

func LogSystemInfo(message string) {
	LogInfo("System", message)
}

func LogError(ledgerName string, message string) {
	fmt.Printf("[Error] [%s] [%s]: %s\n", time.Now().Format("2006-01-02 15:04:05"), ledgerName, message)
}

func LogSystemError(message string) {
	LogError("System", message)
}

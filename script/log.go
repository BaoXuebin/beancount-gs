package script

import (
	"fmt"
	"time"
)

func LogInfo(message string) {
	fmt.Printf("[Info] [%s] System: %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
}

func LogError(message string) {
	fmt.Printf("[Error] [%s] System: %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
}

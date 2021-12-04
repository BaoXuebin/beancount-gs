package script

import (
	"os/exec"
	"runtime"
)

func isWindows() bool {
	os := runtime.GOOS
	return os == "windows"
}

func OpenBrowser(url string) {
	if isWindows() {
		cmd := exec.Command("cmd", "/C", "start", url)
		err := cmd.Start()
		if err != nil {
			LogSystemError("Failed to open browser, error is " + err.Error())
		}
	}
}

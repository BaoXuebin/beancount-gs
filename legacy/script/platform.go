package script

import (
	"os/exec"
	"runtime"
)

func isWindows() bool {
	os := runtime.GOOS
	return os == "windows"
}

func isMacOS() bool {
	os := runtime.GOOS
	return os == "darwin"
}

func OpenBrowser(url string) {
	if isWindows() {
		cmd := exec.Command("cmd", "/C", "start", url)
		err := cmd.Start()
		if err != nil {
			LogSystemError("Failed to open browser, error is " + err.Error())
		}
	} else if isMacOS() {
		cmd := exec.Command("open", url)
		err := cmd.Start()
		if err != nil {
			LogSystemError("Failed to open browser, error is " + err.Error())
		}
	}
}

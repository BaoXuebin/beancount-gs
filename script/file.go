package script

import (
	"io/ioutil"
	"os"
)

func FileIfExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if nil != err {
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func ReadFile(filePath string) []byte {
	content, err := ioutil.ReadFile(filePath)
	if nil != err {
		LogError(filePath + " read failed")
	}
	return content
}

func CreateFile(filePath string) {
	f, err := os.Create(filePath)
	if nil != err {
		LogError(filePath + " create failed")
		return
	}
	defer f.Close()
}

func MkDir(dirPath string) {
	err := os.MkdirAll(dirPath, os.ModePerm)
	if nil != err {
		LogError(dirPath + " mkdir failed")
	}
}

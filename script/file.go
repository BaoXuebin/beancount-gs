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

func ReadFile(filePath string) ([]byte, error) {
	content, err := ioutil.ReadFile(filePath)
	if nil != err {
		LogError("Failed to read file (" + filePath + ")")
		return content, err
	}
	LogInfo("Success read file (" + filePath + ")")
	return content, nil
}

func WriteFile(filePath string, content string) error {
	err := ioutil.WriteFile(filePath, []byte(content), os.ModePerm)
	if err != nil {
		LogError("Failed to write file (" + filePath + ")")
		return err
	}
	LogInfo("Success write file (" + filePath + ")")
	return nil
}

func CreateFile(filePath string) error {
	f, err := os.Create(filePath)
	if nil != err {
		LogError(filePath + " create failed")
		return err
	}
	defer f.Close()
	LogInfo("Success create file " + filePath)
	return nil
}

func MkDir(dirPath string) error {
	err := os.MkdirAll(dirPath, os.ModePerm)
	if nil != err {
		LogError("Failed mkdir " + dirPath)
		return err
	}
	LogInfo("Success mkdir " + dirPath)
	return nil
}

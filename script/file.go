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
		LogSystemError("Failed to read file (" + filePath + ")")
		return content, err
	}
	LogSystemInfo("Success read file (" + filePath + ")")
	return content, nil
}

func WriteFile(filePath string, content string) error {
	err := ioutil.WriteFile(filePath, []byte(content), 0777)
	if err != nil {
		LogSystemError("Failed to write file (" + filePath + ")")
		return err
	}
	LogSystemInfo("Success write file (" + filePath + ")")
	return nil
}

func AppendFileInNewLine(filePath string, content string) error {
	content = "\r\n" + content
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err == nil {
		_, err = file.WriteString(content)
		if err != nil {
			LogSystemError("Failed to append file (" + filePath + ")")
			return err
		}
	}
	defer file.Close()
	LogSystemInfo("Success append file (" + filePath + ")")
	return nil
}

func CreateFile(filePath string) error {
	f, err := os.Create(filePath)
	if nil != err {
		LogSystemError(filePath + " create failed")
		return err
	}
	defer f.Close()
	LogSystemInfo("Success create file " + filePath)
	return nil
}

func CopyFile(sourceFilePath string, targetFilePath string) error {
	if !FileIfExist(sourceFilePath) {
		panic("File is not found, " + sourceFilePath)
	}
	if !FileIfExist(targetFilePath) {
		err := CreateFile(targetFilePath)
		if err != nil {
			return err
		}
	}
	bytes, err := ReadFile(sourceFilePath)
	if err != nil {
		return err
	}
	err = WriteFile(targetFilePath, string(bytes))
	if err != nil {
		return err
	}
	return nil
}

func MkDir(dirPath string) error {
	err := os.MkdirAll(dirPath, os.ModePerm)
	if nil != err {
		LogSystemError("Failed mkdir " + dirPath)
		return err
	}
	LogSystemInfo("Success mkdir " + dirPath)
	return nil
}

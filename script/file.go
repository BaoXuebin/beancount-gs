package script

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	err := CreateFileIfNotExist(filePath)
	if err != nil {
		return err
	}
	content = "\r\n" + content
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		LogSystemError("Failed to open file (" + filePath + ")")
		return err
	} else {
		_, err = file.WriteString(content)
		if err != nil {
			LogSystemError("Failed to append file (" + filePath + ")")
			return err
		}
	}
	defer file.Close()
	LogSystemInfo("Success append file (" + filePath + ")")
	return err
}

func DeleteLinesWithText(filePath string, textToDelete string) error {
	// 打开文件以供读写
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建一个缓冲读取器
	scanner := bufio.NewScanner(file)

	// 创建一个字符串切片，用于保存文件的每一行
	var lines []string

	// 逐行读取文件内容
	for scanner.Scan() {
		line := scanner.Text()

		// 检查行是否包含要删除的文本
		if !strings.Contains(line, textToDelete) {
			lines = append(lines, line)
		}
	}

	// 关闭文件
	file.Close()

	// 重新打开文件以供写入
	file, err = os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建一个写入器
	writer := bufio.NewWriter(file)

	// 将修改后的内容写回文件
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	// 刷新缓冲区，确保所有数据被写入文件
	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func CreateFile(filePath string) error {
	if _, e := os.Stat(filePath); os.IsNotExist(e) {
		_ = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		f, err := os.Create(filePath)
		if nil != err {
			LogSystemError(filePath + " create failed")
			return err
		}
		defer f.Close()
		LogSystemInfo("Success create file " + filePath)
	} else {
		LogSystemInfo("File is exist " + filePath)
	}
	return nil
}

func CreateFileIfNotExist(filePath string) error {
	if _, e := os.Stat(filePath); os.IsNotExist(e) {
		_ = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		f, err := os.Create(filePath)
		if nil != err {
			LogSystemError(filePath + " create failed")
			return err
		}
		defer f.Close()
		LogSystemInfo("Success create file " + filePath)
	}
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

func CopyDir(sourceDir string, targetDir string) error {
	dirs, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	err = MkDir(targetDir)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		newSourceDir := filepath.Join(sourceDir, dir.Name())
		newTargetDir := filepath.Join(targetDir, dir.Name())
		if dir.IsDir() {
			err := CopyFile(newSourceDir, newTargetDir)
			if err != nil {
				LogSystemError("Failed to copy dir from [" + newSourceDir + "] to [" + newTargetDir + "]")
				return err
			}
		} else {
			err := CreateFileIfNotExist(newTargetDir)
			if err != nil {
				return err
			}
			err = CopyFile(newSourceDir, newTargetDir)
			if err != nil {
				return err
			}
		}
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

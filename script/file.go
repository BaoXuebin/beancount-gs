package script

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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

// FindConsecutiveMultilineTextInFile 查找文件中连续多行文本片段的开始和结束行号
func FindConsecutiveMultilineTextInFile(filePath string, multilineLines []string) (startLine, endLine int, err error) {
	for i := range multilineLines {
		multilineLines[i] = CleanString(multilineLines[i])
	}

	file, err := os.Open(filePath)
	if err != nil {
		return -1, -1, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	startLine = -1
	endLine = -1
	lineNumber := 0
	matchIndex := 0

	for scanner.Scan() {
		lineNumber++
		// 清理文件中的当前行
		lineText := CleanString(scanner.Text())

		// 检查当前行是否匹配多行文本片段的当前行
		if lineText == multilineLines[matchIndex] {
			if startLine == -1 {
				startLine = lineNumber // 记录起始行号
			}
			matchIndex++
			// 如果所有行都匹配完成，记录结束行号并退出循环
			if matchIndex == len(multilineLines) {
				endLine = lineNumber
				break
			}
		} else {
			// 如果匹配失败，重置匹配索引和起始行号
			matchIndex = 0
			startLine = -1
		}
	}

	if err := scanner.Err(); err != nil {
		return -1, -1, err
	}

	// 如果未找到完整的多行文本片段，则返回 -1
	if startLine == -1 || endLine == -1 {
		return -1, -1, fmt.Errorf("未找到连续的多行文本片段")
	}

	LogSystemInfo("Success find content in file " + filePath + " line range:  " + string(rune(startLine)) + "," + string(rune(endLine)))
	return startLine, endLine, nil
}

// CleanString 去除字符串中的首尾空白和中间的所有空格字符
func CleanString(str string) string {
	if IsComment(str) {
		return ""
	}
	result := getAccountWithNumber(str)
	// 去除 " ", ";", "\r"
	result = strings.ReplaceAll(result, ",", "")
	result = strings.ReplaceAll(result, " ", "")
	// 过滤空白的商户信息 ““
	result = strings.ReplaceAll(result, "\"\"", "")
	result = strings.ReplaceAll(result, ";", "")
	result = strings.ReplaceAll(result, "\r", "")
	// 清楚汇率转换

	return result
}

// 正则提取：
// Assets:Flow:Cash:现金 -20.00 USD {xxx CNY, 2025-01-01} -> Assets:Flow:Cash:现金 -20.00 USD
func getAccountWithNumber(str string) string {
	// 定义正则表达式模式
	pattern := `^[^\{]+`
	// 编译正则表达式
	re := regexp.MustCompile(pattern)
	// 使用正则提取匹配的部分
	return re.FindString(str)
}

func IsComment(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	if strings.HasPrefix(trimmed, ";") {
		return true
	}
	return false
}

// 删除指定行范围的内容
func RemoveLines(filePath string, startLineNo, endLineNo int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取文件的每一行
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 检查行号的有效性
	if startLineNo < 1 || endLineNo > len(lines) || startLineNo > endLineNo {
		return nil, fmt.Errorf("行号范围无效")
	}

	// 删除从 startLineNo 到 endLineNo 的行（下标从 0 开始）
	modifiedLines := append(lines[:startLineNo-1], lines[endLineNo:]...)
	return modifiedLines, nil
}

// 在指定行号插入多行文本
func InsertLines(lines []string, startLineNo int, newLines []string) ([]string, error) {
	// 检查插入位置的有效性
	if startLineNo < 1 || startLineNo > len(lines)+1 {
		return nil, fmt.Errorf("插入行号无效")
	}
	// 在指定位置插入新的内容
	modifiedLines := append(lines[:startLineNo-1], append(newLines, lines[startLineNo-1:]...)...)
	return modifiedLines, nil
}

// 写回文件
func WriteToFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 将修改后的内容写回文件
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	LogSystemInfo("Success write content in file " + filePath)
	return writer.Flush()
}

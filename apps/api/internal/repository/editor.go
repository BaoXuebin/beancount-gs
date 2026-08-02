package repository

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var costPartRe = regexp.MustCompile(`^[^{]+`)

// ReplaceLinesByNormalizedMatch 在文件中定位 oldLines 的连续匹配块（忽略空白、逗号、
// 空引号、注释等格式差异），替换为 newLines；newLines 为空表示删除该块。
func ReplaceLinesByNormalizedMatch(filePath string, oldLines, newLines []string) error {
	fileLines, err := readLines(filePath)
	if err != nil {
		return err
	}
	pattern := make([]string, len(oldLines))
	for i, l := range oldLines {
		pattern[i] = normalizeBeanLine(l)
	}
	start, end := findConsecutiveMatch(fileLines, pattern)
	if start == -1 {
		return fmt.Errorf("transaction block not found in %s", filePath)
	}
	result := make([]string, 0, len(fileLines)-len(pattern)+len(newLines))
	result = append(result, fileLines[:start]...)
	result = append(result, newLines...)
	result = append(result, fileLines[end:]...)
	return writeLines(filePath, result)
}

func findConsecutiveMatch(fileLines, pattern []string) (int, int) {
	if len(pattern) == 0 {
		return -1, -1
	}
	for i := 0; i+len(pattern) <= len(fileLines); i++ {
		matched := true
		for j := range pattern {
			if normalizeBeanLine(fileLines[i+j]) != pattern[j] {
				matched = false
				break
			}
		}
		if matched {
			return i, i + len(pattern)
		}
	}
	return -1, -1
}

// normalizeBeanLine 归一化 bean 行用于模糊匹配：去注释、空格、逗号、空引号与成本花括号部分。
func normalizeBeanLine(line string) string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, ";") {
		return ""
	}
	line = costPartRe.FindString(line)
	line = strings.ReplaceAll(line, ",", "")
	line = strings.ReplaceAll(line, " ", "")
	line = strings.ReplaceAll(line, "\t", "")
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.ReplaceAll(line, `""`, "")
	return line
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLines(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for i, l := range lines {
		if i > 0 {
			if _, err := w.WriteString("\n"); err != nil {
				return err
			}
		}
		if _, err := w.WriteString(l); err != nil {
			return err
		}
	}
	return w.Flush()
}

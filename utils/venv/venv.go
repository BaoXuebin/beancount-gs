/*
 * @Author: liangzai450
 * @Date: 2025-09-17 20:50:11
 * @LastEditors: liangzai liangzai450@qq.com
 * @LastEditTime: 2025-09-18 08:15:42
 * @FilePath: \\cnb-beancount\\utils\\venv\\venv.go
 * @Description:
 * Copyright (c) 2025 by ${git_name_email}, All Rights Reserved.
 * ==============================================
 */

package venv

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// VenvExecutor 虚拟环境执行器
type VenvExecutor struct {
	venvPath string
}

// NewVenvExecutor 创建新的虚拟环境执行器
func NewVenvExecutor(venvPath string) *VenvExecutor {
	return &VenvExecutor{venvPath: venvPath}
}

// GetCommandPath 获取虚拟环境中命令的完整路径
func (v *VenvExecutor) GetCommandPath(command string) (string, error) {
	var cmdPath string

	if runtime.GOOS == "windows" {
		cmdPath = filepath.Join(v.venvPath, "Scripts", command+".exe")
	} else {
		cmdPath = filepath.Join(v.venvPath, "bin", command)
	}

	if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
		return "", fmt.Errorf("command %s not found at: %s", command, cmdPath)
	}

	return cmdPath, nil
}

// Execute 执行虚拟环境中的命令
func (v *VenvExecutor) Execute(command string, args ...string) error {
	cmdPath, err := v.GetCommandPath(command)
	if err != nil {
		return err
	}

	cmd := exec.Command(cmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ExecuteWithOutput 执行命令并返回输出
func (v *VenvExecutor) ExecuteWithOutput(command string, args ...string) ([]byte, []byte, error) {
	cmdPath, err := v.GetCommandPath(command)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(cmdPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// BeanQueryStdout 执行 bean-query 命令，只返回纯净的标准输出
func (v *VenvExecutor) BeanQueryStdout(beancountFile, query string) ([]byte, error) {
	stdout, stderr, err := v.ExecuteWithOutput("bean-query", beancountFile, query)
	if err != nil {
		// 你可以选择记录 stderr 或根据错误类型处理
		fmt.Printf("Bean-query stderr: %s", string(stderr))
		return nil, err
	}
	return stdout, nil
}

// BeanQuery 执行 bean-query 命令
// 保持原有的 BeanQuery 函数（如果需要兼容旧代码）
func (v *VenvExecutor) BeanQuery(beancountFile, query string) ([]byte, error) {
	return v.BeanQueryStdout(beancountFile, query)
}

// BeanCheck 执行 bean-check 语法检查
func (v *VenvExecutor) BeanCheck(beancountFile string) error {
	_, err := v.BeanQueryStdout("bean-check", beancountFile)
	return err
}

// Fava 启动 fava 服务器
func (v *VenvExecutor) Fava(beancountFile string, port int) error {
	return v.Execute("fava", beancountFile, "--port", fmt.Sprintf("%d", port))
}

// CheckVenvExists 检查虚拟环境是否存在
func CheckVenvExists(venvPath string) bool {
	if runtime.GOOS == "windows" {
		return dirExists(filepath.Join(venvPath, "Scripts"))
	}
	return dirExists(filepath.Join(venvPath, "bin"))
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

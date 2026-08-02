package repository

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZip 将 zip 安全解压到 dest，返回写入的相对路径列表（/ 分隔）。
// 拒绝绝对路径、.. 穿越与符号链接，防止 zip-slip。
func ExtractZip(r io.ReaderAt, size int64, dest string) ([]string, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("打开 zip: %w", err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	destClean := filepath.Clean(dest)
	files := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		rel := filepath.ToSlash(filepath.Clean(f.Name))
		if rel == "." || rel == "" {
			continue
		}
		if strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "/") || filepath.IsAbs(f.Name) {
			return nil, fmt.Errorf("zip 中存在非法路径: %s", f.Name)
		}
		target := filepath.Join(destClean, filepath.FromSlash(rel))
		if target != destClean && !strings.HasPrefix(target, destClean+string(os.PathSeparator)) {
			return nil, fmt.Errorf("zip 路径越界: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("zip 中不允许符号链接: %s", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return nil, err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return nil, err
		}
		rc.Close()
		if err := out.Close(); err != nil {
			return nil, err
		}
		files = append(files, rel)
	}
	return files, nil
}

// SnapshotTree 将 srcDir 下所有文件镜像复制到 backupDir（跳过 bak/ 目录），返回复制的文件数。
func SnapshotTree(srcDir, backupDir string) (int, error) {
	count := 0
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		if d.IsDir() && d.Name() == "bak" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(backupDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

// CopyTree 将 srcDir 下所有文件复制到 dstDir（覆盖同名文件），返回复制的文件数。
func CopyTree(srcDir, dstDir string) (int, error) {
	count := 0
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

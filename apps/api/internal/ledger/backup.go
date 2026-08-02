package ledger

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/beancount-gs/api/internal/db"
	"github.com/beancount-gs/api/internal/repository"
)

// ImportBackup 将已通过校验的备份目录（srcDir）内容导入账本：
// 先 CAS 递增修订号，再把现有文件快照到 bak/，最后覆盖写入备份文件并记录审计日志。
func (s *Service) ImportBackup(ctx context.Context, l db.Ledger, srcDir string, expectedRevision int64, actor Actor) (int64, error) {
	unlock := s.lockLedger(l.ID)
	defer unlock()

	newRev, err := s.Store.CompareAndBumpRevision(ctx, l.ID, expectedRevision)
	if err != nil {
		return 0, err
	}
	bakDir := filepath.Join(l.DataPath, "bak", fmt.Sprintf("%d", time.Now().Unix()))
	if _, err := repository.SnapshotTree(l.DataPath, bakDir); err != nil {
		return 0, fmt.Errorf("写前快照失败: %w", err)
	}
	if _, err := repository.CopyTree(srcDir, l.DataPath); err != nil {
		return 0, fmt.Errorf("写入备份文件失败: %w", err)
	}
	if err := s.audit(ctx, l, actor, "import_backup", "revision:"+strconv.FormatInt(newRev, 10)); err != nil {
		return 0, err
	}
	return newRev, nil
}

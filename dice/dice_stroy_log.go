package dice

import (
	"errors"
	"os"
	"path/filepath"
)

type StoryLogBackup struct {
	Name     string `json:"name"`
	FileSize int64  `json:"fileSize"`
}

func StoryLogBackupList(d *Dice) ([]StoryLogBackup, error) {
	dirpath := filepath.Join(d.BaseConfig.DataDir, "log-exports")
	backups, err := os.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	var res []StoryLogBackup
	for _, backup := range backups {
		info, err := backup.Info()
		if err != nil {
			return nil, err
		}
		res = append(res, StoryLogBackup{
			Name:     backup.Name(),
			FileSize: info.Size(),
		})
	}
	return res, nil
}

func StoryLogBackupDownloadPath(d *Dice, backup string) (string, error) {
	if backup == "" {
		return "", errors.New("empty name")
	}
	dirpath := filepath.Join(d.BaseConfig.DataDir, "log-exports")
	backupPath := filepath.Join(dirpath, backup)
	_, err := os.Stat(backupPath)
	if err != nil {
		return "", err
	}
	return backupPath, nil
}

func StoryLogBackupBatchDelete(d *Dice, backups []string) []string {
	if len(backups) == 0 {
		return nil
	}
	dirpath := filepath.Join(d.BaseConfig.DataDir, "log-exports")
	var fails []string
	for _, backupName := range backups {
		if backupName == "" {
			fails = append(fails, "empty name")
		}
		backupPath := filepath.Join(dirpath, backupName)
		err := os.Remove(backupPath)
		if err != nil {
			fails = append(fails, backupName)
		}
	}
	return fails
}

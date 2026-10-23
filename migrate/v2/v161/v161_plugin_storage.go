package v161

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"github.com/tidwall/buntdb"

	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/dboperator/gormkv"
	"sealdice-core/utils/dboperator/gormkv/codec"
	upgrade "sealdice-core/utils/upgrader"
)

func V161PluginStorageMigrate(dbOperator operator.DatabaseOperator, logf func(string)) error {
	readDB := dbOperator.GetPluginDB(constant.READ)
	writeDB := dbOperator.GetPluginDB(constant.WRITE)
	if writeDB == nil {
		return errors.New("plugin database is unavailable")
	}
	if readDB == nil {
		readDB = writeDB
	}
	backend, err := gormkv.NewBackend(gormkv.BackendOptions{
		ReadDB:  readDB,
		WriteDB: writeDB,
		Compression: codec.Options{
			Enabled: true,
			Level:   codec.DefaultZstdLevel,
			MinSize: codec.DefaultMinSize,
		},
	})
	if err != nil {
		return fmt.Errorf("initialize plugin KV backend: %w", err)
	}
	if migrateErr := backend.AutoMigrate(); migrateErr != nil {
		return fmt.Errorf("initialize plugin KV table: %w", migrateErr)
	}

	extensionsDir := filepath.Join("data", "default", "extensions")
	entries, err := os.ReadDir(extensionsDir)
	if errors.Is(err, os.ErrNotExist) {
		logf("[INFO] V161 插件存储迁移：未发现旧插件存储")
		return nil
	}
	if err != nil {
		return fmt.Errorf("scan plugin storage directory %s: %w", extensionsDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginName := entry.Name()
		sourcePath := filepath.Join(extensionsDir, pluginName, "storage.db")
		if _, err := os.Stat(sourcePath); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("stat legacy storage %s: %w", sourcePath, err)
		}

		repo, err := backend.Namespace(pluginName)
		if err != nil {
			return fmt.Errorf("create repository for %s: %w", pluginName, err)
		}
		rowCount, err := migratePluginStorageFile(repo, sourcePath)
		if err != nil {
			return fmt.Errorf("migrate plugin storage for %s: %w", pluginName, err)
		}
		logf(fmt.Sprintf("[INFO] V161 插件存储迁移完成: %s (%d 条)", pluginName, rowCount))
	}
	return nil
}

func migratePluginStorageFile(repo *gormkv.Repository, sourcePath string) (int, error) {
	legacy, err := buntdb.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer legacy.Close()

	var rowCount int
	var migrationErr error
	err = legacy.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("*", func(key, value string) bool {
			migrationErr = repo.Put(context.Background(), key, []byte(value), store.PutOptions{
				ValueKind: store.ValueKindText,
			})
			if migrationErr == nil {
				rowCount++
			}
			return migrationErr == nil
		})
	})
	if err != nil {
		return rowCount, err
	}
	return rowCount, migrationErr
}

var V161PluginStorageMigration = upgrade.Upgrade{
	ID: "013_V161PluginStorageMigration",
	Description: `
# 升级说明
初始化共享插件 KV 表，并将 extensions 下各插件的 BuntDB storage.db 迁移到插件数据库。
`,
	Apply: func(logf func(string), dbOperator operator.DatabaseOperator) error {
		logf("[INFO] V161 插件存储迁移开始")
		if err := V161PluginStorageMigrate(dbOperator, logf); err != nil {
			return err
		}
		logf("[INFO] V161 插件存储迁移处置完毕")
		return nil
	},
}

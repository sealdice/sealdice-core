package migrate

import (
	"fmt"
	"os"
)

func TryMigrateToV12() {
	_, err := os.Stat("./data/default/data.bdb")
	if err != nil {
		return
	}

	fmt.Println("检测到旧数据库存在，试图进行转换")
	_ = ConvertServe()
	_ = ConvertLogs()
	_ = os.Remove("./data/default/data.bdb")
	fmt.Println("V1.2 版本数据库升级完成")
}

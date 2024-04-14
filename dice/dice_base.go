//nolint:gosec
package dice

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

func Base64ToImageFunc(logger *zap.SugaredLogger) func(string) string {
	return func(b64 string) string {
		// use logger here
		// 解码 Base64 值
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			logger.Errorf("不合法的base64值：%s", b64)
			return "" // 出现错误，拒绝向下执行
		}
		// 计算 MD5 哈希值作为文件名
		hash := md5.Sum(data)
		filename := fmt.Sprintf("%x", hash)
		tempDir := os.TempDir()
		// 构建文件路径
		imageurlPath := filepath.Join(tempDir, filename)
		imageurlPath = filepath.ToSlash(imageurlPath)
		// 将数据写入文件
		fi, err := os.OpenFile(imageurlPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
		if err != nil {
			logger.Errorf("创建文件出错%s", err.Error())
			return ""
		}
		defer fi.Close()
		_, err = fi.Write(data)
		if err != nil {
			logger.Errorf("写入文件出错%s", err.Error())
			return ""
		}
		logger.Info("File saved to:", imageurlPath)
		return "file://" + imageurlPath
	}
}
func RegisterBuiltinState(self *Dice) {
	cmdState := &CmdItemInfo{
		Name:      "state",
		ShortHelp: ".state",
		Help:      "实时获取系统运行状态",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			f := func() CmdExecuteResult {
				v, _ := mem.VirtualMemory()
				m := "总内存：" + strconv.FormatFloat(float64(v.Total)/(1024*1024*1024), 'f', 4, 64) + "GB ，"
				m += "空闲中：" + strconv.FormatFloat(float64(v.Free)/(1024*1024*1024), 'f', 4, 64) + "GB ，"
				m += "使用率：" + strconv.FormatFloat((float64(v.Used)/float64(v.Total))*100, 'f', 4, 64) + "%"
				c, _ := cpu.Info()
				u := "型号：" + c[0].ModelName + " ,"
				u += "核心数：" + strconv.Itoa(int(c[0].Cores))

				diskPart, err := disk.Partitions(false)
				if err != nil {
					fmt.Println("获取磁盘使用情况时出错:", err)
				}
				var d string
				for _, dp := range diskPart {
					d += "分区：" + dp.Mountpoint + "， "
					d = strings.ReplaceAll(d, ":", "")
					d += "文件系统：" + dp.Fstype + "， "
					diskUsed, _ := disk.Usage(dp.Mountpoint)
					d += "分区总大小：" + strconv.Itoa(int(diskUsed.Total/(1024*1024*1024))) + "GB， "
					d += "分区使用率：" + strconv.FormatFloat(diskUsed.UsedPercent, 'f', 4, 64) + "%, "
					d += "分区inode使用率：" + strconv.FormatFloat(diskUsed.InodesUsedPercent, 'f', 4, 64) + "%\n"
				}
				reply := "操作系统：" + runtime.GOOS + "\n系统架构：" + runtime.GOARCH + "\n内存相关：\n" + m + "\nCPU相关：\n" + u + "\n硬盘相关： \n" + d
				ReplyToSender(ctx, msg, reply)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			switch ctx.PrivilegeLevel {
			case 100:
				f()
			default:
				ReplyToSender(ctx, msg, "你无权这样做")
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	theExt := &ExtInfo{
		Name:       "state", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "实时获取系统运行状态",
		Author:     "炽热",
		AutoActive: true, // 是否自动开启
		Official:   true,
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		OnLoad:      func() {},
		CmdMap: CmdMapCls{
			"state": cmdState,
			"状态":    cmdState,
		},
	}
	self.RegisterExtension(theExt)
}

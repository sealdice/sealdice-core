package oschecker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	log "sealdice-core/utils/kratos"
)

func OldVersionCheck() (bool, string) {
	version := getGlibcVersion()
	if version == 0 {
		return true, fmt.Sprintf("%v", version)
	}
	if getGlibcVersion() <= 2.17 {
		return false, fmt.Sprintf("%v", version)
	}
	return true, fmt.Sprintf("%v", version)
}

// GetGlibcVersion 获取glibc版本号，默认设置为CentOS7 的 2.17版本，这个版本默认会认为是低版本
func getGlibcVersion() float64 {
	shell, err := execShell("ldd --version | awk 'NR==1{print $NF}'")
	if err != nil {
		log.Debugf("获取glibc版本号失败: %s", err.Error())
		return 0
	}
	version := clearStr(shell)
	f, err := strconv.ParseFloat(version, 64)
	if err != nil {
		log.Debugf("转换glibc版本号失败:version-%s err-%s", version, err.Error())
		return 0
	}
	return f
}

// ClearStr 清理字符串中的空格、换行符、制表符
func clearStr(str string) string {
	return string(clearBytes([]byte(str)))
}

func clearBytes(str []byte) []byte {
	return bytes.ReplaceAll(bytes.ReplaceAll(bytes.ReplaceAll(bytes.ReplaceAll(str, []byte(" "), []byte("")), []byte("\r"), []byte("")), []byte("\t"), []byte("")), []byte("\n"), []byte(""))
}

func execShell(cmdStr string) (string, error) {
	if !strings.HasPrefix(cmdStr, "sudo") {
		cmdStr = "sudo " + cmdStr
	}
	cmd := exec.Command("bash", "-c", cmdStr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errMsg := "Shell: " + cmdStr + ";"
		if len(stderr.String()) != 0 {
			errMsg = fmt.Sprintf("stderr: %s", stderr.String())
		}
		if len(stdout.String()) != 0 {
			if len(errMsg) != 0 {
				errMsg = fmt.Sprintf("%s; stdout: %s", errMsg, stdout.String())
			} else {
				errMsg = fmt.Sprintf("stdout: %s", stdout.String())
			}
		}
		fmt.Printf("ExecShell-errMsg-> %s\n", errMsg)
		return errMsg, fmt.Errorf(errMsg)
	}
	return stdout.String(), nil
}

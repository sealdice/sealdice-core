package oschecker

// copied from https://github.com/jfjallid/go-secdump/blob/e307524e114f9abb39e2cd2b13ae421aae02d2de/utils.go with some changes

import (
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"

	log "sealdice-core/utils/kratos"
)

const (
	WIN_UNKNOWN = iota
	WINXP
	WIN_SERVER_2003
	WIN_VISTA
	WIN_SERVER_2008
	WIN7
	WIN_SERVER_2008_R2
	WIN8
	WIN_SERVER_2012
	WIN81
	WIN_SERVER_2012_R2
	WIN10
	WIN_SERVER_2016
	WIN_SERVER_2019
	WIN_SERVER_2022
	WIN11
)

var osNameMap = map[byte]string{
	WIN_UNKNOWN:        "Windows Unknown",
	WINXP:              "Windows XP",
	WIN_VISTA:          "Windows Vista",
	WIN7:               "Windows 7",
	WIN8:               "Windows 8",
	WIN81:              "Windows 8.1",
	WIN10:              "Windows 10",
	WIN11:              "Windows 11",
	WIN_SERVER_2003:    "Windows Server 2003",
	WIN_SERVER_2008:    "Windows Server 2008",
	WIN_SERVER_2008_R2: "Windows Server 2008 R2",
	WIN_SERVER_2012:    "Windows Server 2012",
	WIN_SERVER_2012_R2: "Windows Server 2012 R2",
	WIN_SERVER_2016:    "Windows Server 2016",
	WIN_SERVER_2019:    "Windows Server 2019",
	WIN_SERVER_2022:    "Windows Server 2022",
}

// OldVersionCheck 只获取最低版本
func OldVersionCheck() (bool, string) {
	build, f, b, err := getOSVersionBuild()
	if err != nil {
		// 不知道的版本，就认为是支持的
		return true, osNameMap[WIN_UNKNOWN]
	}
	os := GetOSVersion(build, f, b)
	if (WINXP <= os) && (os <= WIN10) {
		return true, osNameMap[os]
	} else {
		return false, osNameMap[os]
	}
}

func GetOSVersion(currentBuild int, currentVersion float64, server bool) (os byte) {
	currentVersionStr := strconv.FormatFloat(currentVersion, 'f', 1, 64)
	if server {
		switch {
		case currentBuild >= 3790 && currentBuild < 6001:
			os = WIN_SERVER_2003
		case currentBuild >= 6001 && currentBuild < 7601:
			os = WIN_SERVER_2008
		case currentBuild >= 7601 && currentBuild < 9200:
			os = WIN_SERVER_2008_R2
		case currentBuild >= 9200 && currentBuild < 9600:
			os = WIN_SERVER_2012
		case currentBuild >= 9200 && currentBuild < 14393:
			os = WIN_SERVER_2012_R2
		case currentBuild >= 14393 && currentBuild < 17763:
			os = WIN_SERVER_2016
		case currentBuild >= 17763 && currentBuild < 20348:
			os = WIN_SERVER_2019
		case currentBuild >= 20348:
			os = WIN_SERVER_2022
		default:
			log.Debugf("Unknown server version of Windows with CurrentBuild %d and CurrentVersion %f\n", currentBuild, currentVersion)
			os = WIN_UNKNOWN
		}
	} else {
		switch currentVersionStr {
		case "5.1":
			os = WINXP
		case "6.0":
			// Windows Vista but it shares CurrentVersion and CurrentBuild with Windows Server 2008
			os = WIN_VISTA
		case "6.1":
			// Windows 7 but it shares CurrentVersion and CurrentBuild with Windows Server 2008 R2
			os = WIN7
		case "6.2":
			// Windows 8 but it shares CurrentVersion and CurrentBuild with Windows Server 2012
			os = WIN8
		case "6.3":
			// Windows 8.1 but it shares CurrentVersion and CurrentBuild with Windows Server 2012 R2
			os = WIN81
		case "10.0":
			if currentBuild < 22000 {
				os = WIN10
			} else {
				os = WIN11
			}
		default:
			log.Debugf("Unknown version of Windows with CurrentBuild %d and CurrentVersion %f\n", currentBuild, currentVersion)
			os = WIN_UNKNOWN
		}
	}

	log.Debugf("OS Version: %s\n", osNameMap[os])
	return
}

func getOSVersionBuild() (build int, version float64, server bool, err error) {
	hSubKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		log.Errorf("Failed to open registry key CurrentVersion with error: %v\n", err)
		return
	}
	defer func(hSubKey registry.Key) {
		err = hSubKey.Close()
		if err != nil {
			log.Fatalf("Failed to close hSubkey with error: %v\n", err)
		}
	}(hSubKey)

	buildStr, _, err := hSubKey.GetStringValue("CurrentBuild")
	if err != nil {
		log.Error(err)
		return
	}
	build, err = strconv.Atoi(buildStr)
	if err != nil {
		log.Error(err)
		return
	}
	versionStr, _, err := hSubKey.GetStringValue("CurrentVersion")
	if err != nil {
		log.Error(err)
		return
	}

	version, err = strconv.ParseFloat(versionStr, 32)
	if err != nil {
		log.Errorf("Failed to get CurrentVersion with error: %v\n", err)
		return
	}
	majorVersionStr, _, err := hSubKey.GetIntegerValue("CurrentMajorVersionNumber")
	if err != nil {
		return
	}
	// 据说，当前Win11和Win10的大版本号还相同，没有Win11，难以测试
	if majorVersionStr == 10 {
		version = 10.0
	}

	hSubKey, err = registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\ProductOptions`, registry.QUERY_VALUE)
	if err != nil {
		log.Errorf("Failed to open registry key ProductOptions with error: %v\n", err)
		return
	}

	serverFlag, _, err := hSubKey.GetStringValue("ProductType")
	if err != nil {
		log.Error(err)
		return
	}

	if strings.Compare(serverFlag, "ServerNT") == 0 {
		server = true
	}

	return
}

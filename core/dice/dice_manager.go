package dice

import (
	"fmt"
	"github.com/dop251/goja_nodejs/require"
	"github.com/fy0/lockfree"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

type VersionInfo struct {
	VersionLatest           string `yaml:"versionLatest" json:"versionLatest"`
	VersionLatestDetail     string `yaml:"versionLatestDetail" json:"versionLatestDetail"`
	VersionLatestCode       int64  `yaml:"versionLatestCode" json:"versionLatestCode"`
	VersionLatestNote       string `yaml:"versionLatestNote" json:"versionLatestNote"`
	MinUpdateSupportVersion int64  `yaml:"minUpdateSupportVersion" json:"minUpdateSupportVersion"`
	NewVersionUrlPrefix     string `yaml:"newVersionUrlPrefix" json:"newVersionUrlPrefix"`
}

type GroupNameCacheItem struct {
	Name string
	time int64
}

type DiceManager struct {
	Dice                 []*Dice
	ServeAddress         string
	Help                 *HelpManager
	IsHelpReloading      bool
	UseDictForTokenizer  bool
	HelpDocEngineType    int
	progressExitGroupWin ProcessExitGroup

	IsNamesReloading bool
	NamesGenerator   *NamesGenerator
	NamesInfo        map[string]map[string][]string

	UIPasswordHash string
	UIPasswordSalt string
	AccessTokens   map[string]bool
	IsReady        bool

	AutoBackupEnable bool
	AutoBackupTime   string

	AppBootTime      int64
	AppVersionCode   int64
	AppVersionOnline *VersionInfo

	UpdateRequestChan      chan int
	UpdateDownloadedChan   chan string
	RebootRequestChan      chan int
	UpdateCheckRequestChan chan int

	GroupNameCache lockfree.HashMap // 群名缓存，全局共享, key string value *GroupNameCacheItem
	UserNameCache  lockfree.HashMap // 用户缓存，全局共享, key string value *GroupNameCacheItem
	UserIdCache    lockfree.HashMap // 用户id缓存 key username (string) value int64 目前仅Telegram adapter使用

	Cron          *cron.Cron
	ServiceName   string
	JustForTest   bool
	backupEntryId cron.EntryID
	JsRegistry    *require.Registry
}

type DiceConfigs struct {
	DiceConfigs       []DiceConfig `yaml:"diceConfigs"`
	ServeAddress      string       `yaml:"serveAddress"`
	WebUIAddress      string       `yaml:"webUIAddress"`
	HelpDocEngineType int          `yaml:"helpDocEngineType"`

	UIPasswordSalt string   `yaml:"UIPasswordFrontendSalt"`
	UIPasswordHash string   `yaml:"uiPasswordHash"`
	AccessTokens   []string `yaml:"accessTokens"`

	AutoBackupEnable bool   `yaml:"autoBackupEnable"`
	AutoBackupTime   string `yaml:"autoBackupTime"`
	ServiceName      string `yaml:"serviceName"`

	ConfigVersion int `yaml:"configVersion"`
}

func (dm *DiceManager) InitHelp() {
	dm.IsHelpReloading = true
	_ = os.MkdirAll("./data/helpdoc", 0755)
	dm.Help = new(HelpManager)
	dm.Help.Parent = dm
	dm.Help.EngineType = dm.HelpDocEngineType
	dm.Help.Load()
	dm.IsHelpReloading = false
}

// LoadDice 初始化函数
func (dm *DiceManager) LoadDice() {
	dm.AppVersionCode = VERSION_CODE
	dm.AppBootTime = time.Now().Unix()
	dm.GroupNameCache = lockfree.NewHashMap()
	dm.UserNameCache = lockfree.NewHashMap()
	dm.UserIdCache = lockfree.NewHashMap()

	_ = os.MkdirAll("./backups", 0755)
	_ = os.MkdirAll("./data/images", 0755)
	_ = os.MkdirAll("./data/decks", 0755)
	_ = os.MkdirAll("./data/names", 0755)
	_ = os.WriteFile("./data/images/sealdice.png", ICON_PNG, 0644)

	// this can be shared by multiple runtimes
	dm.JsRegistry = new(require.Registry)

	dm.Cron = cron.New()
	dm.Cron.Start()

	dm.AccessTokens = map[string]bool{}
	if dm.UIPasswordSalt == "" {
		// 旧版本升级，或新用户
		dm.UIPasswordSalt = RandStringBytesMaskImprSrcSB2(32)
	}
	dm.AutoBackupEnable = true
	dm.AutoBackupTime = "@every 12h" // 每12小时一次

	data, err := os.ReadFile("./data/dice.yaml")
	if err != nil {
		// 注意！！！！ 这里会退出，所以下面的都可能不执行！
		return
	}

	var dc DiceConfigs
	err = yaml.Unmarshal(data, &dc)
	if err != nil {
		fmt.Println("读取 data/dice.yaml 发生错误: 配置文件格式不正确")
		panic(err)
	}

	if dc.UIPasswordSalt == "" {
		// 旧版本升级
		dc.UIPasswordSalt = dm.UIPasswordSalt
	}

	dm.ServeAddress = dc.ServeAddress
	dm.HelpDocEngineType = dc.HelpDocEngineType
	dm.UIPasswordHash = dc.UIPasswordHash
	dm.UIPasswordSalt = dc.UIPasswordSalt
	dm.AutoBackupTime = dc.AutoBackupTime
	dm.AutoBackupEnable = dc.AutoBackupEnable

	if dc.AutoBackupTime == "" {
		// 从旧版升级
		dm.AutoBackupEnable = true
		dm.AutoBackupTime = "@every 12h" // 每12小时一次
	}

	for _, i := range dc.AccessTokens {
		dm.AccessTokens[i] = true
	}

	for _, i := range dc.DiceConfigs {
		newDice := new(Dice)
		newDice.BaseConfig = i
		dm.Dice = append(dm.Dice, newDice)
	}
}

func (dm *DiceManager) Save() {
	var dc DiceConfigs
	dc.ServeAddress = dm.ServeAddress
	dc.HelpDocEngineType = dm.HelpDocEngineType
	dc.UIPasswordSalt = dm.UIPasswordSalt
	dc.UIPasswordHash = dm.UIPasswordHash
	dc.AccessTokens = []string{}
	dc.AutoBackupTime = dm.AutoBackupTime
	dc.AutoBackupEnable = dm.AutoBackupEnable
	dc.ServiceName = dm.ServiceName
	dc.ConfigVersion = 9914

	for k, _ := range dm.AccessTokens {
		dc.AccessTokens = append(dc.AccessTokens, k)
	}

	for _, i := range dm.Dice {
		dc.DiceConfigs = append(dc.DiceConfigs, i.BaseConfig)
	}

	data, err := yaml.Marshal(dc)
	if err == nil {
		_ = os.WriteFile("./data/dice.yaml", data, 0644)
	}
}

func (dm *DiceManager) InitDice() {
	dm.UpdateRequestChan = make(chan int, 1)
	dm.RebootRequestChan = make(chan int, 1)
	dm.UpdateCheckRequestChan = make(chan int, 1)
	dm.UpdateDownloadedChan = make(chan string, 1)

	dm.LoadNames()

	g, err := NewProcessExitGroup()
	if err != nil {
		fmt.Println("进程组创建失败，若进程崩溃，gocqhttp进程可能需要手动结束。")
	} else {
		dm.progressExitGroupWin = g
	}

	for _, i := range dm.Dice {
		i.Parent = dm
		i.Init()
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("帮助文档加载失败。可能是由于退出程序过快，帮助文档还未加载完成所致", r)
				if dm.Help != nil {
					fmt.Println("帮助文件加载失败:", dm.Help.LoadingFn)
				}
			}
		}()

		// 加载帮助
		dm.InitHelp()
		if len(dm.Dice) >= 1 {
			dm.AddHelpWithDice(dm.Dice[0])
		}
	}()

	dm.ResetAutoBackup()
}

func (dm *DiceManager) ResetAutoBackup() {
	if dm.backupEntryId != 0 {
		dm.Cron.Remove(dm.backupEntryId)
		dm.backupEntryId = 0
	}
	if dm.AutoBackupEnable {
		var err error
		dm.backupEntryId, err = dm.Cron.AddFunc(dm.AutoBackupTime, func() {
			err := dm.BackupAuto()
			if err != nil {
				fmt.Println("自动备份失败: ", err.Error())
			}
		})
		if err != nil {
			if len(dm.Dice) > 0 {
				dm.Dice[0].Logger.Errorf("设定的自动备份间隔有误: %v", err.Error())
			}
			return
		}
	}
}

func (dm *DiceManager) TryCreateDefault() {
	if dm.ServeAddress == "" {
		dm.ServeAddress = "0.0.0.0:3211"
	}

	if len(dm.Dice) == 0 {
		defaultDice := new(Dice)
		defaultDice.BaseConfig.Name = "default"
		defaultDice.BaseConfig.IsLogPrint = true
		defaultDice.MessageDelayRangeStart = 0.4
		defaultDice.MessageDelayRangeEnd = 0.9
		defaultDice.MarkModified()
		dm.Dice = append(dm.Dice, defaultDice)
	}
}

func (dm *DiceManager) LoadNames() {
	dm.IsNamesReloading = true
	dm.NamesGenerator = &NamesGenerator{}
	dm.NamesGenerator.Load()
	dm.IsNamesReloading = false
}

func (dm *DiceManager) TryGetGroupName(id string) string {
	item, exists := dm.GroupNameCache.Get(id)
	if exists {
		realItem, ok := item.(*GroupNameCacheItem)
		if ok {
			return realItem.Name
		}
	}
	return "%未知群名%"
}

func (dm *DiceManager) TryGetUserName(id string) string {
	item, exists := dm.UserNameCache.Get(id)
	if exists {
		realItem, ok := item.(*GroupNameCacheItem)
		if ok {
			return realItem.Name
		}
	}
	return "%未知用户%"
}

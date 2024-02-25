package dice

import (
	"fmt"
	"os"
	"time"

	"github.com/dop251/goja_nodejs/require"
	"github.com/fy0/lockfree"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type VersionInfo struct {
	VersionLatest           string `yaml:"versionLatest" json:"versionLatest"`
	VersionLatestDetail     string `yaml:"versionLatestDetail" json:"versionLatestDetail"`
	VersionLatestCode       int64  `yaml:"versionLatestCode" json:"versionLatestCode"`
	VersionLatestNote       string `yaml:"versionLatestNote" json:"versionLatestNote"`
	MinUpdateSupportVersion int64  `yaml:"minUpdateSupportVersion" json:"minUpdateSupportVersion"`
	NewVersionURLPrefix     string `yaml:"newVersionUrlPrefix" json:"newVersionUrlPrefix"`
	UpdaterURLPrefix        string `yaml:"updaterUrlPrefix" json:"updaterUrlPrefix"`
}

type GroupNameCacheItem struct {
	Name string
	time int64
}

type DiceManager struct { //nolint:revive
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
	backupEntryID    cron.EntryID
	// 备份自动清理配置
	BackupCleanStrategy  BackupCleanStrategy // 关闭 / 保留一定数量 / 保留一定时间
	BackupCleanKeepCount int                 // 保留的数量
	BackupCleanKeepDur   time.Duration       // 保留的时间
	BackupCleanTrigger   BackupCleanTrigger  // 触发方式: cron触发 / 随自动备份触发 (多种方式按位OR)
	BackupCleanCron      string              // 如果使用cron触发, 表达式
	backupCleanCronID    cron.EntryID

	AppBootTime      int64
	AppVersionCode   int64
	AppVersionOnline *VersionInfo

	UpdateRequestChan      chan *Dice
	UpdateDownloadedChan   chan string
	RebootRequestChan      chan int
	UpdateCheckRequestChan chan int

	GroupNameCache lockfree.HashMap // 群名缓存，全局共享, key string value *GroupNameCacheItem
	UserNameCache  lockfree.HashMap // 用户缓存，全局共享, key string value *GroupNameCacheItem
	UserIDCache    lockfree.HashMap // 用户id缓存 key username (string) value int64 目前仅Telegram adapter使用

	Cron                 *cron.Cron
	ServiceName          string
	JustForTest          bool
	JsRegistry           *require.Registry
	UpdateSealdiceByFile func(packName string, log *zap.SugaredLogger) bool // 使用指定压缩包升级海豹，如果出错返回false，如果成功进程会自动结束
}

type DiceConfigs struct { //nolint:revive
	DiceConfigs       []DiceConfig `yaml:"diceConfigs"`
	ServeAddress      string       `yaml:"serveAddress"`
	WebUIAddress      string       `yaml:"webUIAddress"`
	HelpDocEngineType int          `yaml:"helpDocEngineType"`

	UIPasswordSalt string   `yaml:"UIPasswordFrontendSalt"`
	UIPasswordHash string   `yaml:"uiPasswordHash"`
	AccessTokens   []string `yaml:"accessTokens"`

	AutoBackupEnable bool   `yaml:"autoBackupEnable"`
	AutoBackupTime   string `yaml:"autoBackupTime"`

	BackupClean struct {
		Strategy  int    `yaml:"strategy"`
		KeepCount int    `yaml:"keepCount"`
		KeepDur   int64  `yaml:"keepDur"`
		Trigger   int    `yaml:"trigger"`
		Cron      string `yaml:"cron"`
	} `yaml:"backupClean"`

	ServiceName string `yaml:"serviceName"`

	ConfigVersion int `yaml:"configVersion"`
}

func (dm *DiceManager) InitHelp() {
	dm.IsHelpReloading = true
	_ = os.MkdirAll("./data/helpdoc", 0o755)
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
	dm.UserIDCache = lockfree.NewHashMap()

	_ = os.MkdirAll(BackupDir, 0o755)
	_ = os.MkdirAll("./data/images", 0o755)
	_ = os.MkdirAll("./data/decks", 0o755)
	_ = os.MkdirAll("./data/names", 0o755)
	_ = os.WriteFile("./data/images/sealdice.png", IconPNG, 0o644)

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

	dm.BackupCleanStrategy = BackupCleanStrategy(dc.BackupClean.Strategy)
	dm.BackupCleanKeepCount = dc.BackupClean.KeepCount
	dm.BackupCleanKeepDur = time.Duration(dc.BackupClean.KeepDur)
	dm.BackupCleanTrigger = BackupCleanTrigger(dc.BackupClean.Trigger)
	dm.BackupCleanCron = dc.BackupClean.Cron

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
	dc.BackupClean.Strategy = int(dm.BackupCleanStrategy)
	dc.BackupClean.KeepCount = dm.BackupCleanKeepCount
	dc.BackupClean.KeepDur = int64(dm.BackupCleanKeepDur)
	dc.BackupClean.Trigger = int(dm.BackupCleanTrigger)
	dc.BackupClean.Cron = dm.BackupCleanCron
	dc.ServiceName = dm.ServiceName
	dc.ConfigVersion = 9914

	for k := range dm.AccessTokens {
		dc.AccessTokens = append(dc.AccessTokens, k)
	}

	for _, i := range dm.Dice {
		dc.DiceConfigs = append(dc.DiceConfigs, i.BaseConfig)
	}

	data, err := yaml.Marshal(dc)
	if err == nil {
		_ = os.WriteFile("./data/dice.yaml", data, 0o644)
	}
}

func (dm *DiceManager) InitDice() {
	dm.UpdateRequestChan = make(chan *Dice, 1)
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
	dm.ResetBackupClean()
}

func (dm *DiceManager) ResetAutoBackup() {
	if dm.backupEntryID != 0 {
		dm.Cron.Remove(dm.backupEntryID)
		dm.backupEntryID = 0
	}
	if dm.AutoBackupEnable {
		var err error
		dm.backupEntryID, err = dm.Cron.AddFunc(dm.AutoBackupTime, func() {
			errBackup := dm.BackupAuto()
			if errBackup != nil {
				fmt.Println("自动备份失败: ", errBackup.Error())
				return
			}
			if errBackup = dm.BackupClean(true); errBackup != nil {
				fmt.Println("滚动清理备份失败: ", errBackup.Error())
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

func (dm *DiceManager) ResetBackupClean() {
	if dm.backupCleanCronID > 0 {
		dm.Cron.Remove(dm.backupCleanCronID)
		dm.backupCleanCronID = 0
	}

	if (dm.BackupCleanTrigger & BackupCleanTriggerCron) > 0 {
		var err error
		dm.backupCleanCronID, err = dm.Cron.AddFunc(dm.BackupCleanCron, func() {
			errBackup := dm.BackupClean(false)
			if errBackup != nil {
				fmt.Println("定时清理备份失败: ", errBackup.Error())
			}
		})

		if err != nil {
			if len(dm.Dice) > 0 {
				dm.Dice[0].Logger.Errorf("设定的备份清理cron有误: %q %v", dm.BackupCleanCron, err)
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

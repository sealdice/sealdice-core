package dice

import (
	"os"
	"sync/atomic"
	"time"

	"github.com/dop251/goja_nodejs/require"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"

	"sealdice-core/logger"
	"sealdice-core/utils/dboperator/engine"
)

type VersionInfo struct {
	VersionLatest           string `json:"versionLatest"           yaml:"versionLatest"`
	VersionLatestDetail     string `json:"versionLatestDetail"     yaml:"versionLatestDetail"`
	VersionLatestCode       int64  `json:"versionLatestCode"       yaml:"versionLatestCode"`
	VersionLatestNote       string `json:"versionLatestNote"       yaml:"versionLatestNote"`
	MinUpdateSupportVersion int64  `json:"minUpdateSupportVersion" yaml:"minUpdateSupportVersion"`
	NewVersionURLPrefix     string `json:"newVersionUrlPrefix"     yaml:"newVersionUrlPrefix"`
	UpdaterURLPrefix        string `json:"updaterUrlPrefix"        yaml:"updaterUrlPrefix"`
}

type GroupNameCacheItem struct {
	Name string
	time int64
}

type DiceManager struct { //nolint:revive
	Dice                 []*Dice
	Operator             engine.DatabaseOperator
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
	AccessTokens   SyncMap[string, bool]
	IsReady        bool

	AutoBackupEnable    bool
	AutoBackupTime      string
	AutoBackupSelection BackupSelection
	backupEntryID       cron.EntryID

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

	GroupNameCache SyncMap[string, *GroupNameCacheItem] // 群名缓存，全局共享, key string value *GroupNameCacheItem
	UserNameCache  SyncMap[string, *GroupNameCacheItem] // 用户缓存，全局共享, key string value *GroupNameCacheItem
	UserIDCache    SyncMap[string, int64]               // 用户id缓存 key username (string) value int64 目前仅Telegram adapter使用

	Cron                 *cron.Cron
	ServiceName          string
	JustForTest          bool
	JsRegistry           *require.Registry
	UpdateSealdiceByFile func(packName string) bool // 使用指定压缩包升级海豹，如果出错返回false，如果成功进程会自动结束

	ContainerMode bool          // 容器模式：禁用内置适配器，不允许使用内置Lagrange和旧的内置Gocq
	CleanupFlag   atomic.Uint32 // 1 为正在清理，0为普通状态
}

type Configs struct { //nolint:revive
	DiceConfigs       []BaseConfig `yaml:"diceConfigs"`
	ServeAddress      string       `yaml:"serveAddress"`
	WebUIAddress      string       `yaml:"webUIAddress"`
	HelpDocEngineType int          `yaml:"helpDocEngineType"`

	UIPasswordSalt string   `yaml:"UIPasswordFrontendSalt"`
	UIPasswordHash string   `yaml:"uiPasswordHash"`
	AccessTokens   []string `yaml:"accessTokens"` //nolint:gosec

	AutoBackupEnable    bool   `yaml:"autoBackupEnable"`
	AutoBackupTime      string `yaml:"autoBackupTime"`
	AutoBackupSelection uint64 `yaml:"autoBackupSelection"`

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
	log := logger.M()
	dm.IsHelpReloading = true
	_ = os.MkdirAll("./data/helpdoc", 0755)
	dm.Help = new(HelpManager)
	dm.Help.EngineType = EngineType(dm.HelpDocEngineType)
	if len(dm.Dice) == 0 {
		log.Fatalf("Dice实例不存在!")
		return
	}
	dm.Help.Load(dm.Dice[0].CmdMap, dm.Dice[0].ExtList)
	dm.IsHelpReloading = false
}

// LoadDice 初始化函数
func (dm *DiceManager) LoadDice() {
	log := logger.M()
	dm.AppVersionCode = VERSION_CODE
	dm.AppBootTime = time.Now().Unix()

	_ = os.MkdirAll(BackupDir, 0755)
	_ = os.MkdirAll("./data/images", 0755)
	_ = os.MkdirAll("./data/decks", 0755)
	_ = os.MkdirAll("./data/names", 0755)
	_ = os.WriteFile("./data/images/sealdice.png", IconPNG, 0644)

	// this can be shared by multiple runtimes
	dm.JsRegistry = new(require.Registry)

	dm.Cron = cron.New()
	dm.Cron.Start()

	dm.AccessTokens = SyncMap[string, bool]{}
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

	var dc Configs
	err = yaml.Unmarshal(data, &dc)
	if err != nil {
		log.Error("读取 data/dice.yaml 发生错误: 配置文件格式不正确", err)
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
	dm.AutoBackupSelection = BackupSelection(dc.AutoBackupSelection)

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
		dm.AccessTokens.Store(i, true)
	}

	for _, i := range dc.DiceConfigs {
		newDice := new(Dice)
		newDice.BaseConfig = i
		newDice.ContainerMode = dm.ContainerMode
		dm.Dice = append(dm.Dice, newDice)
	}
}

func (dm *DiceManager) Save() {
	var dc Configs
	dc.ServeAddress = dm.ServeAddress
	dc.HelpDocEngineType = dm.HelpDocEngineType
	dc.UIPasswordSalt = dm.UIPasswordSalt
	dc.UIPasswordHash = dm.UIPasswordHash
	dc.AccessTokens = []string{}
	dc.AutoBackupTime = dm.AutoBackupTime
	dc.AutoBackupEnable = dm.AutoBackupEnable
	dc.AutoBackupSelection = uint64(dm.AutoBackupSelection)
	dc.BackupClean.Strategy = int(dm.BackupCleanStrategy)
	dc.BackupClean.KeepCount = dm.BackupCleanKeepCount
	dc.BackupClean.KeepDur = int64(dm.BackupCleanKeepDur)
	dc.BackupClean.Trigger = int(dm.BackupCleanTrigger)
	dc.BackupClean.Cron = dm.BackupCleanCron
	dc.ServiceName = dm.ServiceName
	dc.ConfigVersion = 9914

	dm.AccessTokens.Range(func(k string, v bool) bool {
		dc.AccessTokens = append(dc.AccessTokens, k)
		return true
	})

	for _, i := range dm.Dice {
		dc.DiceConfigs = append(dc.DiceConfigs, i.BaseConfig)
	}

	data, err := yaml.Marshal(dc)
	if err == nil {
		_ = os.WriteFile("./data/dice.yaml", data, 0644)
	}
}

func (dm *DiceManager) InitDice(writer *logger.UIWriter) {
	log := logger.M()
	dm.UpdateRequestChan = make(chan *Dice, 1)
	dm.RebootRequestChan = make(chan int, 1)
	dm.UpdateCheckRequestChan = make(chan int, 1)
	dm.UpdateDownloadedChan = make(chan string, 1)

	dm.LoadNames()

	g, err := NewProcessExitGroup()
	if err != nil {
		log.Warn("进程组创建失败，若进程崩溃，gocqhttp进程可能需要手动结束。")
	} else {
		dm.progressExitGroupWin = g
	}

	for _, i := range dm.Dice {
		i.Parent = dm
		i.Init(dm.Operator, writer)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Warn("帮助文档加载失败。可能是由于退出程序过快，帮助文档还未加载完成所致", r)
				if dm.Help != nil {
					log.Warn("帮助文件加载失败:", dm.Help.LoadingFn)
				}
			}
		}()
		// 加载帮助
		dm.InitHelp()
	}()

	dm.ResetAutoBackup()
	dm.ResetBackupClean()
}

func (dm *DiceManager) ResetAutoBackup() {
	log := logger.M()
	if dm.backupEntryID != 0 {
		dm.Cron.Remove(dm.backupEntryID)
		dm.backupEntryID = 0
	}
	if dm.AutoBackupEnable {
		var err error
		dm.backupEntryID, err = dm.Cron.AddFunc(dm.AutoBackupTime, func() {
			errBackup := dm.BackupAuto()
			if errBackup != nil {
				log.Errorf("自动备份失败: %v", errBackup)
				return
			}
			if errBackup = dm.BackupClean(true); errBackup != nil {
				log.Errorf("滚动清理备份失败: %v", errBackup)
			}
		})
		if err != nil {
			log.Errorf("设定的自动备份间隔有误: %v", err)
			return
		}
	}
}

func (dm *DiceManager) ResetBackupClean() {
	log := logger.M()
	if dm.backupCleanCronID > 0 {
		dm.Cron.Remove(dm.backupCleanCronID)
		dm.backupCleanCronID = 0
	}

	if (dm.BackupCleanTrigger & BackupCleanTriggerCron) > 0 {
		var err error
		dm.backupCleanCronID, err = dm.Cron.AddFunc(dm.BackupCleanCron, func() {
			errBackup := dm.BackupClean(false)
			if errBackup != nil {
				log.Errorf("定时清理备份失败: %v", errBackup)
			}
		})
		if err != nil {
			log.Errorf("设定的备份清理cron有误: %q %v", dm.BackupCleanCron, err)
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
		defaultDice.Config.MessageDelayRangeStart = DefaultConfig.MessageDelayRangeStart
		defaultDice.Config.MessageDelayRangeEnd = DefaultConfig.MessageDelayRangeEnd
		defaultDice.MarkModified()
		defaultDice.ContainerMode = dm.ContainerMode
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
	item, exists := dm.GroupNameCache.Load(id)
	if exists {
		return item.Name
	}
	return "%未知群名%"
}

// ShouldRefreshGroupInfo 检查是否应该刷新群信息，内置30秒CD
// 返回 true 表示可以刷新（不在CD中），false 表示在CD中应跳过
// 注意：返回 true 时会立即更新时间戳，防止并发调用重复触发刷新
func (dm *DiceManager) ShouldRefreshGroupInfo(id string) bool {
	now := time.Now().Unix()
	item, exists := dm.GroupNameCache.Load(id)
	if exists && now-item.time < 30 {
		return false // 30秒内不重复刷新
	}
	// 立即更新时间戳，防止并发重复刷新
	// 保留原有名称（如果存在），只更新时间
	name := ""
	if exists {
		name = item.Name
	}
	dm.GroupNameCache.Store(id, &GroupNameCacheItem{Name: name, time: now})
	return true
}

func (dm *DiceManager) TryGetUserName(id string) string {
	item, exists := dm.UserNameCache.Load(id)
	if exists {
		return item.Name
	}
	return "%未知用户%"
}

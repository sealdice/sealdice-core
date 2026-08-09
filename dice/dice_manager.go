package dice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// MaxTrayTooltipPrefixLength 自定义托盘提示前缀的最大字符数。
const MaxTrayTooltipPrefixLength = 10

var windowsReservedDiceNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {}, "CLOCK$": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {},
	"COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {},
	"LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

// ValidateDiceConfigNames ensures every Dice name is a portable single path
// segment before Dice.Init derives a data directory from it.
func ValidateDiceConfigNames(configs []BaseConfig) error {
	names := make([]string, 0, len(configs))
	for index, config := range configs {
		name := config.Name
		if name == "" || strings.TrimSpace(name) != name {
			return fmt.Errorf("第 %d 个 Dice 名称为空或含首尾空白", index+1)
		}
		if name == "." || name == ".." || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
			return fmt.Errorf("dice 名称 %q 不是安全的单一路径段", name)
		}
		if strings.ContainsAny(name, `/\\:<>"|?*`) {
			return fmt.Errorf("dice 名称 %q 含路径分隔符或非法字符", name)
		}
		if strings.HasSuffix(name, ".") {
			return fmt.Errorf("dice 名称 %q 不能以点结尾", name)
		}
		for _, current := range name {
			if current < 0x20 {
				return fmt.Errorf("dice 名称 %q 含控制字符", name)
			}
		}
		base := name
		if dot := strings.IndexByte(base, '.'); dot >= 0 {
			base = base[:dot]
		}
		if _, reserved := windowsReservedDiceNames[strings.ToUpper(base)]; reserved {
			return fmt.Errorf("dice 名称 %q 是 Windows 保留设备名", name)
		}
		for _, previous := range names {
			if strings.EqualFold(previous, name) {
				return fmt.Errorf("dice 名称 %q 与 %q 仅大小写不同", name, previous)
			}
		}
		names = append(names, name)
	}
	return nil
}

type GroupNameCacheItem struct {
	Name string
	time int64
}

type DiceManager struct { //nolint:revive
	Dice                 []*Dice
	diceLock             sync.RWMutex
	Operator             engine.DatabaseOperator
	ServeAddress         string
	trayTooltip          string
	trayTooltipLock      sync.RWMutex
	Help                 *HelpManager
	IsHelpReloading      bool
	helpReloadLock       sync.Mutex
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

	ContainerMode            bool          // 容器模式：禁用内置适配器，不允许使用内置Lagrange和旧的内置Gocq
	CleanupFlag              atomic.Uint32 // 0 为运行中，1 为静默中，2 为已完成释放
	runtimeCtx               context.Context
	runtimeCancel            context.CancelFunc
	runtimeMu                sync.Mutex
	runtimeClosing           bool
	runtimeListenAddress     string
	runtimeConfiguredAddress string
	runtimeWG                sync.WaitGroup
	quiescePhase             lifecyclePhase
	finalizePhase            lifecyclePhase
}

type Configs struct { //nolint:revive
	DiceConfigs       []BaseConfig `yaml:"diceConfigs"`
	ServeAddress      string       `yaml:"serveAddress"`
	TrayTooltip       string       `yaml:"trayTooltip"`
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
	_ = dm.reloadHelp(false)
}

func (dm *DiceManager) ReloadHelp() error {
	return dm.reloadHelp(true)
}

func (dm *DiceManager) reloadHelp(closeCurrent bool) error {
	log := logger.M()
	dm.helpReloadLock.Lock()
	defer dm.helpReloadLock.Unlock()

	dm.IsHelpReloading = true
	defer func() {
		dm.IsHelpReloading = false
	}()
	_ = os.MkdirAll("./data/helpdoc", 0755)
	if closeCurrent && dm.Help != nil {
		dm.Help.Close()
	}
	if len(dm.Dice) == 0 {
		err := errors.New("Dice实例不存在")
		log.Error(err)
		return err
	}

	nextHelp := dm.Help
	if nextHelp == nil || closeCurrent {
		nextHelp = &HelpManager{EngineType: EngineType(dm.HelpDocEngineType)}
	}
	nextHelp.Load(dm.Dice[0], dm.Dice[0].CmdMap, dm.Dice[0].ExtList)
	if !nextHelp.IsAvailable() {
		err := errors.New("帮助文档搜索引擎不可用")
		log.Error(err)
		return err
	}
	dm.Help = nextHelp
	return nil
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
	dm.SetTrayTooltip(dc.TrayTooltip)
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
		dm.appendDice(newDice)
	}
}

// SetRuntimeContext 设置当前 Runtime 的取消域，必须在 InitDice 前调用。
func (dm *DiceManager) SetRuntimeContext(ctx context.Context, cancel context.CancelFunc) {
	dm.runtimeMu.Lock()
	defer dm.runtimeMu.Unlock()
	dm.runtimeCtx = ctx
	dm.runtimeCancel = cancel
	dm.runtimeClosing = false
}

// SetRuntimeServeAddress separates the process listener from the address that
// should be persisted for the next process start.
func (dm *DiceManager) SetRuntimeServeAddress(listenAddress, configuredAddress string) {
	dm.runtimeMu.Lock()
	defer dm.runtimeMu.Unlock()
	dm.runtimeListenAddress = listenAddress
	dm.runtimeConfiguredAddress = configuredAddress
	dm.ServeAddress = listenAddress
}

func (dm *DiceManager) context() context.Context {
	if dm.runtimeCtx == nil {
		return context.Background()
	}
	return dm.runtimeCtx
}

func (dm *DiceManager) goRuntime(fn func(context.Context)) bool {
	dm.runtimeMu.Lock()
	if dm.runtimeClosing {
		dm.runtimeMu.Unlock()
		return false
	}
	ctx := dm.context()
	dm.runtimeWG.Add(1)
	dm.runtimeMu.Unlock()
	go func() {
		defer dm.runtimeWG.Done()
		fn(ctx)
	}()
	return true
}

// GoRuntime 注册属于当前 Runtime generation 的后台任务。
// Runtime 进入关闭阶段后会拒绝新任务，避免 Wait 与 Add 并发。
func (dm *DiceManager) GoRuntime(fn func(context.Context)) bool {
	if fn == nil {
		return false
	}
	return dm.goRuntime(fn)
}

func (dm *DiceManager) beginRuntimeTask() (func(), bool) {
	dm.runtimeMu.Lock()
	defer dm.runtimeMu.Unlock()
	if dm.runtimeClosing {
		return nil, false
	}
	dm.runtimeWG.Add(1)
	return dm.runtimeWG.Done, true
}

func (dm *DiceManager) Save() {
	var dc Configs
	dm.runtimeMu.Lock()
	serveAddress := dm.ServeAddress
	if dm.runtimeListenAddress != "" {
		if serveAddress != dm.runtimeListenAddress {
			dm.runtimeConfiguredAddress = serveAddress
			dm.ServeAddress = dm.runtimeListenAddress
		}
		if dm.runtimeConfiguredAddress != "" {
			serveAddress = dm.runtimeConfiguredAddress
		}
	}
	dm.runtimeMu.Unlock()
	dc.ServeAddress = serveAddress
	dc.TrayTooltip = dm.GetTrayTooltip()
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

	data, err := yaml.Marshal(dc) //nolint:gosec
	if err == nil {
		_ = os.WriteFile("./data/dice.yaml", data, 0644)
	}
}

func (dm *DiceManager) GetTrayTooltip() string {
	dm.trayTooltipLock.RLock()
	defer dm.trayTooltipLock.RUnlock()
	return dm.trayTooltip
}

// NormalizeTrayTooltipPrefix 规范化托盘提示前缀，并按 Unicode 字符数限制长度。
func NormalizeTrayTooltipPrefix(tooltip string) string {
	tooltip = strings.TrimSpace(tooltip)
	runes := []rune(tooltip)
	if len(runes) > MaxTrayTooltipPrefixLength {
		return string(runes[:MaxTrayTooltipPrefixLength])
	}
	return tooltip
}

func (dm *DiceManager) SetTrayTooltip(tooltip string) {
	dm.trayTooltipLock.Lock()
	defer dm.trayTooltipLock.Unlock()
	dm.trayTooltip = NormalizeTrayTooltipPrefix(tooltip)
}

// DiceSnapshot 返回当前 Dice 实例切片的副本。
func (dm *DiceManager) DiceSnapshot() []*Dice {
	dm.diceLock.RLock()
	defer dm.diceLock.RUnlock()
	return append([]*Dice(nil), dm.Dice...)
}

func (dm *DiceManager) appendDice(instance *Dice) {
	dm.diceLock.Lock()
	defer dm.diceLock.Unlock()
	dm.Dice = append(dm.Dice, instance)
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

	dm.goRuntime(func(ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Warn("帮助文档加载失败。可能是由于退出程序过快，帮助文档还未加载完成所致", r)
				if dm.Help != nil {
					log.Warn("帮助文件加载失败:", dm.Help.LoadingFn)
				}
			}
		}()
		select {
		case <-ctx.Done():
			return
		default:
		}
		// 加载帮助
		dm.InitHelp()
	})

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

	dm.diceLock.Lock()
	defer dm.diceLock.Unlock()
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

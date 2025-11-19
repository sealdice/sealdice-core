package dice

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	maxChainDepth = 10
)

// StringSet 是一个字符串集合，内部使用 map[string]struct{} 实现，
// 序列化时转换为 []string 格式以提高可读性。
type StringSet map[string]struct{}

// MarshalYAML 将 StringSet 序列化为 []string
func (s StringSet) MarshalYAML() (interface{}, error) {
	if s == nil {
		return []string{}, nil
	}
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// UnmarshalYAML 将 []string 反序列化为 StringSet
func (s *StringSet) UnmarshalYAML(node *yaml.Node) error {
	var items []string
	if err := node.Decode(&items); err != nil {
		return err
	}
	*s = make(map[string]struct{}, len(items))
	for _, item := range items {
		(*s)[item] = struct{}{}
	}
	return nil
}

// ActivateReason 用于区分扩展开启的触发来源（手动、首次消息自动、脚本重载等）。
type ActivateReason string

// DeactivateReason 用于区分扩展关闭的触发来源。
type DeactivateReason string

const (
	ActivateReasonManual       ActivateReason   = "manual"
	ActivateReasonFirstMessage ActivateReason   = "first_message_auto"
	ActivateReasonReload       ActivateReason   = "reload"
	DeactivateReasonManual     DeactivateReason = "manual"
	DeactivateReasonSystem     DeactivateReason = "system"
)

// ActiveWithGraph 描述扩展伴随关系：A -> [B,C] 表示激活 A 时需要附带 B、C。
type ActiveWithGraph map[string][]string

func (d *Dice) rebuildActiveWithGraph() {
	d.ActiveWithGraph = BuildActiveWithGraph(d.ExtList)
}

func (d *Dice) activeWithGraph() *SyncMap[string, []string] {
	if d.ActiveWithGraph == nil {
		d.rebuildActiveWithGraph()
	}
	return d.ActiveWithGraph
}

// BuildActiveWithGraph 根据扩展列表构建伴随激活图。
// 返回反向图：A -> [B,C] 表示 B、C 跟随 A（当 A 开启时，B、C 也开启）
func BuildActiveWithGraph(exts []*ExtInfo) *SyncMap[string, []string] {
	graph := new(SyncMap[string, []string])
	// 构建反向图：如果 ext 跟随 target，则在 graph[target] 中加入 ext
	for _, ext := range exts {
		for _, target := range ext.ActiveWith {
			followers, _ := graph.Load(target)
			followers = append(followers, ext.Name)
			graph.Store(target, followers)
		}
	}
	return graph
}

// collectChainedNames 收集所有跟随 base 扩展的伴随扩展（递归查找，返回拓扑顺序）。
func collectChainedNames(logger *zap.SugaredLogger, graph *SyncMap[string, []string], base string, depthLimit int) []string {
	visited := map[string]bool{}
	visiting := map[string]bool{}
	var order []string

	if graph == nil {
		return order
	}

	var dfs func(name string, depth int)
	dfs = func(name string, depth int) {
		if depth > depthLimit {
			logger.Warnf("扩展 %s 的 ActiveWith 链深度超过 %d，停止展开", name, depthLimit)
			return
		}
		if visiting[name] {
			logger.Warnf("检测到扩展 %s 的 ActiveWith 存在循环，跳过连带操作", name)
			return
		}
		if visited[name] {
			return
		}
		visiting[name] = true
		followers, _ := graph.Load(name)
		for _, follower := range followers {
			dfs(follower, depth+1)
		}
		visiting[name] = false
		visited[name] = true
		if name != base {
			order = append(order, name)
		}
	}

	dfs(base, 0)
	return order
}

// RegisterBuiltinExt 注册所有内置扩展。
func (d *Dice) RegisterBuiltinExt() {
	RegisterBuiltinExtCoc7(d)
	RegisterBuiltinExtLog(d)
	RegisterBuiltinExtFun(d)
	RegisterBuiltinExtDeck(d)
	RegisterBuiltinExtReply(d)
	RegisterBuiltinExtDnd5e(d)
	RegisterBuiltinStory(d)
	RegisterBuiltinExtExp(d)
	RegisterBuiltinExtCore(d)

	d.RegisterBuiltinSystemTemplate()
}

func (d *Dice) RegisterBuiltinSystemTemplate() {
	for _, asset := range []string{"coc7.yaml", "dnd5e.yaml"} {
		tmpl, err := loadBuiltinTemplate(asset)
		if err != nil {
			if d.Logger != nil {
				d.Logger.Errorf("failed to load builtin game template %s: %v", asset, err)
			}
			continue
		}
		d.GameSystemTemplateAdd(tmpl)
	}
}

// RegisterExtension 注册扩展并刷新伴随激活图/版本号。
//
// panic 当扩展的 Name/Aliases 与现有扩展冲突。
func (d *Dice) RegisterExtension(extInfo *ExtInfo) {
	for _, name := range append(extInfo.Aliases, extInfo.Name) {
		if collide := d.ExtFind(name, false); collide != nil {
			panicMsg := fmt.Sprintf("扩展<%s>的名称%q与扩展<%s>冲突", extInfo.Name, name, collide.Name)
			panic(panicMsg)
		}
	}

	extInfo.dice = d
	d.ExtList = append(d.ExtList, extInfo)
	if d.ExtRegistry == nil {
		d.ExtRegistry = new(SyncMap[string, *ExtInfo])
	}
	registerKey := func(key string) {
		if key == "" {
			return
		}
		d.ExtRegistry.Store(key, extInfo)
		lower := strings.ToLower(key)
		if lower != key {
			d.ExtRegistry.Store(lower, extInfo)
		}
	}
	registerKey(extInfo.Name)
	for _, alias := range extInfo.Aliases {
		registerKey(alias)
	}
	d.ExtRegistryVersion++
	d.ActiveWithGraph = nil
}

func (d *Dice) GetExtDataDir(extName string) string {
	p := path.Join(d.BaseConfig.DataDir, "extensions", extName)
	_ = os.MkdirAll(p, 0o755)
	return p
}

func (d *Dice) GetDiceDataPath(name string) string {
	return path.Join(d.BaseConfig.DataDir, name)
}

func (d *Dice) GetExtConfigFilePath(extName string, filename string) string {
	return path.Join(d.GetExtDataDir(extName), filename)
}

// ClearExtStorage 删除扩展 storage.db 并重新初始化。
func ClearExtStorage(d *Dice, ext *ExtInfo, name string) error {
	err := ext.StorageClose()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(d.BaseConfig.DataDir, "extensions", name, "storage.db")
	err = os.Remove(dbPath)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return err
	}

	return ext.StorageInit()
}

func GetExtensionDesc(ei *ExtInfo) string {
	var text strings.Builder
	text.WriteString("> ")
	text.WriteString(ei.Brief)
	text.WriteString("\n提供指令:\n")
	keys := make([]string, 0, len(ei.CmdMap))

	valueMap := map[*CmdItemInfo]bool{}

	for k := range ei.CmdMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		item := ei.CmdMap[name]
		if valueMap[item] {
			continue
		}
		valueMap[item] = true
		if item.ShortHelp == "" {
			text.WriteString(".")
			text.WriteString(item.Name)
			text.WriteString("\n")
		} else {
			text.WriteString(item.ShortHelp)
			text.WriteString("\n")
		}
	}

	return text.String()
}

// callWithJsCheck 保留旧行为：JS 扩展需要切回事件循环，避免并发问题。
func (i *ExtInfo) callWithJsCheck(d *Dice, f func()) {
	if i.IsJsExt {
		if d.Config.JsEnable {
			loop, err := d.ExtLoopManager.GetLoop(i.JSLoopVersion)
			if err != nil {
				i.dice.Logger.Errorf("扩展<%s>运行环境已经过期: %v", i.Name, err)
				return
			}
			waitRun := make(chan int, 1)
			loop.RunOnLoop(func(vm *goja.Runtime) {
				defer func() {
					if r := recover(); r != nil {
						d.Logger.Error("JS脚本异常:", r)
					}
					waitRun <- 1
				}()

				f()
			})
			<-waitRun
		} else {
			d.Logger.Infof("当前已关闭js扩展<%v>", i.Name)
		}
	} else {
		f()
	}
}

// StorageInit 与旧版一致：使用互斥锁确保只初始化一次。
func (i *ExtInfo) StorageInit() error {
	var err error
	if i.dice == nil {
		return errors.New("[扩展]:必须先把扩展注册到 Dice")
	}
	d := i.dice

	i.dbMu.Lock()
	defer i.dbMu.Unlock()
	if i.init {
		return nil
	}

	dir := d.GetExtDataDir(i.Name)
	fn := path.Join(dir, "storage.db")
	i.Storage, err = buntdb.Open(fn)
	if err != nil {
		d.Logger.Errorf("[扩展]:初始化扩展数据库失败，原因: %v，路径为：%s", err, fn)
		return err
	}
	i.init = true
	return nil
}

// StorageClose 与旧逻辑一致：先判断 init，再关闭数据库并回收状态。
func (i *ExtInfo) StorageClose() error {
	i.dbMu.Lock()
	defer i.dbMu.Unlock()
	if !i.init {
		return nil
	}
	if i.Storage == nil {
		return errors.New("[扩展]:Storage 未初始化")
	}
	err := i.Storage.Close()
	if err != nil {
		i.dice.Logger.Errorf("[扩展]:关闭扩展数据库失败，原因: %v", err)
		return err
	}
	i.Storage = nil
	i.init = false
	return nil
}

// StorageSet/StorageGet 只是辅助函数，原注释无变更。
func (i *ExtInfo) StorageSet(k, v string) error {
	if err := i.StorageInit(); err != nil {
		return err
	}
	db := i.Storage
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(k, v, nil)
		return err
	})
}

// StorageGet 与旧实现保持一致，出现 ErrNotFound 时直接返回空字符串。
func (i *ExtInfo) StorageGet(k string) (string, error) {
	if err := i.StorageInit(); err != nil {
		return "", err
	}

	var val string
	var err error

	db := i.Storage
	err = db.View(func(tx *buntdb.Tx) error {
		val, err = tx.Get(k)
		if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		return nil
	})

	return val, err
}

// -------------------- 群扩展状态管理 --------------------

func (group *GroupInfo) ensureInactivatedSet() {
	if group.InactivatedExtSet == nil {
		group.InactivatedExtSet = StringSet{}
	}
}

// IsExtInactivated 判断扩展是否被标记为手动关闭。
func (group *GroupInfo) IsExtInactivated(name string) bool {
	if group.InactivatedExtSet == nil {
		return false
	}
	_, ok := group.InactivatedExtSet[name]
	return ok
}

func (group *GroupInfo) RemoveFromInactivated(name string) {
	if group.InactivatedExtSet == nil {
		return
	}
	delete(group.InactivatedExtSet, name)
}

func (group *GroupInfo) AddToInactivated(name string) {
	group.ensureInactivatedSet()
	group.InactivatedExtSet[name] = struct{}{}
}

func (group *GroupInfo) indexOfActivated(name string) int {
	for idx, item := range group.ActivatedExtList {
		if item != nil && item.Name == name {
			return idx
		}
	}
	return -1
}

func (group *GroupInfo) removeActivatedByName(name string) bool {
	idx := group.indexOfActivated(name)
	if idx == -1 {
		return false
	}
	group.ActivatedExtList = append(group.ActivatedExtList[:idx], group.ActivatedExtList[idx+1:]...)
	return true
}

func (group *GroupInfo) ensureSnapshotFromActivated() {
	if len(group.ActivatedExtList) == 0 {
		group.ExtActiveListSnapshot = nil
		return
	}
	names := make([]string, 0, len(group.ActivatedExtList))
	for _, item := range group.ActivatedExtList {
		if item != nil {
			names = append(names, item.Name)
		}
	}
	group.ExtActiveListSnapshot = names
}

// ExtActive 开启扩展（手动触发）。
func (group *GroupInfo) ExtActive(ei *ExtInfo) {
	group.extActivateInternal(ei, ActivateReasonManual)
}

// ExtActiveBySnapshotOrder 兼容旧逻辑：按照快照顺序尝试开启扩展。
func (group *GroupInfo) ExtActiveBySnapshotOrder(ei *ExtInfo, isFirstTimeLoad bool) {
	if ei == nil {
		return
	}
	firstMap := map[string]bool{}
	if isFirstTimeLoad {
		firstMap[ei.Name] = true
	}
	group.ExtActiveBatchBySnapshotOrder([]*ExtInfo{ei}, firstMap)
}

// ExtActiveBatchBySnapshotOrder 批量按照快照顺序开启扩展。
func (group *GroupInfo) ExtActiveBatchBySnapshotOrder(extInfos []*ExtInfo, isFirstTimeLoad map[string]bool) {
	if len(extInfos) == 0 {
		return
	}
	// 保存快照到本地变量，因为 extActivateInternal 会修改 group.ExtActiveListSnapshot
	snapshot := make([]string, len(group.ExtActiveListSnapshot))
	copy(snapshot, group.ExtActiveListSnapshot)

	orderSet := StringSet{}
	for _, name := range snapshot {
		orderSet[name] = struct{}{}
	}

	lookup := map[string]*ExtInfo{}
	for _, ext := range extInfos {
		if ext != nil {
			lookup[ext.Name] = ext
		}
	}

	// 反向遍历快照，因为 extActivateInternal 会将扩展插入列表前端
	// 这样可以保持原有的优先级顺序
	for i := len(snapshot) - 1; i >= 0; i-- {
		name := snapshot[i]
		if group.IsExtInactivated(name) {
			continue
		}
		if ext := lookup[name]; ext != nil {
			group.extActivateInternal(ext, ActivateReasonReload)
		}
	}

	for _, ext := range extInfos {
		if ext == nil {
			continue
		}
		if _, ok := orderSet[ext.Name]; ok {
			continue
		}
		if group.IsExtInactivated(ext.Name) {
			continue
		}
		if first, exists := isFirstTimeLoad[ext.Name]; exists && first {
			group.extActivateInternal(ext, ActivateReasonReload)
			continue
		}
		if ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive {
			group.extActivateInternal(ext, ActivateReasonReload)
		}
	}
	group.ensureSnapshotFromActivated()
}

// ExtInactive 手动关闭扩展，连带关闭伴随扩展。
func (group *GroupInfo) ExtInactive(ei *ExtInfo) *ExtInfo {
	return group.extDeactivateInternal(ei, DeactivateReasonManual, true)
}

// ExtInactiveSystem 系统级关闭（不写入 Inactivated 集合）。
func (group *GroupInfo) ExtInactiveSystem(ei *ExtInfo) *ExtInfo {
	return group.extDeactivateInternal(ei, DeactivateReasonSystem, false)
}

// ExtInactiveByName 通过名称关闭扩展。
func (group *GroupInfo) ExtInactiveByName(name string) *ExtInfo {
	for _, item := range group.ActivatedExtList {
		if item != nil && item.Name == name {
			return group.ExtInactive(item)
		}
	}
	return nil
}

// ExtGetActive 查询扩展是否处于激活状态。
func (group *GroupInfo) ExtGetActive(name string) *ExtInfo {
	for _, item := range group.ActivatedExtList {
		if item != nil && item.Name == name {
			return item
		}
	}
	return nil
}

func (group *GroupInfo) extActivateInternal(ei *ExtInfo, reason ActivateReason) {
	if ei == nil || ei.dice == nil {
		return
	}
	group.ensureInactivatedSet()
	d := ei.dice
	graph := d.activeWithGraph()
	// 先激活主扩展
	group.promoteExt(ei)
	// 再激活伴随扩展，使其指令优先级更高（越晚激活优先级越高）
	names := collectChainedNames(d.Logger, graph, ei.Name, maxChainDepth)
	for _, name := range names {
		if ext := d.ExtFind(name, false); ext != nil {
			group.promoteExt(ext)
		}
	}
	group.ensureSnapshotFromActivated()
	group.UpdatedAtTime = time.Now().Unix()
}

func (group *GroupInfo) promoteExt(ext *ExtInfo) {
	if ext == nil {
		return
	}
	group.RemoveFromInactivated(ext.Name)
	if group.removeActivatedByName(ext.Name) {
		// 已在列表中，被移除后再插入
	}
	group.ActivatedExtList = append([]*ExtInfo{ext}, group.ActivatedExtList...)
}

func (group *GroupInfo) extDeactivateInternal(ei *ExtInfo, reason DeactivateReason, markInactivated bool) *ExtInfo {
	if ei == nil || ei.dice == nil {
		return nil
	}
	d := ei.dice
	graph := d.activeWithGraph()
	names := []string{ei.Name}
	names = append(names, collectChainedNames(d.Logger, graph, ei.Name, maxChainDepth)...)

	var removed *ExtInfo
	for _, name := range names {
		if markInactivated {
			group.AddToInactivated(name)
		}
		if ext := d.ExtFind(name, false); ext != nil {
			if ext.Storage != nil {
				_ = ext.StorageClose()
			}
			if group.removeActivatedByName(name) && removed == nil {
				removed = ext
			}
		}
	}
	group.ensureSnapshotFromActivated()
	group.UpdatedAtTime = time.Now().Unix()
	return removed
}

// SyncExtensionsOnMessage 在群首次收到消息或扩展集发生变动时调用，按 AutoActive 规则处理新扩展。
func (group *GroupInfo) SyncExtensionsOnMessage(d *Dice) {
	if d == nil {
		return
	}
	if group.ExtAppliedVersion == d.ExtRegistryVersion {
		return
	}
	group.ensureInactivatedSet()
	known := make(map[string]struct{}, len(group.ActivatedExtList)+len(group.InactivatedExtSet))
	for _, ext := range group.ActivatedExtList {
		if ext != nil {
			known[ext.Name] = struct{}{}
		}
	}
	for name := range group.InactivatedExtSet {
		known[name] = struct{}{}
	}

	for _, ext := range d.ExtList {
		if _, exists := known[ext.Name]; exists {
			continue
		}
		if ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive) {
			group.extActivateInternal(ext, ActivateReasonFirstMessage)
		} else {
			group.AddToInactivated(ext.Name)
		}
	}
	group.ExtAppliedVersion = d.ExtRegistryVersion
	group.ensureSnapshotFromActivated()
}

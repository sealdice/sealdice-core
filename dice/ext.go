package dice

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/dop251/goja"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"

	"sealdice-core/dice/events"
)

const (
	maxChainDepth = 10
)

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
	d.ActiveWithGraphMu.Lock()
	defer d.ActiveWithGraphMu.Unlock()
	d.rebuildActiveWithGraphLocked()
}

func (d *Dice) rebuildActiveWithGraphLocked() {
	d.ActiveWithGraph = BuildActiveWithGraph(d.ExtList)
}

func (d *Dice) activeWithGraph() *SyncMap[string, []string] {
	d.ActiveWithGraphMu.RLock()
	graph := d.ActiveWithGraph
	d.ActiveWithGraphMu.RUnlock()
	if graph != nil {
		return graph
	}

	d.ActiveWithGraphMu.Lock()
	defer d.ActiveWithGraphMu.Unlock()
	if d.ActiveWithGraph == nil {
		d.rebuildActiveWithGraphLocked()
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
	d.ActiveWithGraphMu.Lock()
	d.ActiveWithGraph = nil
	d.ActiveWithGraphMu.Unlock()
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

	cmdMap := ei.GetCmdMap()
	keys := make([]string, 0, len(cmdMap))

	valueMap := map[*CmdItemInfo]bool{}

	for k := range cmdMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		item := cmdMap[name]
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

// GetRealExt 获取真实扩展（处理 wrapper 代理）
// 如果不是 wrapper，返回自身
// 如果是有效的 wrapper，返回 JsExtRegistry 中的真实扩展
// 如果是无效的 wrapper 或找不到真实扩展，返回 nil
func (i *ExtInfo) GetRealExt() *ExtInfo {
	if !i.IsWrapper {
		return i // 非 wrapper，返回自身
	}
	if i.IsDeleted {
		return nil // 已删除的 wrapper
	}
	if i.dice == nil || i.dice.JsExtRegistry == nil {
		return nil // 未初始化
	}
	if realExt, ok := i.dice.JsExtRegistry.Load(i.TargetName); ok {
		return realExt
	}
	return nil // 找不到真实扩展
}

// GetCmdMap 获取命令映射（处理 wrapper 代理）
// 如果是 wrapper，返回真实扩展的 CmdMap
// 如果正在重载或无法获取真实扩展，返回空 CmdMap
func (i *ExtInfo) GetCmdMap() CmdMapCls {
	// 重载期间，wrapper 的真实扩展可能还未注册，返回空 CmdMap
	// 注意：如果 i.dice 为 nil（理论上不应发生），会走到 GetRealExt()
	// GetRealExt() 内部已处理 dice==nil 的情况，会返回 nil，最终返回空 CmdMap
	if i.IsWrapper && i.dice != nil && i.dice.JsReloading {
		return CmdMapCls{}
	}
	ext := i.GetRealExt()
	if ext == nil {
		return CmdMapCls{}
	}
	return ext.CmdMap
}

// CallOnMessageSend 调用 OnMessageSend 回调（处理 wrapper 代理）
func (i *ExtInfo) CallOnMessageSend(d *Dice, ctx *MsgContext, msg *Message, flag string) {
	ext := i.GetRealExt()
	if ext == nil || ext.OnMessageSend == nil {
		return
	}
	ext.callWithJsCheck(d, func() {
		ext.OnMessageSend(ctx, msg, flag)
	})
}

// CallOnMessageDeleted 调用 OnMessageDeleted 回调（处理 wrapper 代理）
func (i *ExtInfo) CallOnMessageDeleted(d *Dice, ctx *MsgContext, msg *Message) {
	ext := i.GetRealExt()
	if ext == nil || ext.OnMessageDeleted == nil {
		return
	}
	ext.callWithJsCheck(d, func() {
		ext.OnMessageDeleted(ctx, msg)
	})
}

// CallOnGroupLeave 调用 OnGroupLeave 回调（处理 wrapper 代理）
func (i *ExtInfo) CallOnGroupLeave(d *Dice, ctx *MsgContext, event *events.GroupLeaveEvent) {
	ext := i.GetRealExt()
	if ext == nil || ext.OnGroupLeave == nil {
		return
	}
	ext.callWithJsCheck(d, func() {
		ext.OnGroupLeave(ctx, event)
	})
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
	target := i.GetRealExt()
	if target == nil {
		return errors.New("[扩展]:目标扩展不存在")
	}
	if err := target.StorageInit(); err != nil {
		return err
	}
	db := target.Storage
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(k, v, nil)
		return err
	})
}

// StorageGet 与旧实现保持一致，出现 ErrNotFound 时直接返回空字符串。
func (i *ExtInfo) StorageGet(k string) (string, error) {
	target := i.GetRealExt()
	if target == nil {
		return "", errors.New("[扩展]:目标扩展不存在")
	}
	if err := target.StorageInit(); err != nil {
		return "", err
	}

	var val string
	var err error

	db := target.Storage
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
	for idx, item := range group.activatedExtList {
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
	group.activatedExtList = append(group.activatedExtList[:idx], group.activatedExtList[idx+1:]...)
	return true
}

// SyncWrapperStatus 延迟更新机制：检查并移除已删除的 wrapper
// 在消息到达时调用，根据时间戳判断是否需要更新
// 返回值表示是否进行了更新
// 注意：只移除被明确标记为 IsDeleted 的 wrapper（由 JsDelete/ExtRemove 设置）
func (group *GroupInfo) SyncWrapperStatus(d *Dice) bool {
	if d == nil {
		return false
	}

	// 确保延迟初始化已完成（GetActivatedExtList 会处理，内部有锁）
	_ = group.GetActivatedExtList(d)

	// 快速路径：无需更新（无锁检查，减少锁竞争）
	if atomic.LoadInt64(&group.ExtAppliedTime) >= d.ExtUpdateTime {
		return false
	}

	// 加锁保护 activatedExtList 的读写操作
	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()

	// double-check：在锁内再次检查，避免重复处理
	if atomic.LoadInt64(&group.ExtAppliedTime) >= d.ExtUpdateTime {
		return false
	}

	// 过滤已删除的 wrapper（只检查 IsDeleted 标志，不检查 JsExtRegistry）
	needsUpdate := false
	newList := make([]*ExtInfo, 0, len(group.activatedExtList))

	for _, wrapper := range group.activatedExtList {
		if wrapper == nil {
			continue
		}

		// 内置扩展（非 wrapper）直接保留
		if !wrapper.IsWrapper {
			newList = append(newList, wrapper)
			continue
		}

		// 只检查 IsDeleted 标志（由 JsDelete/ExtRemove 明确设置）
		if wrapper.IsDeleted {
			needsUpdate = true
			// 关闭 Storage（如果有的话）
			if wrapper.Storage != nil {
				_ = wrapper.StorageClose()
			}
			continue // 移除已删除的 wrapper
		}

		newList = append(newList, wrapper) // 保留有效 wrapper
	}

	if needsUpdate {
		group.activatedExtList = newList
		group.MarkDirty(d)
	}

	atomic.StoreInt64(&group.ExtAppliedTime, d.ExtUpdateTime)
	return needsUpdate
}

// ExtActive 开启扩展（手动触发）。
func (group *GroupInfo) ExtActive(ei *ExtInfo) {
	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()
	group.extActivateInternal(ei, ActivateReasonManual)
}

// ExtActivateBatch 批量激活扩展。
// 对于首次加载的扩展（isFirstTimeLoad[name]=true），直接激活。
// 对于非首次加载的扩展，根据 AutoActive 设置决定是否激活。
// 已在 InactivatedExtSet 中的扩展不会被激活。
func (group *GroupInfo) ExtActivateBatch(extInfos []*ExtInfo, isFirstTimeLoad map[string]bool) {
	if len(extInfos) == 0 {
		return
	}

	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()

	group.ensureInactivatedSet()

	// 构建已知扩展集合
	known := make(map[string]struct{}, len(group.activatedExtList))
	for _, ext := range group.activatedExtList {
		if ext != nil {
			known[ext.Name] = struct{}{}
		}
	}

	for _, ext := range extInfos {
		if ext == nil {
			continue
		}
		// 跳过已激活的扩展
		if _, exists := known[ext.Name]; exists {
			continue
		}
		// 跳过被用户关闭的扩展
		if group.IsExtInactivated(ext.Name) {
			continue
		}
		// 首次加载的扩展直接激活
		if first, exists := isFirstTimeLoad[ext.Name]; exists && first {
			group.extActivateInternal(ext, ActivateReasonFirstMessage)
			known[ext.Name] = struct{}{}
			continue
		}
		// 非首次加载，根据 AutoActive 决定
		if ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive) {
			group.extActivateInternal(ext, ActivateReasonFirstMessage)
			known[ext.Name] = struct{}{}
		} else {
			group.AddToInactivated(ext.Name)
		}
	}
}

// ExtInactive 手动关闭扩展，连带关闭伴随扩展。
func (group *GroupInfo) ExtInactive(ei *ExtInfo) *ExtInfo {
	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()
	return group.extDeactivateInternal(ei, DeactivateReasonManual, true)
}

// ExtInactiveSystem 系统级关闭（不写入 Inactivated 集合）。
func (group *GroupInfo) ExtInactiveSystem(ei *ExtInfo) *ExtInfo {
	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()
	return group.extDeactivateInternal(ei, DeactivateReasonSystem, false)
}

// ExtInactiveByName 通过名称关闭扩展。
func (group *GroupInfo) ExtInactiveByName(name string) *ExtInfo {
	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()
	// 直接调用内部函数避免死锁（不调用 ExtInactive）
	for _, item := range group.activatedExtList {
		if item != nil && item.Name == name {
			return group.extDeactivateInternal(item, DeactivateReasonManual, true)
		}
	}
	return nil
}

// ExtGetActive 查询扩展是否处于激活状态。
func (group *GroupInfo) ExtGetActive(name string) *ExtInfo {
	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()
	for _, item := range group.activatedExtList {
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
	group.MarkDirty(d)
}

func (group *GroupInfo) promoteExt(ext *ExtInfo) {
	if ext == nil {
		return
	}
	group.RemoveFromInactivated(ext.Name)
	_ = group.removeActivatedByName(ext.Name)
	group.activatedExtList = append([]*ExtInfo{ext}, group.activatedExtList...)
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
			if target := ext.GetRealExt(); target != nil && target.Storage != nil {
				_ = target.StorageClose()
			}
			if group.removeActivatedByName(name) && removed == nil {
				removed = ext
			}
		}
	}
	group.MarkDirty(d)
	return removed
}

// SyncExtensionsOnMessage 在群首次收到消息或扩展集发生变动时调用，按 AutoActive 规则处理新扩展。
func (group *GroupInfo) SyncExtensionsOnMessage(d *Dice) {
	if d == nil {
		return
	}
	// 快速路径：无需更新
	if group.ExtAppliedVersion == d.ExtRegistryVersion {
		return
	}

	group.extInitMu.Lock()
	defer group.extInitMu.Unlock()

	// double-check：在锁内再次检查
	if group.ExtAppliedVersion == d.ExtRegistryVersion {
		return
	}

	group.ensureInactivatedSet()
	known := make(map[string]struct{}, len(group.activatedExtList)+len(group.InactivatedExtSet))
	for _, ext := range group.activatedExtList {
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
}

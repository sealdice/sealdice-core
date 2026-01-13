package dice

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sealdice-core/dice/events"
)

// PythonExtInfo Python扩展信息结构体
type PythonExtInfo struct {
	Name    string   `json:"name"    yaml:"name"` // 名字
	Aliases []string `json:"aliases" yaml:"-"`    // 别名
	Version string   `json:"version" yaml:"-"`    // 版本

	AutoActive      bool      `json:"-" yaml:"-"` // 是否自动开启
	CmdMap          CmdMapCls `json:"-" yaml:"-"` // 指令集合
	Brief           string    `json:"-"            yaml:"-"`
	ActiveOnPrivate bool      `json:"-"            yaml:"-"`

	DefaultSetting *ExtDefaultSettingItem `json:"-" yaml:"-"` // 默认配置

	Author       string   `json:"-" yaml:"-"`
	ConflictWith []string `json:"-"        yaml:"-"`
	Official     bool     `json:"-"        yaml:"-"` // js插件

	ActiveWith []string `json:"-" yaml:"-"` // 跟随开关

	dice          *Dice
	IsPythonExt   bool          `json:"-"`

	SourcePath    string        `json:"-" yaml:"-"`
	Storage       interface{}   `json:"-" yaml:"-"` // Python扩展的存储
	dbMu          sync.Mutex    `yaml:"-"` // 互斥锁
	init          bool          `yaml:"-"` // 标记是否已初始化

	// Python特定字段
	FilePath   string `json:"-" yaml:"-"` // Python文件路径
	ModuleName string `json:"-" yaml:"-"` // Python模块名

	// Hook函数 - Python版本
	OnNotCommandReceived func(ctx *MsgContext, msg *Message)                        `json:"-" yaml:"-"` // 指令过滤后剩下的
	OnCommandOverride    func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool `json:"-" yaml:"-"` // 覆盖指令行为

	OnCommandReceived   func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) `json:"-" yaml:"-"`
	OnMessageReceived   func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"`
	OnMessageSend       func(ctx *MsgContext, msg *Message, flag string)      `json:"-" yaml:"-"`
	OnMessageDeleted    func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"`
	OnMessageEdit       func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"`
	OnGroupJoined       func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"`
	OnGroupMemberJoined func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"`
	OnGuildJoined       func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"`
	OnBecomeFriend      func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"`
	OnPoke              func(ctx *MsgContext, event *events.PokeEvent)        `json:"-" yaml:"-"` // 戳一戳
	OnGroupLeave        func(ctx *MsgContext, event *events.GroupLeaveEvent)  `json:"-" yaml:"-"` // 群成员被踢出

	// 新增的hook点
	OnUserJoined         func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"` // 用户加入
	OnUserLeft           func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"` // 用户离开
	OnGroupCreated       func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"` // 群创建
	OnGroupDestroyed     func(ctx *MsgContext, msg *Message)                   `json:"-" yaml:"-"` // 群解散
	OnCommandExecuted    func(ctx *MsgContext, cmdArgs *CmdArgs, result interface{}) `json:"-" yaml:"-"` // 命令执行后
	OnDiceRoll           func(ctx *MsgContext, expr string, result int)        `json:"-" yaml:"-"` // 骰点结果
	OnPluginLoad         func()                                                `json:"-" yaml:"-"` // 插件加载时
	OnPluginUnload       func()                                                `json:"-" yaml:"-"` // 插件卸载时
	OnConfigChanged      func(key string, oldValue, newValue interface{})     `json:"-" yaml:"-"` // 配置变更
	OnDatabaseOperation  func(operation string, table string, data interface{}) `json:"-" yaml:"-"` // 数据库操作

	GetDescText func(i *PythonExtInfo) string `json:"-" yaml:"-"`
	IsLoaded    bool                          `json:"-" yaml:"-"`
	OnLoad      func()                        `json:"-" yaml:"-"`
	OnUnload    func()                        `json:"-" yaml:"-"`
}

// PythonExtensionManager Python扩展管理器
type PythonExtensionManager struct {
	extensions      map[string]*PythonExtInfo
	mu              sync.RWMutex
	pythonInitialized bool
	pythonExecutable string // Python可执行文件路径
}

// GlobalPythonManager 全局Python管理器
var GlobalPythonManager *PythonExtensionManager

func init() {
	GlobalPythonManager = &PythonExtensionManager{
		extensions: make(map[string]*PythonExtInfo),
	}
}

// InitializePython 初始化Python环境
func (pem *PythonExtensionManager) InitializePython() error {
	if pem.pythonInitialized {
		return nil
	}

	// 查找Python 3.11或更高版本
	pythonCmds := []string{"python3.11", "python3.12", "python3.13", "python3"}
	for _, cmd := range pythonCmds {
		if path, err := exec.LookPath(cmd); err == nil {
			// 检查Python版本
			if version, err := getPythonVersion(path); err == nil && strings.HasPrefix(version, "3.") {
				pem.pythonExecutable = path
				pem.pythonInitialized = true
				return nil
			}
		}
	}

	return fmt.Errorf("Python 3.11+ not found")
}

// PythonExecutable 返回已初始化的 Python 可执行路径（必要时尝试初始化）
func (pem *PythonExtensionManager) PythonExecutable() string {
	if !pem.pythonInitialized {
		_ = pem.InitializePython()
	}
	return pem.pythonExecutable
}

// getPythonVersion 获取Python版本
func getPythonVersion(pythonPath string) (string, error) {
	cmd := exec.Command(pythonPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(output))
	if strings.HasPrefix(version, "Python ") {
		return strings.TrimPrefix(version, "Python "), nil
	}
	return version, nil
}

// LoadPythonExtension 加载Python扩展
func (d *Dice) LoadPythonExtension(path string) error {
	// 确保Python已初始化
	if err := GlobalPythonManager.InitializePython(); err != nil {
		return err
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("Python extension file not found: %s", path)
	}

	// 获取模块名
	moduleName := filepath.Base(path)
	if filepath.Ext(moduleName) == ".py" {
		moduleName = moduleName[:len(moduleName)-3]
	}

	// 创建Python扩展信息
	extInfo := &PythonExtInfo{
		Name:            moduleName,
		Version:         "1.0.0",
		SourcePath:      path,
		IsPythonExt:     true,
		FilePath:        path,
		ModuleName:      moduleName,
		AutoActive:      true, // 自动激活
		ActiveOnPrivate: true, // 在私聊中激活
	}

	// 尝试从Python文件中读取元信息
	if err := loadPythonExtensionMeta(extInfo); err != nil {
		if d.Logger != nil {
			d.Logger.Warnf("Failed to load Python extension metadata: %v", err)
		}
	}

	// 绑定hook函数（这里我们使用外部进程调用，所以不需要实际绑定）
	// hook函数将在调用时动态执行

	// 注册扩展
	d.RegisterPythonExtension(extInfo)

	if d.Logger != nil {
		d.Logger.Infof("Loaded Python extension: %s", extInfo.Name)
	}
	return nil
}

// loadPythonExtensionMeta 从Python文件中加载元信息
func loadPythonExtensionMeta(ext *PythonExtInfo) error {
	// 创建临时Python脚本来提取元信息
	tempScript := fmt.Sprintf(`
import sys
import ast
import inspect

# 添加扩展文件所在目录到路径
import os
ext_dir = os.path.dirname(r"%s")
if ext_dir not in sys.path:
    sys.path.insert(0, ext_dir)

try:
    # 动态导入模块
    module_name = os.path.basename(r"%s")
    if module_name.endswith('.py'):
        module_name = module_name[:-3]
    
    module = __import__(module_name)
    
    # 提取元信息
    meta = {}
    if hasattr(module, '__name__'):
        meta['name'] = module.__name__
    if hasattr(module, '__version__'):
        meta['version'] = module.__version__
    if hasattr(module, '__author__'):
        meta['author'] = module.__author__
    if hasattr(module, '__description__'):
        meta['brief'] = module.__description__
    
    # 提取支持的命令
    commands = []
    if hasattr(module, '__commands__'):
        commands = module.__commands__ if isinstance(module.__commands__, list) else []
    
    meta['commands'] = commands
    
    import json
    print(json.dumps(meta))
except Exception as e:
    print('{"error": "' + str(e) + '"}')
`, ext.SourcePath, ext.SourcePath)

	// 执行Python脚本
	cmd := exec.Command(GlobalPythonManager.pythonExecutable, "-c", tempScript)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// 解析JSON结果
	var meta map[string]interface{}
	if err := json.Unmarshal(output, &meta); err != nil {
		return err
	}

	if errorMsg, exists := meta["error"]; exists {
		return fmt.Errorf("Python error: %v", errorMsg)
	}

	// 更新扩展信息
	if name, ok := meta["name"].(string); ok {
		ext.Name = name
	}
	if version, ok := meta["version"].(string); ok {
		ext.Version = version
	}
	if author, ok := meta["author"].(string); ok {
		ext.Author = author
	}
	if brief, ok := meta["brief"].(string); ok {
		ext.Brief = brief
	}

	// 注册支持的命令
	if commands, ok := meta["commands"].([]interface{}); ok {
		if ext.CmdMap == nil {
			ext.CmdMap = make(CmdMapCls)
		}
		for _, cmdInterface := range commands {
			if cmdStr, ok := cmdInterface.(string); ok {
				// 为每个命令创建一个命令项
				cmdName := cmdStr
				extInfoPtr := ext
				cmdItem := &CmdItemInfo{
					Name:      cmdStr,
					ShortHelp: "Python extension command: " + cmdStr,
					Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
						// 调用Python的on_command_received hook并获取返回值
						reply := GlobalPythonManager.callPythonHookWithReply(extInfoPtr, "on_command_received", ctx, msg, cmdArgs)
						
						// 如果有返回值，发送给用户
						if reply != "" {
							ReplyToSender(ctx, msg, reply)
						}
						
						return CmdExecuteResult{Solved: true}
					},
				}
				ext.CmdMap[cmdName] = cmdItem
			}
		}
	}

	return nil
}

// bindPythonHooks 绑定Python的hook函数
func (pem *PythonExtensionManager) bindPythonHooks(ext *PythonExtInfo) {
	// 对于外部进程模式，我们不需要预绑定hook函数
	// hook函数将在调用时动态执行
}

// callPythonHook 调用Python hook函数
func (pem *PythonExtensionManager) callPythonHook(ext *PythonExtInfo, hookName string, args ...interface{}) {
	if ext == nil || ext.FilePath == "" {
		return
	}

	// 构建调用参数
	callArgs := map[string]interface{}{
		"hook_name": hookName,
		"args":      args,
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(callArgs)
	if err != nil {
		fmt.Printf("Failed to marshal hook call args: %v", err)
		return
	}

	// 调用Python脚本
	cmd := exec.Command(pem.pythonExecutable, "-c", fmt.Sprintf(`
import sys
import json
import importlib.util

# 加载模块
try:
    spec = importlib.util.spec_from_file_location("%s", "%s")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    
    # 获取hook函数
    hook_func = getattr(module, "%s", None)
    if hook_func and callable(hook_func):
        # 解析参数
        call_data = json.loads(sys.argv[1])
        hook_args = call_data.get("args", [])
        
        # 确保hook_args是列表
        if not isinstance(hook_args, list):
            hook_args = []
        
        # 调用函数
        if len(hook_args) > 0:
            hook_func(*hook_args)
        else:
            hook_func()
    else:
        print(f"Hook function {hook_func} not found or not callable", file=sys.stderr)
except Exception as e:
    print(f"Error calling hook: {e}", file=sys.stderr)
    sys.exit(1)
`, ext.ModuleName, ext.FilePath, hookName), string(jsonData))

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to call Python hook %s: %v, output: %s", hookName, err, string(output))
	}
}

// callPythonHookWithCmdArgs 调用带CmdArgs的Python hook函数
func (pem *PythonExtensionManager) callPythonHookWithCmdArgs(ext *PythonExtInfo, hookName string, ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	if ext == nil || ext.FilePath == "" {
		return
	}

	// 构建调用参数
	callArgs := map[string]interface{}{
		"hook_name": hookName,
		"args": []interface{}{
			pem.serializeMsgContext(ctx),
			pem.serializeMessage(msg),
			pem.serializeCmdArgs(cmdArgs),
		},
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(callArgs)
	if err != nil {
		fmt.Printf("Failed to marshal hook call args: %v", err)
		return
	}

	// 调用Python脚本
	cmd := exec.Command(pem.pythonExecutable, "-c", fmt.Sprintf(`
import sys
import json
import importlib.util

# 加载模块
try:
    spec = importlib.util.spec_from_file_location("%s", "%s")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    
    # 获取hook函数
    hook_func = getattr(module, "%s", None)
    if hook_func and callable(hook_func):
        # 解析参数
        call_data = json.loads(sys.argv[1])
        hook_args = call_data.get("args", [])
        
        # 反序列化参数
        ctx_data = hook_args[0]
        msg_data = hook_args[1]
        cmd_args_data = hook_args[2]
        
        # 创建Python对象并调用
        # 这里需要根据实际的Python扩展API来构造对象
        hook_func(ctx_data, msg_data, cmd_args_data)
    else:
        print(f"Hook function {hook_func} not found or not callable", file=sys.stderr)
except Exception as e:
    print(f"Error calling hook: {e}", file=sys.stderr)
    sys.exit(1)
`, ext.ModuleName, ext.FilePath, hookName), string(jsonData))

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to call Python hook %s: %v, output: %s", hookName, err, string(output))
	}
}

// callPythonHookWithResult 调用带result的Python hook函数
func (pem *PythonExtensionManager) callPythonHookWithResult(ext *PythonExtInfo, hookName string, ctx *MsgContext, cmdArgs *CmdArgs, result interface{}) {
	if ext == nil || ext.FilePath == "" {
		return
	}

	// 构建调用参数
	callArgs := map[string]interface{}{
		"hook_name": hookName,
		"args": []interface{}{
			pem.serializeMsgContext(ctx),
			pem.serializeCmdArgs(cmdArgs),
			result,
		},
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(callArgs)
	if err != nil {
		fmt.Printf("Failed to marshal hook call args: %v", err)
		return
	}

	// 调用Python脚本
	cmd := exec.Command(pem.pythonExecutable, "-c", fmt.Sprintf(`
import sys
import json
import importlib.util

# 加载模块
try:
    spec = importlib.util.spec_from_file_location("%s", "%s")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    
    # 获取hook函数
    hook_func = getattr(module, "%s", None)
    if hook_func and callable(hook_func):
        # 解析参数
        call_data = json.loads(sys.argv[1])
        hook_args = call_data.get("args", [])
        
        # 调用函数
        hook_func(*hook_args)
    else:
        print(f"Hook function {hook_func} not found or not callable", file=sys.stderr)
except Exception as e:
    print(f"Error calling hook: {e}", file=sys.stderr)
    sys.exit(1)
`, ext.ModuleName, ext.FilePath, hookName), string(jsonData))

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to call Python hook %s: %v, output: %s", hookName, err, string(output))
	}
}

// callPythonHookWithDiceRoll 调用带dice roll的Python hook函数
func (pem *PythonExtensionManager) callPythonHookWithDiceRoll(ext *PythonExtInfo, hookName string, ctx *MsgContext, expr string, result int) {
	if ext == nil || ext.FilePath == "" {
		return
	}

	// 构建调用参数
	callArgs := map[string]interface{}{
		"hook_name": hookName,
		"args": []interface{}{
			pem.serializeMsgContext(ctx),
			expr,
			result,
		},
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(callArgs)
	if err != nil {
		fmt.Printf("Failed to marshal hook call args: %v", err)
		return
	}

	// 调用Python脚本
	cmd := exec.Command(pem.pythonExecutable, "-c", fmt.Sprintf(`
import sys
import json
import importlib.util

# 加载模块
try:
    spec = importlib.util.spec_from_file_location("%s", "%s")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    
    # 获取hook函数
    hook_func = getattr(module, "%s", None)
    if hook_func and callable(hook_func):
        # 解析参数
        call_data = json.loads(sys.argv[1])
        hook_args = call_data.get("args", [])
        
        # 调用函数
        hook_func(*hook_args)
    else:
        print(f"Hook function {hook_func} not found or not callable", file=sys.stderr)
except Exception as e:
    print(f"Error calling hook: {e}", file=sys.stderr)
    sys.exit(1)
`, ext.ModuleName, ext.FilePath, hookName), string(jsonData))

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to call Python hook %s: %v, output: %s", hookName, err, string(output))
	}
}
// RegisterPythonExtension 注册Python扩展
func (d *Dice) RegisterPythonExtension(extInfo *PythonExtInfo) {
	GlobalPythonManager.mu.Lock()
	defer GlobalPythonManager.mu.Unlock()

	extInfo.dice = d
	GlobalPythonManager.extensions[extInfo.Name] = extInfo

	// 同时注册为普通的ExtInfo以兼容现有系统
	normalExt := &ExtInfo{
		Name:         extInfo.Name,
		Aliases:      extInfo.Aliases,
		Version:      extInfo.Version,
		AutoActive:   extInfo.AutoActive,
		CmdMap:       extInfo.CmdMap,
		Brief:        extInfo.Brief,
		ActiveOnPrivate: extInfo.ActiveOnPrivate,
		DefaultSetting: extInfo.DefaultSetting,
		Author:       extInfo.Author,
		ConflictWith: extInfo.ConflictWith,
		Official:     extInfo.Official,
		ActiveWith:   extInfo.ActiveWith,
		dice:         d,
		IsJsExt:      false, // 不是JS扩展
	}

	// 设置Python扩展的hook函数
	extInfo.OnMessageReceived = func(ctx *MsgContext, msg *Message) {
		// 序列化参数
		ctxData := GlobalPythonManager.serializeMsgContext(ctx)
		msgData := GlobalPythonManager.serializeMessage(msg)
		GlobalPythonManager.callPythonHook(extInfo, "on_message_received", ctxData, msgData)
	}
	extInfo.OnCommandReceived = func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		GlobalPythonManager.callPythonHookWithCmdArgs(extInfo, "on_command_received", ctx, msg, cmdArgs)
	}
	extInfo.OnNotCommandReceived = func(ctx *MsgContext, msg *Message) {
		// 序列化参数
		ctxData := GlobalPythonManager.serializeMsgContext(ctx)
		msgData := GlobalPythonManager.serializeMessage(msg)
		GlobalPythonManager.callPythonHook(extInfo, "on_not_command_received", ctxData, msgData)
	}
	extInfo.OnCommandExecuted = func(ctx *MsgContext, cmdArgs *CmdArgs, result interface{}) {
		GlobalPythonManager.callPythonHookWithResult(extInfo, "on_command_executed", ctx, cmdArgs, result)
	}
	extInfo.OnDiceRoll = func(ctx *MsgContext, expr string, result int) {
		GlobalPythonManager.callPythonHookWithDiceRoll(extInfo, "on_dice_roll", ctx, expr, result)
	}
	extInfo.OnLoad = func() {
		GlobalPythonManager.callPythonHook(extInfo, "on_load")
	}
	extInfo.OnUnload = func() {
		GlobalPythonManager.callPythonHook(extInfo, "on_unload")
	}

	// 复制hook函数到normalExt
	normalExt.OnMessageReceived = extInfo.OnMessageReceived
	normalExt.OnCommandReceived = extInfo.OnCommandReceived
	normalExt.OnNotCommandReceived = extInfo.OnNotCommandReceived

	d.RegisterExtension(normalExt)
}

// UnloadPythonExtension 卸载Python扩展
func (d *Dice) UnloadPythonExtension(name string) error {
	GlobalPythonManager.mu.Lock()
	defer GlobalPythonManager.mu.Unlock()

	_, exists := GlobalPythonManager.extensions[name]
	if !exists {
		return fmt.Errorf("Python extension %s not found", name)
	}

	// 从注册表中移除
	delete(GlobalPythonManager.extensions, name)

	if d.Logger != nil {
		d.Logger.Infof("Unloaded Python extension: %s", name)
	}
	return nil
}

// GetPythonExtension 获取Python扩展
func (d *Dice) GetPythonExtension(name string) *PythonExtInfo {
	GlobalPythonManager.mu.RLock()
	defer GlobalPythonManager.mu.RUnlock()

	return GlobalPythonManager.extensions[name]
}

// ListPythonExtensions 列出所有Python扩展
func (d *Dice) ListPythonExtensions() []*PythonExtInfo {
	GlobalPythonManager.mu.RLock()
	defer GlobalPythonManager.mu.RUnlock()

	var exts []*PythonExtInfo
	for _, ext := range GlobalPythonManager.extensions {
		exts = append(exts, ext)
	}
	return exts
}

// 序列化函数 - 将Go对象转换为可传递给python的格式

// serializeMsgContext 将MsgContext序列化为map
func (pem *PythonExtensionManager) serializeMsgContext(ctx *MsgContext) map[string]interface{} {
	if ctx == nil {
		return nil
	}
	
	result := map[string]interface{}{
		"message_type":      ctx.MessageType,
		"is_private":        ctx.IsPrivate,
		"privilege_level":   ctx.PrivilegeLevel,
		"group_role_level":  ctx.GroupRoleLevel,
		"delegate_text":     ctx.DelegateText,
	}
	
	if ctx.Group != nil {
		result["group_id"] = ctx.Group.GroupID
		result["group_name"] = ctx.Group.GroupName
	}
	
	if ctx.Player != nil {
		result["user_id"] = ctx.Player.UserID
		result["name"] = ctx.Player.Name
	}
	
	if ctx.EndPoint != nil {
		result["platform"] = ctx.EndPoint.Platform
	}
	
	return result
}

// serializeMessage 将Message序列化为map
func (pem *PythonExtensionManager) serializeMessage(msg *Message) map[string]interface{} {
	if msg == nil {
		return nil
	}
	
	// 序列化sender信息
	senderMap := map[string]interface{}{
		"user_id":  msg.Sender.UserID,
		"nickname": msg.Sender.Nickname,
	}
	
	return map[string]interface{}{
		"message":     msg.Message,
		"sender":      senderMap,
		"time":        msg.Time,
		"group_id":    msg.GroupID,
		"guild_id":    msg.GuildID,
		"channel_id":  msg.ChannelID,
		"platform":    msg.Platform,
		"group_name":  msg.GroupName,
	}
}

// serializeCmdArgs 将CmdArgs序列化为map
func (pem *PythonExtensionManager) serializeCmdArgs(cmdArgs *CmdArgs) map[string]interface{} {
	if cmdArgs == nil {
		return nil
	}
	return map[string]interface{}{
		"command":    cmdArgs.Command,
		"args":       cmdArgs.Args,
		"raw_args":   cmdArgs.RawArgs,
		"clean_args": cmdArgs.CleanArgs,
		"raw_text":   cmdArgs.RawText,
	}
}

// callPythonHookWithReply 调用pyhook函数并获取返回值作为回复
func (pem *PythonExtensionManager) callPythonHookWithReply(ext *PythonExtInfo, hookName string, ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) string {
	if ext == nil || ext.FilePath == "" {
		return ""
	}

	// 构建调用参数
	callArgs := map[string]interface{}{
		"hook_name": hookName,
		"args": []interface{}{
			pem.serializeMsgContext(ctx),
			pem.serializeMessage(msg),
			pem.serializeCmdArgs(cmdArgs),
		},
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(callArgs)
	if err != nil {
		fmt.Printf("Failed to marshal hook call args: %v", err)
		return ""
	}

	// 调用Python脚本并获取返回值
	cmd := exec.Command(pem.pythonExecutable, "-c", fmt.Sprintf(`
import sys
import json
import importlib.util

# 加载模块
try:
    spec = importlib.util.spec_from_file_location("%s", "%s")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    
    # 获取hook函数
    hook_func = getattr(module, "%s", None)
    if hook_func and callable(hook_func):
        # 解析参数
        call_data = json.loads(sys.argv[1])
        hook_args = call_data.get("args", [])
        
        # 调用函数并获取返回值
        result = hook_func(hook_args[0], hook_args[1], hook_args[2])
        
        # 如果有返回值，打印出来
        if result:
            print(str(result))
    else:
        print("", file=sys.stderr)
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
`, ext.ModuleName, ext.FilePath, hookName), string(jsonData))

	// 执行命令并获取输出
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// SavePythonExtensionConfig 保存Python扩展配置
func (d *Dice) SavePythonExtensionConfig(name string, config map[string]interface{}) error {
	GlobalPythonManager.mu.Lock()
	defer GlobalPythonManager.mu.Unlock()

	ext, exists := GlobalPythonManager.extensions[name]
	if !exists {
		return fmt.Errorf("Python extension %s not found", name)
	}

	// 将配置保存到Storage字段
	ext.Storage = config

	// 持久化配置到文件
	configDir := filepath.Join(d.BaseConfig.DataDir, "extensions", "configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	configFile := filepath.Join(configDir, fmt.Sprintf("%s.json", name))
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	if d.Logger != nil {
		d.Logger.Infof("Saved config for Python extension: %s", name)
	}

	return nil
}

// CallPythonMethod 调用Python扩展的自定义方法
func (d *Dice) CallPythonMethod(name string, method string, arguments map[string]interface{}) (interface{}, error) {
	GlobalPythonManager.mu.RLock()
	ext, exists := GlobalPythonManager.extensions[name]
	GlobalPythonManager.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("Python extension %s not found", name)
	}

	if ext.FilePath == "" {
		return nil, fmt.Errorf("Python extension %s has no file path", name)
	}

	// 构建调用参数
	callData := map[string]interface{}{
		"method":    method,
		"arguments": arguments,
	}

	jsonData, err := json.Marshal(callData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal call data: %v", err)
	}

	// 构建Python调用脚本
	pythonScript := fmt.Sprintf(`
import sys
import json
import importlib.util

try:
    # 加载模块
    spec = importlib.util.spec_from_file_location("%s", "%s")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    
    # 解析调用数据
    call_data = json.loads(sys.argv[1])
    method_name = call_data.get("method")
    arguments = call_data.get("arguments", {})
    
    # 获取方法
    method_func = getattr(module, method_name, None)
    if not method_func or not callable(method_func):
        print(json.dumps({"error": "Method not found or not callable"}))
        sys.exit(1)
    
    # 调用方法
    result = method_func(**arguments)
    
    # 返回结果
    print(json.dumps({"success": True, "result": result}))
    
except Exception as e:
    print(json.dumps({"error": str(e)}))
    sys.exit(1)
`, ext.ModuleName, ext.FilePath)

	// 执行py命令
	cmd := exec.Command(GlobalPythonManager.pythonExecutable, "-c", pythonScript, string(jsonData))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute Python method: %v, output: %s", err, string(output))
	}

	// 解析返回结果
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Python output: %v, output: %s", err, string(output))
	}

	if errMsg, ok := result["error"]; ok {
		return nil, fmt.Errorf("Python method error: %v", errMsg)
	}

	if d.Logger != nil {
		d.Logger.Infof("Called Python method %s.%s with result: %v", name, method, result["result"])
	}

	return result["result"], nil
}
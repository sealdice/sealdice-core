# 海豹核心 Python 插件开发文档

> SealDice Core Python Extension Development Guide
> 更新日期: 2026-01-13

---

##  目录

1. [系统概述](#系统概述)
2. [代码改动清单](#代码改动清单)
3. [插件编写教程](#插件编写教程)
4. [完整示例](#完整示例)
5. [最佳实践](#最佳实践)
6. [故障排查](#故障排查)

---

## 系统概述

###  新增功能

海豹核心现在支持 Python 3.11+ 扩展插件系统，提供以下功能：

-  **自动发现加载** - 启动时自动扫描 `data/default/extensions/` 目录下的 `.py` 文件
-  **命令系统集成** - Python 命令与海豹核心命令系统集成
-  **完整生命周期** - 6个 Hook 函数支持插件的完整生命周期
-  **消息回复** - 插件可以返回消息给用户
-  **数据持久化** - 支持 JSON 格式的数据存储
-  **跨平台** - 支持所有海豹支持的平台（QQ、Discord、Kook 等）
-  **私聊自动激活** - 在私聊中自动激活设置了 `ActiveOnPrivate` 的扩展

###  架构设计

```
┌─────────────────────────────────────────────────────┐
│                   海豹核心 (Go)                      │
│  ┌───────────────────────────────────────────────┐  │
│  │          PythonExtensionManager              │  │
│  │  - 管理所有 Python 扩展                       │  │
│  │  - 处理 Hook 调用                            │  │
│  │  - 参数序列化/反序列化                        │  │
│  └───────────────────────────────────────────────┘  │
│                       ↓ ↑                            │
│  ┌───────────────────────────────────────────────┐  │
│  │         命令系统 (CmdMap)                     │  │
│  │  - 注册 Python 命令                          │  │
│  │  - 路由消息到插件                            │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
                       ↓ ↑
          通过 exec.Command 调用 Python
                       ↓ ↑
┌─────────────────────────────────────────────────────┐
│              Python 插件 (.py 文件)                  │
│  ┌───────────────────────────────────────────────┐  │
│  │  插件元数据:                                   │  │
│  │  - __name__: 插件名称                        │  │
│  │  - __version__: 版本号                       │  │
│  │  - __commands__: 支持的命令列表              │  │
│  └───────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────┐  │
│  │  Hook 函数:                                    │  │
│  │  - on_load()                                  │  │
│  │  - on_command_received()                     │  │
│  │  - on_message_received()                     │  │
│  │  - on_dice_roll()                            │  │
│  │  - on_command_executed()                     │  │
│  │  - on_unload()                               │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

---

## 代码改动清单

###  新增文件

#### 1. `dice/ext_python.go` (新建, ~780行)

**功能**: Python扩展管理器核心

**主要结构**:

```go
// Python扩展信息结构
type PythonExtInfo struct {
    Name            string
    Version         string
    AutoActive      bool
    ActiveOnPrivate bool
    CmdMap          CmdMapCls
    FilePath        string
    ModuleName      string
    // Hook函数...
}

// Python扩展管理器
type PythonExtensionManager struct {
    pythonExecutable string
    extensions       map[string]*PythonExtInfo
    mu               sync.Mutex
}
```

**核心函数**:

- `InitializePython()` - 初始化 Python 环境
- `LoadPythonExtension(path)` - 加载单个 Python 插件
- `RegisterPythonExtension(extInfo)` - 注册 Python 扩展到系统
- `callPythonHookWithReply()` - 调用 Python Hook 并获取返回值
- `serializeMsgContext()` - 序列化消息上下文
- `serializeMessage()` - 序列化消息对象
- `serializeCmdArgs()` - 序列化命令参数

**关键特性**:

- 外部进程执行 Python 代码（避免 CGo 依赖）
- JSON 格式的参数传递
- 自动提取 Python 文件的元数据（`__name__`, `__version__`, `__commands__`）
- 为每个命令创建 `CmdItemInfo` 并实现 `Solve` 函数

---

#### 2. `data/default/extensions/test_dice_plugin.py`

**功能**: 基础测试插件，演示所有 Hook 函数

**支持的命令**:

- `.hello` - 返回问候消息
- `.py` - 显示插件状态

**实现的 Hook**:

- `on_load()` - 插件加载时记录日志
- `on_message_received()` - 统计消息数量
- `on_command_received()` - 处理命令并返回回复
- `on_dice_roll()` - 检测大成功/大失败
- `on_command_executed()` - 记录命令执行
- `on_unload()` - 插件卸载时保存状态

**数据持久化**:

- 日志输出到 `python_plugin.log`
- 全局变量存储消息计数

---

#### 3. `data/default/extensions/gacha_simulator.py`

**功能**: 完整的抽卡模拟器，演示数据持久化

**支持的命令**:

- `.抽卡` / `.gacha` - 单次抽卡
- `.十连` / `.gacha10` - 十连抽卡
- `.抽卡记录` / `.gacharecord` - 查看抽卡历史

**卡池配置**:

```python
CARD_POOL = {
    "SSR": {"rate": 0.03, "cards": ["传说剑", "神话盾", "史诗法杖"]},
    "SR":  {"rate": 0.15, "cards": ["稀有剑", "精良盾", "优质法杖"]},
    "R":   {"rate": 0.82, "cards": ["普通剑", "木盾", "木杖"]},
}
```

**数据存储**:

- 位置: `data/extensions/gacha_records.json`
- 格式: 每个用户的抽卡历史记录
- 自动保存和加载

---

###  修改的文件

#### 1. `dice/dice.go`

**改动位置**: 末尾新增 `loadPythonExtensions()` 方法

**改动内容**:

```go
// 在 Init() 函数中调用 (约第388行)
if d.Logger != nil {
    d.Logger.Info("Python扩展支持：开启")
}
d.loadPythonExtensions(d.Logger)

// 新增方法 (约第1054行)
func (d *Dice) loadPythonExtensions(loggerInstance *zap.SugaredLogger) {
    extensionsDir := filepath.Join(d.BaseConfig.DataDir, "extensions")
    // ... 扫描目录并加载所有 .py 文件
}
```

**功能**:

- 在海豹核心初始化时自动加载 Python 插件
- 扫描 `data/default/extensions/` 目录
- 跳过 `__` 开头的文件（如 `__init__.py`）
- 记录加载成功的插件数量

---

#### 2. `dice/im_session.go`

**改动位置**: `GetActivatedExtList()` 函数中 (~第160-178行)

**改动内容**:

```go
// 原代码
newExtCount := 0
for _, ext := range d.ExtList {
    // ...
    if ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive) {
        // 激活扩展
    }
}

// 修改后
newExtCount := 0
isPrivateGroup := strings.HasPrefix(g.GroupID, "PG-")  // 新增
for _, ext := range d.ExtList {
    // ...
    shouldActivate := ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive)
    // 私聊群组：自动激活设置了 ActiveOnPrivate 的扩展 (新增)
    if isPrivateGroup && ext.ActiveOnPrivate {
        shouldActivate = true
    }
    if shouldActivate {
        // 激活扩展
    }
}
```

**功能**:

- 检测私聊群组（以 `PG-` 开头的 GroupID）
- 自动激活设置了 `ActiveOnPrivate: true` 的扩展
- 确保 Python 插件在私聊中可用

---

###  关键技术实现

#### 1. 命令注册机制

**位置**: `dice/ext_python.go` - `loadPythonExtensionMeta()` 函数

```go
// 注册支持的命令
if commands, ok := meta["commands"].([]interface{}); ok {
    for _, cmdInterface := range commands {
        if cmdStr, ok := cmdInterface.(string); ok {
            cmdItem := &CmdItemInfo{
                Name:      cmdStr,
                ShortHelp: "Python extension command: " + cmdStr,
                Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
                    // 调用Python的on_command_received hook并获取返回值
                    reply := GlobalPythonManager.callPythonHookWithReply(...)
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
```

**工作流程**:

1. 从 Python 文件的 `__commands__` 列表提取命令
2. 为每个命令创建 `CmdItemInfo` 对象
3. 实现 `Solve` 函数：调用 Python Hook → 获取返回值 → 发送给用户
4. 注册到扩展的 `CmdMap` 中

---

#### 2. Python 调用机制

**位置**: `dice/ext_python.go` - `callPythonHookWithReply()` 函数

```go
func (pem *PythonExtensionManager) callPythonHookWithReply(...) string {
    // 1. 序列化参数为 JSON
    callArgs := map[string]interface{}{
        "hook_name": hookName,
        "args": []interface{}{
            pem.serializeMsgContext(ctx),
            pem.serializeMessage(msg),
            pem.serializeCmdArgs(cmdArgs),
        },
    }
    jsonData, _ := json.Marshal(callArgs)
  
    // 2. 构建 Python 代码
    pythonCode := `
import sys, json, importlib.util
spec = importlib.util.spec_from_file_location("module", "path.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

hook_func = getattr(module, "on_command_received", None)
call_data = json.loads(sys.argv[1])
result = hook_func(*call_data["args"])
if result:
    print(str(result))
`
  
    // 3. 执行 Python 并获取输出
    cmd := exec.Command("python3", "-c", pythonCode, string(jsonData))
    output, _ := cmd.Output()
    return strings.TrimSpace(string(output))
}
```

**优点**:

- 无需 CGo，不依赖 Python C API
- 进程隔离，Python 崩溃不影响核心
- 跨平台兼容性好

---

#### 3. 数据序列化

**上下文序列化**:

```go
func (pem *PythonExtensionManager) serializeMsgContext(ctx *MsgContext) map[string]interface{} {
    return map[string]interface{}{
        "user_id":    ctx.Player.UserID,
        "group_id":   ctx.Group.GroupID,
        "is_private": ctx.IsPrivate,
        "dice_id":    ctx.EndPoint.UserID,
        // ... 更多字段
    }
}
```

**消息序列化**:

```go
func (pem *PythonExtensionManager) serializeMessage(msg *Message) map[string]interface{} {
    return map[string]interface{}{
        "message":    msg.Message,
        "sender":     map[string]interface{}{
            "user_id":  msg.Sender.UserID,
            "nickname": msg.Sender.Nickname,
        },
        "platform":   msg.Platform,
        "group_id":   msg.GroupID,
        // ... 更多字段
    }
}
```

---

## 插件编写教程

###  快速开始

#### 第一步：创建插件文件

在 `data/default/extensions/` 目录下创建 `my_plugin.py`：

```python
"""
我的第一个海豹Python插件
"""

# 插件元数据（必需）
__name__ = "我的插件"
__version__ = "1.0.0"
__author__ = "你的名字"
__description__ = "这是一个示例插件"
__commands__ = ["hello", "test"]  # 支持的命令列表

def on_load():
    """插件加载时调用"""
    print(" 插件已加载！")

def on_command_received(ctx, msg, cmd_args):
    """处理命令"""
    command = cmd_args.get('command', '')
  
    if command == 'hello':
        return " 你好！这是我的第一个Python插件！"
    elif command == 'test':
        user = msg.get('sender', {}).get('nickname', '未知用户')
        return f" 测试成功！\n欢迎 {user}！"
  
    return None  # 不返回消息
```

#### 第二步：重启海豹核心

```bash
# 如果核心正在运行，重启它
pkill -f "./sealdice"
./sealdice
```

#### 第三步：测试插件

在海豹 UI 或任意平台发送：

```
.hello
.test
```

---

###  插件元数据详解

#### 必需字段

| 字段             | 类型 | 说明         | 示例                  |
| ---------------- | ---- | ------------ | --------------------- |
| `__name__`     | str  | 插件显示名称 | `"抽卡模拟器"`      |
| `__version__`  | str  | 版本号       | `"1.0.0"`           |
| `__commands__` | list | 支持的命令   | `["抽卡", "gacha"]` |

#### 可选字段

| 字段                | 类型 | 说明     | 示例                |
| ------------------- | ---- | -------- | ------------------- |
| `__author__`      | str  | 作者名称 | `"SealDice Team"` |
| `__description__` | str  | 插件描述 | `"一个抽卡系统"`  |

#### 示例

```python
__name__ = "我的插件"
__version__ = "1.0.0"
__author__ = "张三"
__description__ = "这是一个非常酷的插件"
__commands__ = [
    "cmd1",      # 主命令
    "命令2",     # 支持中文
    "alias",     # 别名
]
```

---

###  Hook 函数详解

#### 1. `on_load()`

**调用时机**: 插件加载时（核心启动时）

**参数**: 无

**返回值**: 无

**用途**: 初始化资源、加载配置、创建数据文件

```python
def on_load():
    """插件加载时调用"""
    print("插件正在加载...")
  
    # 初始化数据
    global data
    data = load_config()
  
    # 创建必要的目录
    import os
    os.makedirs("data/my_plugin", exist_ok=True)
  
    print(f"插件 {__name__} v{__version__} 已就绪！")
```

---

#### 2. `on_command_received(ctx, msg, cmd_args)`

**调用时机**: 用户发送插件注册的命令时

**参数**:

- `ctx` (dict): 消息上下文

  ```python
  {
      "user_id": "QQ:12345",       # 用户ID
      "group_id": "QQ-Group:67890", # 群组ID
      "is_private": False,          # 是否私聊
      "dice_id": "QQ:骰子ID"        # 骰子ID
  }
  ```
- `msg` (dict): 消息对象

  ```python
  {
      "message": ".hello world",     # 完整消息
      "sender": {
          "user_id": "QQ:12345",
          "nickname": "用户昵称"
      },
      "platform": "QQ",              # 平台
      "group_id": "QQ-Group:67890",  # 群组ID
      "time": 1234567890             # 时间戳
  }
  ```
- `cmd_args` (dict): 命令参数

  ```python
  {
      "command": "hello",            # 命令名
      "args": ["world"],             # 参数列表
      "raw_args": "world",           # 原始参数字符串
  }
  ```

**返回值**:

- `str`: 回复消息（会发送给用户）
- `None`: 不回复

**示例**:

```python
def on_command_received(ctx, msg, cmd_args):
    """处理命令"""
    command = cmd_args.get('command', '')
    args = cmd_args.get('args', [])
  
    # 获取用户信息
    user_id = ctx.get('user_id', 'unknown')
    nickname = msg.get('sender', {}).get('nickname', '未知')
  
    if command == 'hello':
        if args:
            return f"你好，{args[0]}！"
        else:
            return f"你好，{nickname}！"
  
    elif command == 'info':
        return f" 命令信息:\n" \
               f"  命令: {command}\n" \
               f"  参数: {args}\n" \
               f"  用户: {nickname} ({user_id})\n" \
               f"  平台: {msg.get('platform', '未知')}"
  
    return None
```

---

#### 3. `on_message_received(ctx, msg)`

**调用时机**: 收到任何消息时（包括非命令消息）

**参数**:

- `ctx` (dict): 消息上下文
- `msg` (dict): 消息对象

**返回值**: 无

**用途**: 消息监听、关键词触发、统计分析

```python
# 全局变量
message_count = 0
keyword_triggers = {}

def on_message_received(ctx, msg):
    """监听所有消息"""
    global message_count, keyword_triggers
  
    message_count += 1
    message_text = msg.get('message', '')
  
    # 关键词统计
    keywords = ['骰子', '投掷', '检定']
    for keyword in keywords:
        if keyword in message_text:
            keyword_triggers[keyword] = keyword_triggers.get(keyword, 0) + 1
  
    # 记录日志
    print(f"消息 #{message_count}: {message_text[:50]}")
```

---

#### 4. `on_dice_roll(ctx, expr, result)`

**调用时机**: 骰子投掷时

**参数**:

- `ctx` (dict): 消息上下文
- `expr` (str): 骰子表达式，如 `"1d100"`, `"3d6+5"`
- `result` (int): 骰点结果

**返回值**: 无

**用途**: 骰点分析、大成功/大失败检测、统计

```python
def on_dice_roll(ctx, expr, result):
    """处理骰点"""
    user_id = ctx.get('user_id', 'unknown')
  
    print(f" {user_id} 投掷了 {expr}，结果: {result}")
  
    # 检测特殊结果
    if result >= 95:
        print(f"   大成功！")
    elif result <= 5:
        print(f"   大失败！")
  
    # 可以在这里记录统计数据
    # save_dice_stats(user_id, expr, result)
```

---

#### 5. `on_command_executed(ctx, cmd_args, result)`

**调用时机**: 命令执行完成后

**参数**:

- `ctx` (dict): 消息上下文
- `cmd_args` (dict): 命令参数
- `result`: 命令执行结果

**返回值**: 无

**用途**: 命令后处理、日志记录、性能监控

```python
import time

execution_times = {}

def on_command_executed(ctx, cmd_args, result):
    """命令执行后"""
    command = cmd_args.get('command', '')
  
    # 记录执行
    if command not in execution_times:
        execution_times[command] = []
  
    execution_times[command].append(time.time())
  
    print(f" 命令 '{command}' 已执行，结果: {result}")
```

---

#### 6. `on_unload()`

**调用时机**: 插件卸载时（核心关闭时）

**参数**: 无

**返回值**: 无

**用途**: 清理资源、保存数据、关闭连接

```python
def on_unload():
    """插件卸载时调用"""
    print("正在卸载插件...")
  
    # 保存数据
    save_all_data()
  
    # 清理临时文件
    import os
    if os.path.exists("temp_data.tmp"):
        os.remove("temp_data.tmp")
  
    print(f"插件已处理 {message_count} 条消息")
    print("再见！")
```

---

###  数据持久化

#### 方法一：JSON 文件

```python
import json
import os

DATA_FILE = "data/extensions/my_plugin_data.json"

def load_data():
    """加载数据"""
    if os.path.exists(DATA_FILE):
        with open(DATA_FILE, 'r', encoding='utf-8') as f:
            return json.load(f)
    return {}

def save_data(data):
    """保存数据"""
    os.makedirs(os.path.dirname(DATA_FILE), exist_ok=True)
    with open(DATA_FILE, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

# 使用示例
user_data = {}

def on_load():
    global user_data
    user_data = load_data()

def on_command_received(ctx, msg, cmd_args):
    command = cmd_args.get('command', '')
    user_id = ctx.get('user_id', 'unknown')
  
    if command == 'save':
        user_data[user_id] = {
            'nickname': msg.get('sender', {}).get('nickname'),
            'last_command': command,
            'timestamp': msg.get('time')
        }
        save_data(user_data)
        return " 数据已保存！"

def on_unload():
    save_data(user_data)
```

---

#### 方法二：文本日志

```python
from datetime import datetime

LOG_FILE = "data/extensions/my_plugin.log"

def log(message):
    """写入日志"""
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    with open(LOG_FILE, 'a', encoding='utf-8') as f:
        f.write(f"[{timestamp}] {message}\n")

def on_command_received(ctx, msg, cmd_args):
    command = cmd_args.get('command', '')
    user = msg.get('sender', {}).get('nickname', '未知')
  
    log(f"用户 {user} 执行了命令: {command}")
  
    return f"命令已记录到日志文件"
```

---

###  实用功能示例

#### 示例 1：签到系统

```python
import json
import os
from datetime import datetime, date

__name__ = "签到系统"
__version__ = "1.0.0"
__commands__ = ["签到", "签到榜"]

DATA_FILE = "data/extensions/checkin_data.json"
checkin_data = {}

def load_data():
    if os.path.exists(DATA_FILE):
        with open(DATA_FILE, 'r', encoding='utf-8') as f:
            return json.load(f)
    return {}

def save_data():
    os.makedirs(os.path.dirname(DATA_FILE), exist_ok=True)
    with open(DATA_FILE, 'w', encoding='utf-8') as f:
        json.dump(checkin_data, f, ensure_ascii=False, indent=2)

def on_load():
    global checkin_data
    checkin_data = load_data()

def on_command_received(ctx, msg, cmd_args):
    command = cmd_args.get('command', '')
    user_id = ctx.get('user_id', 'unknown')
    nickname = msg.get('sender', {}).get('nickname', '用户')
  
    if command == '签到':
        today = date.today().isoformat()
  
        if user_id not in checkin_data:
            checkin_data[user_id] = {
                'nickname': nickname,
                'days': [],
                'total': 0
            }
  
        user = checkin_data[user_id]
  
        if today in user['days']:
            return f" {nickname}，你今天已经签到过了！\n 连续签到: {len(user['days'])} 天"
  
        user['days'].append(today)
        user['total'] += 1
        user['nickname'] = nickname
        save_data()
  
        return f" {nickname} 签到成功！\n" \
               f" 连续签到: {len(user['days'])} 天\n" \
               f" 累计签到: {user['total']} 天"
  
    elif command == '签到榜':
        if not checkin_data:
            return " 还没有人签到过哦~"
  
        # 排序
        sorted_users = sorted(
            checkin_data.items(),
            key=lambda x: x[1]['total'],
            reverse=True
        )[:10]
  
        reply = " 签到排行榜（TOP 10）\n\n"
        for i, (uid, data) in enumerate(sorted_users, 1):
            emoji = "" if i == 1 else "" if i == 2 else "" if i == 3 else f"{i}."
            reply += f"{emoji} {data['nickname']}: {data['total']} 天\n"
  
        return reply
  
    return None

def on_unload():
    save_data()
```

---

#### 示例 2：关键词回复

```python
__name__ = "关键词回复"
__version__ = "1.0.0"
__commands__ = []  # 不注册命令，只监听消息

# 关键词映射
KEYWORDS = {
    "你好": ["你好啊！", "欢迎！", "Hi~"],
    "骰子": ["叫我吗？", "我在这里！"],
    "谢谢": ["不客气~", "很高兴帮到你！"],
    "再见": ["再见！", "下次见~", "慢走！"],
}

import random

def on_message_received(ctx, msg):
    """监听消息中的关键词"""
    message = msg.get('message', '')
  
    # 检查是否包含关键词
    for keyword, replies in KEYWORDS.items():
        if keyword in message:
            reply = random.choice(replies)
            # 注意：on_message_received 不能直接回复
            # 需要记录并在其他地方处理
            print(f"检测到关键词 '{keyword}'，回复: {reply}")
```

---

#### 示例 3：掷骰统计

```python
import json
import os

__name__ = "掷骰统计"
__version__ = "1.0.0"
__commands__ = ["骰子统计", "我的统计"]

DATA_FILE = "data/extensions/dice_stats.json"
dice_stats = {}

def load_data():
    if os.path.exists(DATA_FILE):
        with open(DATA_FILE, 'r', encoding='utf-8') as f:
            return json.load(f)
    return {}

def save_data():
    os.makedirs(os.path.dirname(DATA_FILE), exist_ok=True)
    with open(DATA_FILE, 'w', encoding='utf-8') as f:
        json.dump(dice_stats, f, ensure_ascii=False, indent=2)

def on_load():
    global dice_stats
    dice_stats = load_data()

def on_dice_roll(ctx, expr, result):
    """记录每次骰点"""
    user_id = ctx.get('user_id', 'unknown')
  
    if user_id not in dice_stats:
        dice_stats[user_id] = {
            'total_rolls': 0,
            'results': [],
            'critical_success': 0,
            'critical_fail': 0
        }
  
    stats = dice_stats[user_id]
    stats['total_rolls'] += 1
    stats['results'].append(result)
  
    if result >= 95:
        stats['critical_success'] += 1
    elif result <= 5:
        stats['critical_fail'] += 1
  
    # 只保留最近100次结果
    if len(stats['results']) > 100:
        stats['results'] = stats['results'][-100:]

def on_command_received(ctx, msg, cmd_args):
    command = cmd_args.get('command', '')
    user_id = ctx.get('user_id', 'unknown')
    nickname = msg.get('sender', {}).get('nickname', '用户')
  
    if command in ['骰子统计', '我的统计']:
        if user_id not in dice_stats:
            return f"{nickname}，你还没有掷过骰子哦~"
  
        stats = dice_stats[user_id]
        results = stats['results']
  
        if not results:
            return f"{nickname}，统计数据为空"
  
        avg = sum(results) / len(results)
  
        reply = f" {nickname} 的掷骰统计\n\n"
        reply += f"总掷骰次数: {stats['total_rolls']}\n"
        reply += f"平均点数: {avg:.2f}\n"
        reply += f" 大成功: {stats['critical_success']} 次\n"
        reply += f" 大失败: {stats['critical_fail']} 次\n"
        reply += f"最近点数: {results[-5:]}"
  
        return reply
  
    return None

def on_unload():
    save_data()
```

---

## 完整示例

### 完整功能的待办事项插件

```python
"""
海豹骰子 - 待办事项插件
功能：添加待办、查看待办、完成待办
"""

import json
import os
from datetime import datetime

# ===== 插件元数据 =====
__name__ = "待办事项"
__version__ = "1.0.0"
__author__ = "SealDice Team"
__description__ = "一个简单的待办事项管理插件"
__commands__ = [
    "添加待办",
    "待办列表",
    "完成待办",
    "删除待办",
    "todo",
    "todolist",
]

# ===== 数据存储 =====
DATA_FILE = "data/extensions/todo_data.json"
todo_data = {}

# ===== 辅助函数 =====
def load_data():
    """加载数据"""
    if os.path.exists(DATA_FILE):
        try:
            with open(DATA_FILE, 'r', encoding='utf-8') as f:
                return json.load(f)
        except:
            return {}
    return {}

def save_data():
    """保存数据"""
    try:
        os.makedirs(os.path.dirname(DATA_FILE), exist_ok=True)
        with open(DATA_FILE, 'w', encoding='utf-8') as f:
            json.dump(todo_data, f, ensure_ascii=False, indent=2)
    except Exception as e:
        print(f"保存数据失败: {e}")

def log(msg):
    """日志输出"""
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] [待办插件] {msg}")

# ===== Hook 函数 =====
def on_load():
    """插件加载"""
    global todo_data
    todo_data = load_data()
    log(f"插件已加载，当前有 {len(todo_data)} 个用户的待办数据")

def on_unload():
    """插件卸载"""
    save_data()
    log("插件已卸载，数据已保存")

def on_command_received(ctx, msg, cmd_args):
    """处理命令"""
    command = cmd_args.get('command', '').lower()
    args = cmd_args.get('args', [])
    user_id = ctx.get('user_id', 'unknown')
    nickname = msg.get('sender', {}).get('nickname', '用户')
  
    log(f"收到命令: {command} from {nickname}")
  
    # 确保用户数据存在
    if user_id not in todo_data:
        todo_data[user_id] = {
            'nickname': nickname,
            'todos': []
        }
  
    user_todos = todo_data[user_id]['todos']
  
    # 添加待办
    if command in ['添加待办', 'todo']:
        if not args:
            return " 用法: .添加待办 <内容>\n例如: .添加待办 完成作业"
  
        todo_text = ' '.join(args)
        todo_item = {
            'id': len(user_todos) + 1,
            'text': todo_text,
            'done': False,
            'created_at': datetime.now().isoformat()
        }
        user_todos.append(todo_item)
        todo_data[user_id]['nickname'] = nickname
        save_data()
  
        log(f"{nickname} 添加待办: {todo_text}")
        return f" 待办已添加！\n {todo_text}\n\n现在有 {len(user_todos)} 个待办事项"
  
    # 查看待办列表
    elif command in ['待办列表', 'todolist']:
        if not user_todos:
            return f" {nickname}，你还没有待办事项哦~\n使用 .添加待办 <内容> 来添加"
  
        reply = f" {nickname} 的待办列表\n\n"
  
        pending = [t for t in user_todos if not t['done']]
        completed = [t for t in user_todos if t['done']]
  
        if pending:
            reply += " 待完成:\n"
            for todo in pending:
                reply += f"  {todo['id']}.  {todo['text']}\n"
  
        if completed:
            reply += "\n 已完成:\n"
            for todo in completed:
                reply += f"  {todo['id']}.  {todo['text']}\n"
  
        reply += f"\n总计: {len(user_todos)} 项 (待完成: {len(pending)}, 已完成: {len(completed)})"
        return reply
  
    # 完成待办
    elif command in ['完成待办']:
        if not args:
            return " 用法: .完成待办 <编号>\n例如: .完成待办 1"
  
        try:
            todo_id = int(args[0])
            todo = next((t for t in user_todos if t['id'] == todo_id), None)
      
            if not todo:
                return f" 找不到编号为 {todo_id} 的待办事项"
      
            if todo['done']:
                return f" 待办 #{todo_id} 已经完成过了"
      
            todo['done'] = True
            todo['completed_at'] = datetime.now().isoformat()
            save_data()
      
            log(f"{nickname} 完成待办 #{todo_id}: {todo['text']}")
      
            pending_count = len([t for t in user_todos if not t['done']])
            reply = f" 太棒了！待办已完成\n {todo['text']}"
            if pending_count > 0:
                reply += f"\n\n还有 {pending_count} 个待办事项待完成"
            else:
                reply += "\n\n 所有待办都完成啦！"
            return reply
      
        except ValueError:
            return " 请输入有效的待办编号"
  
    # 删除待办
    elif command in ['删除待办']:
        if not args:
            return " 用法: .删除待办 <编号>\n例如: .删除待办 1"
  
        try:
            todo_id = int(args[0])
            todo = next((t for t in user_todos if t['id'] == todo_id), None)
      
            if not todo:
                return f" 找不到编号为 {todo_id} 的待办事项"
      
            user_todos.remove(todo)
      
            # 重新编号
            for i, t in enumerate(user_todos, 1):
                t['id'] = i
      
            save_data()
      
            log(f"{nickname} 删除待办 #{todo_id}: {todo['text']}")
            return f" 待办已删除\n {todo['text']}"
      
        except ValueError:
            return " 请输入有效的待办编号"
  
    return None
```

**使用示例**:

```
用户: .添加待办 完成Python作业
骰子:  待办已添加！
       完成Python作业
      现在有 1 个待办事项

用户: .添加待办 写插件文档
骰子:  待办已添加！
       写插件文档
      现在有 2 个待办事项

用户: .待办列表
骰子:  用户 的待办列表
  
       待完成:
        1.  完成Python作业
        2.  写插件文档
  
      总计: 2 项 (待完成: 2, 已完成: 0)

用户: .完成待办 1
骰子:  太棒了！待办已完成
       完成Python作业
  
      还有 1 个待办事项待完成
```

---

## 最佳实践

###  推荐做法

1. **错误处理**

```python
def on_command_received(ctx, msg, cmd_args):
    try:
        command = cmd_args.get('command', '')
        # 处理逻辑...
    except Exception as e:
        print(f"错误: {e}")
        return " 命令执行出错，请稍后重试"
```

2. **参数验证**

```python
def on_command_received(ctx, msg, cmd_args):
    args = cmd_args.get('args', [])
  
    if not args:
        return " 缺少必需参数\n用法: .命令 <参数>"
  
    if len(args) < 2:
        return " 参数不足\n用法: .命令 <参数1> <参数2>"
```

3. **定期保存**

```python
save_counter = 0

def on_command_received(ctx, msg, cmd_args):
    global save_counter
  
    # 处理命令...
    save_counter += 1
  
    # 每10次操作保存一次
    if save_counter >= 10:
        save_data()
        save_counter = 0
```

4. **日志记录**

```python
import logging

logging.basicConfig(
    filename='data/extensions/my_plugin.log',
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

def on_command_received(ctx, msg, cmd_args):
    logger.info(f"Command: {cmd_args.get('command')} from {ctx.get('user_id')}")
```

5. **资源清理**

```python
def on_unload():
    """卸载时清理资源"""
    # 保存数据
    save_data()
  
    # 关闭文件
    if 'log_file' in globals():
        log_file.close()
  
    # 清理临时文件
    import os
    temp_files = ['temp.txt', 'cache.tmp']
    for f in temp_files:
        if os.path.exists(f):
            os.remove(f)
```

---

###  避免的做法

1. **不要阻塞主线程**

```python
#  错误：长时间等待
def on_command_received(ctx, msg, cmd_args):
    import time
    time.sleep(60)  # 会阻塞骰子
    return "等待完成"

#  正确：快速返回
def on_command_received(ctx, msg, cmd_args):
    return "命令已接收，正在处理..."
```

2. **不要修改核心文件**

```python
#  错误：尝试修改核心
import sys
sys.path.insert(0, '/path/to/sealdice')
from dice import Dice  # 不要这样做
```

3. **不要使用全局修改**

```python
#  错误：修改全局状态
import sys
sys.setrecursionlimit(100000)  # 可能影响核心
```

4. **不要忽略错误**

```python
#  错误：忽略异常
def on_command_received(ctx, msg, cmd_args):
    try:
        # 危险操作
        pass
    except:
        pass  # 不要这样做

#  正确：处理异常
def on_command_received(ctx, msg, cmd_args):
    try:
        # 危险操作
        pass
    except Exception as e:
        print(f"错误: {e}")
        return " 操作失败"
```

---

## 故障排查

### 常见问题

#### 1. 插件不加载

**现象**: 启动日志中没有显示插件

**排查**:

```bash
# 检查文件是否在正确位置
ls data/default/extensions/*.py

# 查看启动日志
grep "Python" sealdice.log
```

**解决**:

- 确保文件在 `data/default/extensions/` 目录
- 文件名不要以 `__` 开头
- 检查文件权限（可读）

---

#### 2. 命令不响应

**现象**: 发送命令后显示"未知指令"

**排查**:

```python
# 检查 __commands__ 是否正确定义
__commands__ = ["hello", "test"]  # 必须是列表

# 检查命令名称是否匹配
def on_command_received(ctx, msg, cmd_args):
    command = cmd_args.get('command', '')
    print(f"收到命令: {command}")  # 调试输出
```

**解决**:

- 确保 `__commands__` 定义为列表
- 命令名称要完全匹配（区分大小写）
- 重启核心使插件重新加载

---

#### 3. 返回消息不显示

**现象**: 命令执行了但没有回复

**排查**:

```python
def on_command_received(ctx, msg, cmd_args):
    #  错误：没有返回值
    if command == 'hello':
        print("Hello")  # 只打印，不返回
  
    #  正确：返回字符串
    if command == 'hello':
        return "Hello!"  # 返回消息
```

**解决**:

- 确保函数返回字符串
- 检查返回值不是 `None`
- 查看 `python_plugin.log` 是否有错误

---

#### 4. 数据不保存

**现象**: 重启后数据丢失

**排查**:

```python
# 检查保存路径
print(f"数据文件: {DATA_FILE}")

# 检查是否调用了保存
def on_unload():
    print("正在保存...")
    save_data()
    print("保存完成")
```

**解决**:

- 确保在 `on_unload()` 中调用 `save_data()`
- 检查目录权限
- 使用绝对路径或相对路径（相对于核心目录）

---

#### 5. Python 版本问题

**现象**: 插件无法运行，显示语法错误

**排查**:

```bash
# 检查 Python 版本
python3 --version

# 测试插件语法
python3 -m py_compile data/default/extensions/my_plugin.py
```

**解决**:

- 确保 Python >= 3.11
- 使用兼容的语法（避免 3.12+ 特有的功能，构建时使用的是Python3.11）
- 检查编码声明 `# -*- coding: utf-8 -*-`

---

### 调试技巧

#### 1. 日志输出

```python
# 方法1：标准输出
def on_command_received(ctx, msg, cmd_args):
    print(f"DEBUG: 命令 = {cmd_args}")
    print(f"DEBUG: 上下文 = {ctx}")
    print(f"DEBUG: 消息 = {msg}")

# 方法2：文件日志
LOG_FILE = "data/extensions/debug.log"

def log_debug(msg):
    with open(LOG_FILE, 'a', encoding='utf-8') as f:
        f.write(f"{datetime.now()}: {msg}\n")

def on_command_received(ctx, msg, cmd_args):
    log_debug(f"收到命令: {cmd_args}")
```

---

#### 2. 异常捕获

```python
def on_command_received(ctx, msg, cmd_args):
    try:
        # 你的代码
        result = process_command(cmd_args)
        return result
    except Exception as e:
        # 记录详细错误
        import traceback
        error_msg = traceback.format_exc()
  
        with open('data/extensions/error.log', 'a') as f:
            f.write(f"\n{'='*50}\n")
            f.write(f"时间: {datetime.now()}\n")
            f.write(f"错误: {e}\n")
            f.write(f"详情:\n{error_msg}\n")
  
        return f" 发生错误: {str(e)}"
```

---

#### 3. 参数检查

```python
def on_command_received(ctx, msg, cmd_args):
    # 打印所有参数
    debug_info = f"""
     调试信息:
    命令: {cmd_args.get('command')}
    参数: {cmd_args.get('args')}
    用户ID: {ctx.get('user_id')}
    群组ID: {ctx.get('group_id')}
    平台: {msg.get('platform')}
    消息: {msg.get('message')}
    """
  
    print(debug_info)
    return debug_info
```

---

## 附录

### A. 完整的参数结构

#### MsgContext (ctx)

```python
{
    "user_id": str,         # 用户ID，格式: "平台:ID"
    "group_id": str,        # 群组ID，格式: "平台-Group:ID"
    "is_private": bool,     # 是否私聊
    "dice_id": str,         # 骰子ID
}
```

#### Message (msg)

```python
{
    "message": str,         # 完整消息文本
    "sender": {             # 发送者信息
        "user_id": str,     # 用户ID
        "nickname": str,    # 昵称
    },
    "time": int,            # 时间戳
    "group_id": str,        # 群组ID
    "guild_id": str,        # 频道服务器ID（Discord/Kook）
    "channel_id": str,      # 频道ID
    "platform": str,        # 平台名称（QQ/Discord/Kook等）
    "group_name": str,      # 群组名称
}
```

#### CmdArgs (cmd_args)

```python
{
    "command": str,         # 命令名称（不含前缀）
    "args": list,           # 参数列表
    "raw_args": str,        # 原始参数字符串
    "clean_args": str,      # 清理后的参数
    "raw_text": str,        # 完整原始文本
}
```

---

### B. 支持的平台

| 平台     | ID 前缀       | 说明        |
| -------- | ------------- | ----------- |
| QQ       | `QQ:`       | 腾讯QQ      |
| Discord  | `DISCORD:`  | Discord     |
| Kook     | `KOOK:`     | 开黑啦      |
| Dodo     | `DODO:`     | DoDo        |
| Telegram | `TG:`       | Telegram    |
| DingTalk | `DINGTALK:` | 钉钉        |
| Slack    | `SLACK:`    | Slack       |
| SealChat | `SEALCHAT:` | 海豹聊天    |
| UI       | `UI:`       | Web UI 测试 |

---

### C. 有用的资源

- **海豹官方文档**: https://sealdice.github.io/sealdice-manual-next/
- **Python 官方文档**: https://docs.python.org/3/
- **JSON 教程**: https://www.json.org/
- **正则表达式**: https://regex101.com/

---

### D. 版本历史

| 版本  | 日期       | 更新内容                         |
| ----- | ---------- | -------------------------------- |
| 1.0.0 | 2026-01-13 | 初始版本，完整的 Python 插件系统 |

# Python扩展开发指南

## 概述

SealDice现在支持使用Python语言开发扩展.

## 特性

- **多语言支持**: 除了原有的JavaScript，现在还支持Python
- **丰富的Hook**: 提供更多的事件钩子供开发者使用
- **API访问**: 可以通过API调用访问SealDice的各种功能
- **配置管理**: 支持扩展配置的保存和加载

## 快速开始

### 1. 创建Python扩展文件

创建一个 `.py`文件，放在 `extensions/python/`目录下。

```python
# my_extension.py
name = "my_extension"
version = "1.0.0"
author = "Your Name"
brief = "我的第一个Python扩展"

def on_load():
    print("扩展已加载")

def on_message_received(ctx, msg):
    print(f"收到消息: {msg.content}")
```

### 2. 上传和加载扩展

通过Web UI或API上传Python文件，然后加载扩展。

### 3. API调用

```python
import requests

def some_function():
    # 调用SealDice API
    response = requests.get("http://localhost:3211/sd-api/some_endpoint")
    # 处理响应
```

## Hook函数

### 消息相关

- `on_message_received(ctx, msg)`: 收到消息时调用
- `on_command_received(ctx, msg, cmd_args)`: 收到命令时调用
- `on_not_command_received(ctx, msg)`: 收到非命令消息时调用

### 用户事件

- `on_user_joined(ctx, msg)`: 用户加入群组时调用
- `on_user_left(ctx, msg)`: 用户离开群组时调用

### 命令和骰点

- `on_command_executed(ctx, cmd_args, result)`: 命令执行后调用
- `on_dice_roll(ctx, expr, result)`: 骰点结果产生时调用

### 生命周期

- `on_load()`: 扩展加载时调用
- `on_unload()`: 扩展卸载时调用

## API端点

### Python扩展管理

- `GET /sd-api/python/list`: 列出所有Python扩展
- `POST /sd-api/python/upload`: 上传Python扩展文件
- `POST /sd-api/python/load`: 加载Python扩展
- `POST /sd-api/python/unload`: 卸载Python扩展
- `POST /sd-api/python/execute`: 执行Python代码
- `GET /sd-api/python/config`: 获取扩展配置
- `POST /sd-api/python/config`: 设置扩展配置
- `POST /sd-api/python/call_api`: 调用扩展API

## 开发注意事项

1. **错误处理**: 在hook函数中添加适当的错误处理
2. **性能**: 避免在hook中执行耗时操作
3. **线程安全**: 注意Python的GIL和Go的并发性
4. **配置**: 使用提供的配置API保存扩展设置

## 示例扩展

见 `extensions/python/example_ext.py`文件。

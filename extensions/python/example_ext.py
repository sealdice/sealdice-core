# 示例Python扩展
# 这是一个演示如何创建Python扩展的示例

name = "example_python_ext"
version = "1.0.0"
author = "Developer"
brief = "一个示例Python扩展，演示各种hook的使用"

def on_load():
    """扩展加载时的回调"""
    print("Python扩展已加载")

def on_unload():
    """扩展卸载时的回调"""
    print("Python扩展已卸载")

def on_message_received(ctx, msg):
    """收到消息时的回调"""
    print(f"收到消息: {msg.content}")
    # 这里可以添加消息处理逻辑

def on_command_received(ctx, msg, cmd_args):
    """收到命令时的回调"""
    print(f"收到命令: {cmd_args.command}")
    # 这里可以添加命令处理逻辑

def on_not_command_received(ctx, msg):
    """收到非命令消息时的回调"""
    print(f"收到非命令消息: {msg.content}")
    # 这里可以添加非命令消息处理逻辑

def on_user_joined(ctx, msg):
    """用户加入群组时的回调"""
    print(f"用户 {msg.sender.user_id} 加入了群组")
    # 这里可以添加欢迎逻辑

def on_user_left(ctx, msg):
    """用户离开群组时的回调"""
    print(f"用户 {msg.sender.user_id} 离开了群组")
    # 这里可以添加告别逻辑

def on_command_executed(ctx, cmd_args, result):
    """命令执行后的回调"""
    print(f"命令 {cmd_args.command} 执行完成，结果: {result}")
    # 这里可以添加执行后处理逻辑

def on_dice_roll(ctx, expr, result):
    """骰点结果回调"""
    print(f"骰点表达式: {expr}, 结果: {result}")
    # 这里可以添加骰点结果处理逻辑

# 导出函数给Go代码调用
# 注意：这些函数名必须与hook名称匹配
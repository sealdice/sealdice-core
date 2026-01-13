"""
测试Python扩展 - 演示基本的hook功能
"""

import json

# 扩展信息
__name__ = "test_python_plugin"
__version__ = "1.0.0"
__author__ = "Test"
__description__ = "一个简单的测试Python插件"

def on_load():
    """插件加载时调用"""
    print("🎉 Python测试插件已加载！")
    print(f"插件名称: {__name__}")
    print(f"版本: {__version__}")

def on_unload():
    """插件卸载时调用"""
    print("👋 Python测试插件已卸载")

def on_message_received(ctx, msg):
    """收到消息时调用"""
    try:
        message_content = msg.get("message", "")
        sender = msg.get("sender", {})
        user_id = sender.get("user_id", "unknown") if isinstance(sender, dict) else str(sender)

        print(f"📨 收到消息: {message_content}")
        print(f"👤 发送者: {user_id}")
        print(f"📅 时间: {msg.get('time', 'unknown')}")

        if "hello" in message_content.lower():
            print("🤖 检测到hello消息，准备回复")

    except Exception as e:
        print(f"❌ 处理消息时出错: {e}")

def on_command_received(ctx, msg, cmd_args):
    """收到命令时调用"""
    try:
        command = cmd_args.get("command", "")
        args = cmd_args.get("args", [])

        print(f"⚡ 收到命令: {command}")
        print(f"🔧 参数: {args}")

        # 如果是测试命令
        if command == "test":
            print("🧪 执行测试命令")

    except Exception as e:
        print(f"❌ 处理命令时出错: {e}")

def on_command_executed(ctx, cmd_args, result):
    """命令执行后调用"""
    try:
        command = cmd_args.get("command", "")
        print(f"✅ 命令执行完成: {command}")
        print(f"📊 执行结果: {result}")
    except Exception as e:
        print(f"❌ 处理命令结果时出错: {e}")

def on_dice_roll(ctx, expr, result):
    """骰点结果时调用"""
    try:
        print(f"🎲 骰点表达式: {expr}")
        print(f"🎯 骰点结果: {result}")
    except Exception as e:
        print(f"❌ 处理骰点结果时出错: {e}")
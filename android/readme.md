# SealDice Android

### [SealDice海豹核心](https://github.com/sealdice/sealdice-core) 用于手机运行的版本

## 搭建须知
该项目使用了ACRA作为崩溃日志收集器
因此在运行本项目前你应当创建 com.sealdice.dice.secrets 包并在其中添加一个名为 Auth 的 java class
其内容如下

```java
package com.sealdice.dice.secrets;

public class Auth {
    public static String ACRA_URL = "YOUR REPORT URL";
    public static String ACRA_BASIC_AUTH = "YOUR AUTH";
    public static String ACRA_LOGIN_PASS = "YOUR PASS";
}

```

或者你可以选择在MyApplication.kt中删除ACRA的初始化代码并移除import语句

然后，你需要将Android NDK内提供的C编译器路径填入CGO的CC参数并将goos参数写为“android” 
随后你需要将编译好的 [SealDice海豹核心](https://github.com/sealdice/sealdice-core) 
放入assets/sealdice 目录内

## 关于 issue 和 pull request
你可以通过 fork 本项目并提交 pull request 的形式贡献代码
关于手机版的功能需求和 bug 反馈请在本仓库内提交
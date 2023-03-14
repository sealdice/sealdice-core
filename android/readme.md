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

## 关于 issue 和 pull request
你可以通过 fork 本项目并提交 pull request 的形式贡献代码
关于手机版的功能需求和 bug 反馈请在本仓库内提交
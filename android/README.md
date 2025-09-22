# SealDice Android

![Software MIT License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)
![Android](https://img.shields.io/badge/SealDice-Android-blue)

[SealDice 海豹核心](https://github.com/sealdice/sealdice-core) 用于手机运行的版本。

## 搭建须知

该项目使用了 ACRA 作为崩溃日志收集器。

因此在运行本项目前你应当创建 `com.sealdice.dice.secrets` 包并在其中添加一个名为 `Auth` 的 Java Class。其内容如下：

```java
package com.sealdice.dice.secrets;

public class Auth {
    public static String ACRA_URL = "YOUR REPORT URL";
    public static String ACRA_BASIC_AUTH = "YOUR AUTH";
    public static String ACRA_LOGIN_PASS = "YOUR PASS";
}
```

或者你可以选择在 `MyApplication.kt` 中删除 ACRA 的初始化代码并移除 `import` 语句。

然后，你需要将 Android NDK 内提供的 C 编译器路径填入 CGO 的 `CC` 参数并将 `goos` 参数写为 `android`。

随后你需要将编译好的海豹核心后端 `sealdice-core` 放入 `assets/sealdice` 目录内。

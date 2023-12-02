---
lang: zh-cn
title: 编写新的TRPG规则
---

# 编写新的TRPG规则

::: info 本节内容

本节将介绍如何添加一个新的TRPG游戏规则到海豹，主要涉及的部分是添加规则模板和指令编写。

:::

## 规则模板是什么？有什么功能？

这里假设我们创建了一个叫做摸鱼大赛的trpg规则，简称为fish规则。

规则模板从早期的同义词模板发展而来，他能为我们的fish规则提供以下帮助：

1. 可以直接使用set fish ，然后对应的fish扩展会自动打开，默认骰子也变更为六面骰

2. st show的内容会改变，coc相关的几个主属性不会再强制排在最前

3. 角色属性可以摆脱coc同义词的影响，同样coc的默认值也不会再影响fish规则了

4. 可以自定义fish规则自己的同义词，包括简繁写法、缩写等内容

5. 可以影响sn指令，添加一条sn fish来标注对当前规则重要的属性在玩家名片上

6. fish规则人物卡会成为与内置的coc、dnd平级的专门卡片，有独立的技能默认值，以及二级属性计算等机制


## 那么，要怎么做？

我们已经写了一个比较完善的示例，可以参考这里，有大量的详细注释：

https://github.com/sealdice/javascript/blob/main/examples_ts/013.%E8%87%AA%E5%AE%9A%E4%B9%89TRPG%E6%B8%B8%E6%88%8F%E8%A7%84%E5%88%99.ts

上面是TypeScript文件，可直接执行的js版本在这里：

https://github.com/sealdice/javascript/blob/main/examples/013.%E8%87%AA%E5%AE%9A%E4%B9%89TRPG%E6%B8%B8%E6%88%8F%E8%A7%84%E5%88%99.js

大部分配置由代码中的数据文件来进行。

关于指令编写的部分，可以参考js文档。

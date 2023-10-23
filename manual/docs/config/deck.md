---
lang: zh-cn
title: 牌堆
---

# 牌堆

::: info 本节内容

本节将介绍牌堆，请善用侧边栏和搜索，按需阅读文档。

:::

## 牌堆

仓库：https://github.com/sealdice/draw

### 牌堆是什么 怎么加在骰子里

牌堆的本质是json文件或者yaml文件，编写遵循两者的语法。

- 一般的牌堆文件放置在.\data\decks 目录下以单独的文件存在，如果要添加
  牌堆，你只需要把下载好的文件放在该目录下，然后对骰子发送 .draw
  reload 指令就可以了。

- 对于带图的牌堆压缩包，如果是按照seal格式来的，解压后整个文件夹拖进
  decks目录里。
  其他情况下的带图的牌堆文件压缩包一般会给一个说明
  这里只给出几个大概的名词间等价关系 请自行摸索

```
dicedata ≈ dice123456789 ≈ dice骰子QQ ≈ cocdata ≈ seal根目录
PublicDeck ≈ draw ≈ decks
mod ≈ helpdoc
pictures ≈ images
Q：.\data\decks 是什么？
A：在我的电脑上就是 D:\Uso\tmp\sealdice-core\data\decks 因为
D:\Uso\tmp\sealdice-core 和每个人解压seal程序的位置有关，是可以自定义的，
所以用 . 来表示
```

### 牌堆怎么用

见 [使用-扩展：牌堆 自定义回复](../use/deck_and_reply.md)

### 牌堆怎么编写

见 [进阶-编写牌堆](../advanced/edit_deck.md)
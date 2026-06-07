<template>
  <TipBox type="info">
    <n-collapse class="wordtips" accordion>
      <n-collapse-item name="txt-template">
        <template #header>
          <n-text tag="strong">敏感词库</n-text>
        </template>
        <n-text tag="p">
          当前支持两种格式的词库：toml（推荐）和 txt。<br />
          toml 格式支持填写包括作者、版本等信息，便于分享，txt 格式则是溯洄系敏感词库的兼容。<br />
          敏感词分为
          <n-flex inline size="small">
            <n-tag size="small" type="default" :bordered="false">忽略</n-tag>
            <n-tag size="small" type="default" :bordered="false">提醒</n-tag>
            <n-tag size="small" type="info" :bordered="false">注意</n-tag>
            <n-tag size="small" type="warning" :bordered="false">警告</n-tag>
            <n-tag size="small" type="error" :bordered="false">危险</n-tag>
            5 个级别，严重程度依次上升。其中
            <n-tag size="small" type="default" :bordered="false">忽略</n-tag>
          </n-flex>
          级别没有实际作用，也不进行展示。不同词库中和同一词库中不同等级的重复词汇，将按照给定的
          <strong>最高级别</strong> 判断。<br />
          更多介绍见下：
        </n-text>

        <n-collapse accordion>
          <n-collapse-item>
            <template #header>
              <n-text>toml 格式的词库</n-text>
            </template>
            <n-text tag="p">
              使用 toml
              格式作为词库，支持填写便于分享的信息，如作者、版本、更新日期等。可以下载模板查看具体的属性（内含注释）。<br />
              toml 格式支持注释、尾逗号等语法，可查看
              <n-button
                type="default"
                size="tiny"
                text
                style="text-decoration: none"
                tag="a"
                target="_blank"
                href="https://toml.io/cn/v1.0.0"
              >
                <template #icon>
                  <n-icon><i-carbon-search /></n-icon>
                </template>
                了解 toml 格式
              </n-button>
              或自行搜索。
            </n-text>
          </n-collapse-item>
          <n-collapse-item>
            <template #header>
              <n-text>txt 格式的词库</n-text>
            </template>
            <n-text tag="p">
              格式示例如下：<br />
              <br />
              <n-text code>
                #notice<br />
                提醒级词汇 1<br />
                提醒级词汇 2<br />
                #caution<br />
                注意级词汇 1<br />
                注意级词汇 2<br />
                #warning<br />
                警告级词汇<br />
                #danger<br />
                危险级词汇<br />
              </n-text>
              <br />
              <br />
              <n-tag size="small" type="default" :bordered="false">#等级</n-tag>
              以下的词汇为该等级的敏感词，每一行代表一个词汇，直到遇到新的等级标记。
            </n-text>
          </n-collapse-item>
        </n-collapse>
      </n-collapse-item>
    </n-collapse>
  </TipBox>
</template>

<script setup lang="ts">
import TipBox from '@/components/shared/TipBox.vue';
</script>

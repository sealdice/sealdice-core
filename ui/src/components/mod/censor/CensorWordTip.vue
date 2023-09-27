<template>
  <div class="tip">
    <el-collapse class="wordtips">
      <el-collapse-item name="txt-template">
        <template #title>
          <el-text tag="strong">敏感词库</el-text>
        </template>
        <el-text tag="p" style="margin-bottom: 1rem">
          当前支持两种格式的词库：toml（推荐）和 txt。<br/>
          toml 格式支持填写包括作者、版本等信息，便于分享，txt 格式则是溯洄系敏感词库的兼容。<br/>
          敏感词分为
          <el-tag size="small" type="info" disable-transitions>忽略</el-tag>
          <el-tag size="small" type="info" disable-transitions>提醒</el-tag>
          <el-tag size="small" disable-transitions>注意</el-tag>
          <el-tag size="small" type="warning" disable-transitions>警告</el-tag>
          <el-tag size="small" type="danger" disable-transitions>危险</el-tag>
          5个级别，严重程度依次上升。其中
          <el-tag type="info" disable-transitions>忽略</el-tag>
          级别没有实际作用，也不进行展示。
          不同词库中和同一词库中不同等级的重复词汇，将按照给定的 <strong>最高级别</strong> 判断。<br/>
          更多介绍见下：
        </el-text>
        <el-collapse accordion>
          <el-collapse-item>
            <template #title>
              <el-text>toml 格式的词库</el-text>
            </template>
            <el-text tag="p">
              使用 toml 格式作为词库，支持填写便于分享的信息，如作者、版本、更新日期等。可以下载模板查看具体的属性（内含注释）。<br/>
              toml 格式支持注释、尾逗号等语法，可查看
              <el-button style="text-decoration: none" type="info" :icon="Search" size="small" link tag="a"
                         target="_blank"
                         href="https://toml.io/cn/v1.0.0">
                了解 toml 格式
              </el-button>
              或自行搜索。
            </el-text>
          </el-collapse-item>
          <el-collapse-item>
            <template #title>
              <el-text>txt 格式的词库</el-text>
            </template>
            <el-text tag="p">
              <em>格式示例如下：</em><br/>
              <br/>
              #notice<br/>
              提醒级词汇1<br/>
              提醒级词汇2<br/>
              #caution<br/>
              注意级词汇1<br/>
              注意级词汇2<br/>
              #warning<br/>
              警告级词汇<br/>
              #danger<br/>
              危险级词汇<br/>
              <br/>
              <em><el-tag size="small" type="info" disable-transitions> #等级</el-tag>以下的词汇为该等级的敏感词，每一行代表一个词汇，直到遇到新的等级标记。</em>
            </el-text>
          </el-collapse-item>
        </el-collapse>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<style scoped>
.wordtips {
  background-color: #f3f5f7;
}

.wordtips :deep().el-collapse-item__header {
  background-color: #f3f5f7;
}

.wordtips :deep().el-collapse-item__wrap {
  background-color: #f3f5f7;
}
</style>
<script setup lang="ts">
import {urlBase} from "~/backend";
import {Download, Search} from "@element-plus/icons-vue";
</script>
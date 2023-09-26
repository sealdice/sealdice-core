<template>
  <header class="page-header">
    <el-switch v-model="censorEnable" @change="enableChange" active-text="启用"
               inactive-text="关闭"/>
    <el-button type="primary" :icon="Refresh"
               v-show="censorEnable" @click="restartCensor">重载拦截
    </el-button>
  </header>

  <el-affix :offset="60" v-if="censorStore.needReload">
    <div class="tip-danger">
      <el-text type="danger" size="large" tag="strong">存在修改，需要重载后生效！</el-text>
    </div>
  </el-affix>

  <template v-if="censorEnable">
    <el-tabs v-model="tab" stretch>
      <el-tab-pane label="拦截设置" name="setting">
        <censor-config></censor-config>
      </el-tab-pane>

      <el-tab-pane label="敏感词管理" name="word">
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
                5个级别，严重程度依次上升。其中<el-tag type="info" disable-transitions>忽略</el-tag>级别没有实际作用，也不进行展示。
                不同词库中和同一词库中不同等级的重复词汇，将按照给定的 <strong>最高级别</strong> 判断。<br/>
                词库格式介绍和相关模板下载见下：
              </el-text>
              <el-collapse accordion>
                <el-collapse-item>
                  <template #title>
                    <el-text>toml 格式的词库</el-text>
                  </template>
                  <div style="margin-bottom: 1rem">
                    <el-button style="text-decoration: none" type="success" tag="a" target="_blank" link size="small"
                               :href="`${urlBase}/sd-api/censor/files/template/toml`" :icon="Download">
                      下载 toml 词库模板
                    </el-button>
                    <el-button style="text-decoration: none" type="info" :icon="Search" size="small" link tag="a"
                               target="_blank"
                               href="https://toml.io/cn/v1.0.0">
                      了解 toml 格式
                    </el-button>
                  </div>
                  <el-text tag="p">
                    使用 toml 格式作为词库，支持填写便于分享的信息，如作者、版本、更新日期等。可以下载模板查看具体的属性（内含注释）。<br/>
                    toml 格式支持注释、尾逗号等语法，可查看上方链接或自行搜索。
                  </el-text>
                </el-collapse-item>
                <el-collapse-item>
                  <template #title>
                    <el-text>txt 格式的词库</el-text>
                  </template>
                  <div style="margin-bottom: 1rem">
                    <el-button style="text-decoration: none" type="success" tag="a" target="_blank" link size="small"
                               :href="`${urlBase}/sd-api/censor/files/template/txt`" :icon="Download">下载 txt 词库模板
                    </el-button>
                  </div>
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
                    <em>#等级 以下的词汇为该等级的敏感词，每一行代表一个词汇，直到遇到新的等级标记。</em>
                  </el-text>
                </el-collapse-item>
              </el-collapse>
            </el-collapse-item>
          </el-collapse>
        </div>

        <h4>词库列表</h4>
        <censor-files></censor-files>
        <h4>敏感词列表</h4>
        <censor-words></censor-words>
      </el-tab-pane>

      <el-tab-pane label="拦截日志" name="log">
        <censor-log></censor-log>
      </el-tab-pane>
    </el-tabs>
  </template>
  <template v-else>
    <el-text type="danger" size="large" style="font-size: 1.5rem; display: block; margin-top: 1rem;">请先启用拦截！
    </el-text>
  </template>
</template>

<script lang='ts' setup>
import {Download, Refresh, Search} from '@element-plus/icons-vue';
import {onBeforeMount, ref} from 'vue';
import {urlPrefix, useStore} from '~/store';
import {backend, urlBase} from '~/backend'

onBeforeMount(() => {
  refreshCensorStatus()
})

import CensorFiles from "~/components/mod/censor/CensorFiles.vue";
import {useCensorStore} from "~/components/mod/censor/censor";

const store = useStore()
const token = store.token

const url = (p: string) => urlPrefix + "/censor/" + p;
const censorEnable = ref<boolean>(false)

const censorStore = useCensorStore()

const refreshCensorStatus = async () => {
  const status: { result: false } | {
    result: true,
    enable: boolean,
    isLoading: boolean
  } = await backend.get(url("status"), {});
  if (status.result) {
    censorEnable.value = status.enable
  }
}

const restartCensor = async () => {
  const restart = await censorStore.restartCensor()
  if (restart.result) {
    censorEnable.value = restart.enable
    censorStore.reload()
  }
}

const stopCensor = async () => {
  const stop = await censorStore.stopCensor()
  if (stop.result) {
    censorEnable.value = false
  }
}

const enableChange = async (value: boolean | number | string) => {
  if (value === true) {
    await restartCensor()
  } else {
    await stopCensor()
  }
}

const tab = ref("setting")

</script>


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
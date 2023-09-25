<template>
  <header>
    <el-button type="primary" :icon="DocumentChecked" @click="submit">保存设置</el-button>
    <el-text style="margin-left: 1rem" v-if="modified" type="danger" size="large" tag="strong">
      内容已修改，不要忘记保存！
    </el-text>
  </header>
  <el-form label-width="130px">
    <h4>匹配选项</h4>
    <el-form-item>
      <template #label>
        <el-text>拦截模式</el-text>
        <el-tooltip raw-content
                    content="">
          <template #content>
            全部：会对所有收到的消息进行检查，命中时拒绝回复，<el-tag size="small" type="info" disable-transitions>拦截_拦截提示_全部模式</el-tag>不为空时会发送提醒。<br />
            仅命令：只会对收到的以 <strong>指令前缀</strong> 开头的消息进行检查，命中时拒绝回复，<el-tag size="small" type="info" disable-transitions>拦截_拦截提示_仅命令模式</el-tag>不为空时会发送提醒。<br />
            仅回复：只检查回复内容，命中时回复将替换为<el-tag size="small" type="info" disable-transitions>拦截_拦截提示_仅回复模式</el-tag>的内容。
          </template>
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-radio-group v-model="config.mode">
        <el-radio-button :label="Mode.All">{{ "全部" }}</el-radio-button>
        <el-radio-button :label="Mode.OnlyCommand">{{ "仅命令" }}</el-radio-button>
        <el-radio-button :label="Mode.OnlyReply">{{ "仅回复" }}</el-radio-button>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="大小写敏感">
      <el-checkbox label="开启" v-model="config.caseSensitive"/>
    </el-form-item>
    <el-form-item>
      <template #label>
        <el-text>匹配拼音</el-text>
        <el-tooltip
            content="匹配敏感词拼音，勾选大小写敏感时该项无效。">
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-checkbox label="开启" v-model="config.matchPinyin"/>
    </el-form-item>
    <el-form-item>
      <template #label>
        <el-text>过滤字符正则</el-text>
        <el-tooltip
            content='判断敏感词时，忽略过滤字符。如敏感词为"114514"，指定过滤字符为空白，则"114   514"也会命中敏感词。'>
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-input v-model="config.filterRegex" style="width: 12rem;"/>
    </el-form-item>
  </el-form>
  <div>
    <h4 style="display: inline;">响应设置</h4>
    <el-text style="display: block; margin: 1rem 0;" type="warning" size="small">注：超过阈值时，对应用户该等级的计数会被清空重新计算。</el-text>
  </div>
  <el-form >
    <el-form-item>
      <template #label>
        <el-tag type="info" style="align-self: center">提醒</el-tag>
      </template>
      <el-space wrap>
        <el-text>用户触发超过</el-text>
        <el-input-number v-model="config.levelConfig.notice.threshold" style="margin: 0 0.5rem;" size="small" :step="1" :min="0"
                         step-strictly/>
        <el-text>次时：</el-text>
      </el-space>
      <el-space direction="vertical" alignment="normal">
        <div>
          <el-checkbox-group v-model="config.levelConfig.notice.handlers">
            <el-checkbox v-for="handle in defaultHandles" :label="handle.key">
              {{ handle.name }}
            </el-checkbox>
          </el-checkbox-group>
          <el-text>怒气值</el-text>
          <el-input-number v-model="config.levelConfig.notice.score" style="margin-left: 1rem;" size="small" :step="1" :min="0"
                           step-strictly/>
        </div>
      </el-space>
    </el-form-item>
    <el-form-item>
      <template #label>
        <el-tag size="small" style="align-self: center">注意</el-tag>
      </template>
      <el-space wrap>
        <el-text>用户触发超过</el-text>
        <el-input-number v-model="config.levelConfig.caution.threshold" style="margin: 0 0.5rem;" size="small" :step="1" :min="0"
                         step-strictly/>
        <el-text>次时：</el-text>
      </el-space>
      <el-space direction="vertical" alignment="normal">
        <div>
          <el-checkbox-group v-model="config.levelConfig.caution.handlers">
            <el-checkbox v-for="handle in defaultHandles" :label="handle.key">
              {{ handle.name }}
            </el-checkbox>
          </el-checkbox-group>
          <el-text>怒气值</el-text>
          <el-input-number v-model="config.levelConfig.caution.score" style="margin-left: 1rem;" size="small" :step="1" :min="0"
                           step-strictly/>
        </div>
      </el-space>
    </el-form-item>
    <el-form-item>
      <template #label>
        <el-tag type="warning" size="small" style="align-self: center">警告</el-tag>
      </template>
      <el-space wrap>
        <el-text>用户触发超过</el-text>
        <el-input-number v-model="config.levelConfig.warning.threshold" style="margin: 0 0.5rem;" size="small" :step="1" :min="0"
                         step-strictly/>
        <el-text>次时：</el-text>
      </el-space>
      <el-space direction="vertical" alignment="normal">
        <div>
          <el-checkbox-group v-model="config.levelConfig.warning.handlers">
            <el-checkbox v-for="handle in defaultHandles" :label="handle.key">
              {{ handle.name }}
            </el-checkbox>
          </el-checkbox-group>
          <el-text>怒气值</el-text>
          <el-input-number v-model="config.levelConfig.warning.score" style="margin-left: 1rem;" size="small" :step="1" :min="0"
                           step-strictly/>
        </div>
      </el-space>
    </el-form-item>
    <el-form-item>
      <template #label>
        <el-tag type="danger" size="small" style="align-self: center">危险</el-tag>
      </template>
      <el-space wrap>
        <el-text>用户触发超过</el-text>
        <el-input-number v-model="config.levelConfig.danger.threshold" style="margin: 0 0.5rem;" size="small" :step="1" :min="0"
                         step-strictly/>
        <el-text>次时：</el-text>
      </el-space>
      <el-space direction="vertical" alignment="normal">
        <div>
          <el-checkbox-group v-model="config.levelConfig.danger.handlers">
            <el-checkbox v-for="handle in defaultHandles" :label="handle.key">
              {{ handle.name }}
            </el-checkbox>
          </el-checkbox-group>
          <el-text>怒气值</el-text>
          <el-input-number v-model="config.levelConfig.danger.score" style="margin-left: 1rem;" size="small" :step="1" :min="0"
                           step-strictly/>
        </div>
      </el-space>
    </el-form-item>

  </el-form>
</template>

<script lang="ts" setup>

import {nextTick, onBeforeMount, onBeforeUnmount, ref, watch} from "vue";
import {backend} from "~/backend";
import {urlPrefix, useStore} from "~/store";
import {DocumentChecked, QuestionFilled} from "@element-plus/icons-vue";
import {isArray, isEqual, isObject, transform} from "lodash-es";
import {ElMessage} from "element-plus";
import {useCensorStore} from "~/components/mod/censor/censor";

onBeforeMount(async () => {
  await refreshCensorConfig()
  nextTick(() => {
    modified.value = false
  })
})

onBeforeUnmount(() => {
  clearInterval(timerId)
})

const store = useStore()
const url = (p: string) => urlPrefix + "/censor/" + p;
const token = store.token
const censorStore = useCensorStore()

const enum Mode {
  All = 0,
  OnlyCommand,
  OnlyReply,
}

interface Config {
  mode: Mode
  caseSensitive: boolean
  matchPinyin: boolean
  filterRegex: string
  levelConfig: LevelConfigs
}

const config = ref<Config>({
  mode: Mode.All,
  caseSensitive: false,
  matchPinyin: false,
  filterRegex: "",
  levelConfig: {
    notice: {threshold: 0, handlers: [], score: 0},
    caution: {threshold: 0, handlers: [], score: 0},
    warning: {threshold: 0, handlers: [], score: 0},
    danger: {threshold: 0, handlers: [], score: 0},
  },
})

interface LevelConfigs {
  notice: LevelConfig
  caution: LevelConfig
  warning: LevelConfig
  danger: LevelConfig
}

interface LevelConfig {
  threshold: number
  handlers: string[]
  score: number
}

const defaultHandles: { key: string, name: string }[] = [
  {key: "SendWarning", name: "发送警告"},
  {key: "SendNotice", name: "通知 Master"},
  {key: "BanUser", name: "拉黑用户"},
  {key: "BanGroup", name: "拉黑群"},
  {key: "BanInviter", name: "拉黑邀请人"},
  {key: "AddScore", name: "增加怒气值"},
]

const modified = ref<boolean>(false)

watch(config, () => {
  modified.value = true
}, {deep: true});

const getCensorConfig = async () => {
  const c = await censorStore.getConfig()
  if (c.result) {
    return c
  }
}

censorStore.$subscribe(async (_, state) => {
  if (state.settingsNeedRefresh === true) {
    await refreshCensorConfig()
    state.settingsNeedRefresh = false
  }
})

let timerId: number
const refreshCensorConfig = async () => {
  const c = await getCensorConfig()
  if (c) {
    config.value = c as any;
  }
  modified.value = false
  await nextTick(() => {
    modified.value = false
  })
}

const confDiff = (object: any, base: any) => {
  const changes = function (object: any, base: any) {
    return transform(object, (result: any, value, key) => {
      if (isArray(value)) {
        result[key] = value
      } else if (!isEqual(value, base[key])) {
        result[key] = (isObject(value) && isObject(base[key])) ? changes(value, base[key]) : value
      }
    })
  }
  return changes(object, base)
}

const submit = async () => {
  const conf = await getCensorConfig()
  const modify = confDiff(config.value, conf)

  const resp = await censorStore.saveConfig(modify);
  if (resp.result) {
    ElMessage.success("保存设置成功")
  } else {
    ElMessage.error("保存设置失败，" + resp.err)
  }

  await refreshCensorConfig()
  censorStore.markReload()
  modified.value = false
  await nextTick(() => {
    modified.value = false
  })
}
</script>
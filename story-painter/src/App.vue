<template>
  <div style="width: 1000px; margin: 0 auto; max-width: 100%;">
    <h2 style="text-align: center;">海豹TRPG跑团Log着色器测试版 V1.01</h2>
    <div style="text-align: center;">SealDice骰QQ群 524364253</div>
    <div style="text-align: center;">新骰系内测中，快来提需求！</div>
    <div class="options" style="display: flex; flex-wrap: wrap; text-align: center;">
      <div>
        <div class="switch">
          <el-switch v-model="exportOptions.commandHide" />
          <h4>骰子指令过滤</h4>
        </div>
        <div>开启后，不显示pc骰点指令，正常显示骰点结果</div>
      </div>

      <div>
        <div class="switch">
          <el-switch v-model="exportOptions.imageHide" />
          <h4>表情包和图片过滤</h4>
        </div>
        <div>开启后，文本内所有的表情包和图片将被豹豹藏起来不显示</div>
      </div>

      <div>
        <div class="switch">
          <el-switch v-model="exportOptions.offSiteHide" />
          <h4>场外发言过滤</h4>
        </div>
        <div>开启后，所有以(和【为开头的发言将被豹豹吃掉不显示</div>
      </div>

      <div>
        <div class="switch">
          <el-switch v-model="exportOptions.timeHide" />
          <h4>时间显示过滤</h4>
        </div>
        <div>开启后，日期和时间会被豹豹丢入海里不显示</div>
      </div>

      <div>
        <div class="switch">
          <el-switch v-model="exportOptions.userIdHide" />
          <h4>隐藏帐号</h4>
        </div>
        <div>开启后，QQ号将在导出结果中不显示</div>
      </div>

      <div>
        <div class="switch">
          <el-switch v-model="exportOptions.yearHide" />
          <h4>隐藏年月日</h4>
        </div>
        <div>开启后，导出结果的日期将只显示几点几分(如果可能)</div>
      </div>
    </div>

    <div class="pc-list">
      <div v-for="i, index in store.pcList">
        <div style="display: flex; align-items: center; width: 20rem;">
          <el-button style="padding: 0 1rem " @click="store.pcList.splice(index, 1)">删除</el-button>

          <el-input
            v-model="i.name"
            class="w-50 m-2"
            :prefix-icon="UserFilled"
            @focus="nameFocus(i)"
            @change="nameChanged(i)"
          />

          <el-select v-model="i.role" class="m-2">
            <el-option value="主持人" />
            <el-option value="角色" />
            <el-option value="骰子" />
            <el-option value="隐藏" />
          </el-select>

          <el-color-picker v-model="i.color" size="large" style="border: none;" />
        </div>
      </div>
    </div>

    <div
      style="margin-bottom: 2rem; display: flex; justify-content: center; align-items: center; flex-wrap: wrap;"
    >
      <el-button @click="exportRecordRaw">下载原始文件</el-button>
      <el-button @click="exportRecordQQ">下载QQ风格记录</el-button>
      <el-button @click="exportRecordIRC">下载IRC风格记录</el-button>
      <el-button @click="exportRecordDOCX">下载Word</el-button>
      <!-- <el-button @click="showPreview">预览</el-button> -->
      <el-checkbox v-model="isShowPreview" label="预览" :border="true" style="margin-left: 1rem;" />
    </div>

    <code-mirror v-show="!isShowPreview" ref="editor" @change="onChange" />
    <!-- <monaco-editor @change="onChange"/> -->
    <div class="preview" ref="preview" v-show="isShowPreview">
      <div v-for="i in previewItems">
        <span
          style="color: #aaa"
          class="_time"
          v-if="!store.exportOptions.timeHide"
        >{{ timeSolve(i) }}</span>
        <span :style="{ 'color': i.color }" class="_nickname">{{ nicknameSolve(i) }}</span>
        <span :style="{ 'color': i.color }" v-html="i.message.replace('\n', '<br />')"></span>
      </div>
    </div>
  </div>

  <div style="margin-bottom: 3rem"></div>
</template>

<script setup lang="ts">
import { nextTick, ref, onMounted, watch } from "vue";
// import MonacoEditor from './components/MonacoEditor.vue'
import { useStore } from './store'
import type { LogItem, CharItem } from './store'
import dayjs from 'dayjs'
import { UserFilled } from '@element-plus/icons-vue'
// import { resetTheme } from "./components/highlight";
import CodeMirror from './components/CodeMirror.vue'
import { reNameLine, reNameLine2 } from "./utils/highlight";
import { EditorState, StateEffect } from '@codemirror/state';
import { debounce } from 'lodash-es'
import { convertToLogItems, exportFileRaw, exportFileQQ, exportFileIRC, exportFileDocx } from "./utils/exporter";

const nicknameSolve = (i: LogItem) => {
  let userid = '(' + i.IMUserId + ')'
  const options = store.exportOptions
  if (options.userIdHide) {
    userid = ''
  }
  return `<${i.nickname}${userid}>:`
}

const timeSolve = (i: LogItem) => {
  let timeText = i.time.toString()
  const options = store.exportOptions
  if (typeof i.time === 'number') {
    timeText = dayjs.unix(i.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss')
  }
  if (options.timeHide) {
    timeText = ''
  }
  return timeText
}

onMounted(() => {
  // cminstance.value = cmRefDom.value?.cminstance;
  // cminstance.value?.focus();

  // console.log(cminstance.value)
  onChange()
});

function exportRecordRaw() {
  exportFileRaw(store.editor.state.doc.toString())
}

function exportRecordQQ() {
  const results = convertToLogItems(store.editor.state.doc.toString(), store.pcList, store.exportOptions)
  exportFileQQ(results, store.exportOptions)
}

function exportRecordIRC() {
  const results = convertToLogItems(store.editor.state.doc.toString(), store.pcList, store.exportOptions)
  exportFileIRC(results, store.exportOptions)
}

const preview = ref(null)

const isShowPreview = ref(false)

function exportRecordDOCX() {
  // 其实是伪doc
  previewItems.value = convertToLogItems(store.editor.state.doc.toString(), store.pcList, store.exportOptions, true)
  isShowPreview.value = true // 强制切换
  nextTick(() => {
    const el = preview.value
    if (el) {
      // 注意有个等待图片加载的时间，暂时没做
      setTimeout(() => {
        exportFileDocx(el as any)
      }, 500)
    }
  })
}

const previewItems = ref<LogItem[]>([])

function showPreview() {
  previewItems.value = convertToLogItems(store.editor.state.doc.toString(), store.pcList, store.exportOptions, true)
}

const store = useStore()
const color2 = ref('#409EFF')

async function loadLog(items: LogItem[]) {
  let text = ""
  let changed = false
  for (let i of items) {
    if (await store.tryAddPcList(i)) {
      changed = true
    }

    const timeText = dayjs.unix(i.time).format('YYYY/MM/DD HH:mm:ss')
    text += `${i.nickname}(${i.IMUserId}) ${timeText}\n${i.message}\n\n`
  }

  store.editor.dispatch({
    changes: { from: 0, to: store.editor.state.doc.length, insert: text }
  })
  return changed
}

let lastPCName = ''

const nameFocus = (i: CharItem) => {
  lastPCName = i.name
}

const nameChanged = (i: CharItem) => {
  if (lastPCName && i.name) {
    const editor = store.editor

    let text = editor.state.doc.toString(), pos = 0
    let changes = []
    for (let next; (next = text.indexOf(lastPCName, pos)) > -1;) {
      const inLeft = next === 0
      const inRight = next + lastPCName.length === text.length

      let leftSafe = false
      let rightSafe = false
      if (!inLeft) {
        leftSafe = text[next - 1] === '\n' || text[next - 1] === '<'
      } else {
        leftSafe = true
      }

      if (!inRight) {
        const pos = next + lastPCName.length
        rightSafe = text[pos] === '(' || text[pos] === '>'
      } else {
        rightSafe = true
      }

      console.log(222, leftSafe, rightSafe, inLeft, inRight)
      if (leftSafe && rightSafe) {
        changes.push({ from: next, to: next + lastPCName.length, insert: i.name })
      }
      pos = next + i.name.length
      // console.log(11111, next)
    }
    editor.dispatch({ changes })
  }
}

const reSinaNyaLine = /^<(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d)>\s+\[([^\]]+)\]:\s+([^\n]+)$/gm
const trySinaNyaLog = (text: string) => {
  let isSinaNyaLog = false
  let testText = text
  if (text.length > 2000) {
    testText = text.slice(0, 2000)
  }

  if (reSinaNyaLine.test(testText)) {
    isSinaNyaLog = true
  }

  // <2022-03-15 20:02:30.0>	[月歌]:	“锁上了么...”扭头看了看周围，看到了个在看假草的牧野，偷偷掏出螺丝刀尝试撬锁

  const startLength = store.pcList.length + 1001
  const nicknames = new Set<string>()

  if (isSinaNyaLog) {
    const items = [] as LogItem[]
    for (let i of text.split('\n')) {
      const m = reSinaNyaLine.exec(i)
      if (m) {
        const item = {} as LogItem
        nicknames.add(m[2])
        item.nickname = m[2]
        item.time = dayjs(m[1]).unix()
        item.message = m[3]
        item.IMUserId = startLength + nicknames.size
        items.push(item)
      }
    }
    loadLog(items)
    return true
  }

  return false
}

const trySealDice = (text: string) => {
  let isTrpgLog = false;
  try {
    const items = JSON.parse(text)
    if (items.length > 0) {
      const keys = Object.keys(items[0])
      isTrpgLog = keys.includes('isDice') && keys.includes('message')
    }

    if (isTrpgLog) {
      loadLog(items)
      return true
    }
  } catch (e) {
  }
  return false
}

const onChange = debounce(() => {
  const payloadText = store.editor.state.doc.toString()
  let isLog = false

  isLog = trySealDice(payloadText)
  if (isLog) return

  isLog = trySinaNyaLog(payloadText)
  if (isLog) return

  let ret = (payloadText as string).matchAll(reNameLine2)
  for (let i of ret) {
    store.tryAddPcList2(i[1])
  }
}, 500)

const reloadFunc = debounce(() => {
  store.reloadEditor()
  store.reloadEditor2()
  showPreview()
}, 500)

watch(store.pcList, reloadFunc, { deep: true })
watch(store.exportOptions, reloadFunc, { deep: true })
watch(isShowPreview, showPreview, { deep: true })

const exportOptions = store.exportOptions

const code = ref("")

</script>

<style lang="scss">
html {
  background: #f3f5f7;
}

.element-plus-logo {
  width: 50%;
}

.options > div {
  width: 30rem;
  max-width: 30rem;
  margin-bottom: 2rem;
}

.options > div > .switch {
  display: flex;
  align-items: center;
  justify-content: center;

  & > h4 {
    margin-left: 1rem;
  }
}

.el-color-picker__trigger {
  border: none !important;
}

.myLineDecoration {
  // background: lightblue;
  margin-bottom: 20px;
  font-size: large;
}

.pc-list {
  display: flex;
  align-items: center;
  flex-direction: column;
}

#app {
  overflow-y: auto;
}

.preview {
  background: #fff;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04);
}
</style>

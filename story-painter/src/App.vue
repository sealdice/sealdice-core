<template>
  <div style="width: 1000px; margin: 0 auto; max-width: 100%;">
    <h2 style="text-align: center;">海豹TRPG跑团Log着色器 V1.08</h2>
    <div style="text-align: center;">SealDice骰QQ群 524364253</div>
    <div style="text-align: center;"><b><el-link type="primary" target="_blank" href="https://dice.weizaima.com/">新骰系测试中</el-link></b>，快来提需求！</div>
    <div class="options" style="display: flex; flex-wrap: wrap; text-align: center;">
      <div>
        <div class="switch">
          <el-switch v-model="exportOptions.commandHide" />
          <h4>骰子指令过滤</h4>
        </div>
        <div>开启后，不显示pc指令，正常显示指令结果</div>
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
        <div>开启后，所有以(和（为开头的发言将被豹豹吃掉不显示</div>
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
          <el-button style="padding: 0 1rem " @click="store.pcList.splice(index, 1)" :disabled="isShowPreview || isShowPreviewBBS || isShowPreviewTRG">删除</el-button>

          <el-input
            :disabled="isShowPreview || isShowPreviewBBS || isShowPreviewTRG"
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
      style="margin-bottom: 0rem; display: flex; justify-content: center; align-items: center; flex-wrap: wrap;"
    >
      <el-button @click="exportRecordRaw">下载原始文件</el-button>
      <el-button @click="exportRecordQQ">下载QQ风格记录</el-button>
      <el-button @click="exportRecordIRC">下载IRC风格记录</el-button>
      <el-button @click="exportRecordDOCX">下载Word</el-button>
      <!-- <el-button @click="showPreview">预览</el-button> -->
      <div style="margin-left: 1rem;">
        <el-checkbox label="预览" v-model="isShowPreview" :border="true" @click="previewClick('preview')" />
        <el-checkbox label="论坛代码" v-model="isShowPreviewBBS" :border="true" @click="previewClick('bbs')" />
        <el-checkbox label="回声工坊" v-model="isShowPreviewTRG" :border="true" @click="previewClick('trg')" />
      </div>
    </div>

    <div style="text-align: center; margin-bottom: 2rem; margin-top: 0.5rem;">
      <div>提示: 海豹骰与回声工坊达成了合作，<el-link type="primary" target="_blank" href="https://github.com/DanDDXuanX/TRPG-Replay-Generator">回声工坊</el-link>可以将海豹的log一键转视频哦！</div>
      <div>回声工坊的介绍和视频教程看这里：<el-link type="primary" target="_blank" href="https://www.bilibili.com/video/BV1GY4y1H7wK/">B站传送门</el-link></div>
    </div>

    <code-mirror v-show="!(isShowPreview || isShowPreviewBBS || isShowPreviewTRG)" ref="editor" @change="onChange">
      <div style="z-index: 1000; position: absolute; right: 1rem">
        <div>
          <el-button @click="clearText" id="btnCopyPreviewBBS" style="" size="large" type="primary">清空内容</el-button>
        </div>
        <el-checkbox label="编辑器染色" v-model="store.doEditorHighlight" :border="false" @click.native="doEditorHighlightClick($event)" />
      </div>
    </code-mirror>

    <!-- <monaco-editor @change="onChange"/> -->
    <div class="preview" ref="preview" v-show="isShowPreview">
      <div v-if="previewItems.length === 0">
        <div>染色失败，内容为空或无法识别此格式。</div>
        <div>已知支持的格式有: 海豹Log(json)、赵/Dice!原始文件、塔原始文件</div>
        <div>请先清空编辑框，再行复制</div>
      </div>
      <div v-for="i in previewItems">
        <span
          style="color: #aaa"
          class="_time"
          v-if="!store.exportOptions.timeHide"
        >{{ timeSolve(i) }}</span>
        <span :style="{ 'color': i.color }" class="_nickname">{{ nicknameSolve(i) }}</span>
        <span :style="{ 'color': i.color }" v-html="previewMessageSolve(i)"></span>
      </div>
    </div>

    <div class="preview" ref="previewBBS" id="previewBBS" v-if="isShowPreviewBBS">
      <el-button @click="copied" id="btnCopyPreviewBBS" style="position: absolute; right: 1rem" size="large" data-clipboard-target="#previewBBS">一键复制</el-button>
      <div v-if="previewItems.length === 0">
        <div>染色失败，内容为空或无法识别此格式。</div>
        <div>已知支持的格式有: 海豹Log(json)、赵/Dice!原始文件、塔原始文件</div>
        <div>请先清空编辑框，再行复制</div>
      </div>
      <div v-for="i in previewItems">
        <span
          style="color: #aaa"
          class="_time"
          v-if="!store.exportOptions.timeHide"
        >[color=#silver]{{ timeSolve(i) }}[/color]</span>
        <span :style="{ 'color': i.color }">[color={{i.color ? i.color : '#fff'}}]
          <span class="_nickname">{{ nicknameSolve(i, 'bbs') }}</span>
          <span v-html="bbsMessageSolve(i)"></span>
        [/color]</span>
      </div>
    </div>

    <div style="margin-bottom: .5rem" v-if="isShowPreviewTRG">
      <el-checkbox :border="true" label="添加语音合成标记" v-model="isAddVoiceMark" />
      <!-- <el-checkbox label="回声工坊" v-model="isShowPreviewTRG2" /> -->
    </div>

    <div class="preview" ref="previewTRG" id="previewTRG" v-if="isShowPreviewTRG">
      <el-button @click="copied" id="btnCopyPreviewTRG" style="position: absolute; right: 1rem" size="large" data-clipboard-target="#previewTRG">一键复制</el-button>
      <div v-if="previewItems.length === 0">
        <div>染色失败，内容为空或无法识别此格式。</div>
        <div>已知支持的格式有: 海豹Log(json)、赵/Dice!原始文件、塔原始文件</div>
        <div>请先清空编辑框，再行复制</div>
      </div>
      <div v-for="i in previewItems" :style="i.isDice ? 'margin-top: 16px; margin-bottom: 16px' : ''">
        <span :style="{ 'color': i.color }" v-if="i.isDice"># </span>
        <span :style="{ 'color': i.color }" class="_nickname">{{ nicknameSolve(i, 'trg') }}</span>
        <span :style="{ 'color': i.color }" v-html="trgMessageSolve(i)"></span>
        <div v-if="store.itemById[i.id.toString()]" style="white-space: pre-wrap;">{{ trgCommandSolve(store.itemById[i.id.toString()]) }}</div>
        <span v-if="isAddVoiceMark && (!i.isDice)">{*}</span>
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
import { debounce, delay } from 'lodash-es'
import { convertToLogItems, exportFileRaw, exportFileQQ, exportFileIRC, exportFileDocx } from "./utils/exporter";
import { ElLoading, ElMessageBox, ElNotification, ElMessage } from "element-plus";
import { strFromU8, unzlibSync } from 'fflate';
import uaParser from 'ua-parser-js'
import { getTextWidth, getCanvasFontSize } from './utils'
import ClipboardJS from 'clipboard'

const isMobile = ref(false)
const downloadUsableRank = ref(0)

const preview = ref(null)

const isShowPreview = ref(false)
const isShowPreviewBBS = ref(false)
const isShowPreviewTRG = ref(false)

const isAddVoiceMark = ref(true)

const copied = () => {
  ElMessage.success('进行了复制！')
}

// 清空文本
const clearText = () => {
  store.editor.dispatch({
    changes: { from: 0, to: store.editor.state.doc.length, insert: '' }
  })
}

const readDiceNum = (expr: string, defaultVal = 100) => {
  let diceNum = defaultVal // 如果读不到，当作100处理
  const m = /[dD](\d+)/.exec(expr)
  if (m) {
    diceNum = parseInt(m[1])
  }
  return diceNum
}

const trgCommandSolve = (item: LogItem) => {
  if (item.commandInfo) {
    const ci = item.commandInfo
    if (ci.rule === 'coc7') {
      switch (ci.cmd) {
        case 'ra': {        
          let items = []
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.expr1)
            items.push(`(${ci.pcName}的${i.expr2},${diceNum},${i.checkVal},${i.attrVal})`)
          }
          return `<dice>:${items.join(',')}`
          break
        }
        case 'st': {
          // { "cmd": "st", "items": [ { "attr": "hp", "isInc": false, "modExpr": "1d4", "type": "mod", "valNew": 63, "valOld": 65 } ], "pcName": "木落", "rule": "coc7" }
          let items = []
          for (let i of ci.items) {
            if (i.attr == 'hp') {
              let maxNow = Math.max(i.valOld, i.valNew)
              items.push(`<hitpoint>:(${ci.pcName},${maxNow},${i.valOld},${i.valNew})`)
            }
            // let diceNum = readDiceNum(i.exprs[0])
            // items.push(`(${ci.pcName}的${i.exprs[0]},${diceNum},${i.sanOld},${i.checkVal})`)
          }
          const tip = '# 请注意，当前版本需要手动调整下方最大生命值(第二项)\n'
          return tip + `${items.join('\n')}`
          break
        }
        case 'sc': {
          // { "cmd": "sc", "cocRule": 11, "items": [ { "checkVal": 55, "exprs": [ "d100", "0", "1" ], "rank": -2, "sanNew": 0, "sanOld": 0 } ], "pcName": "木落", "rule": "coc7" }
          let items = []
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.exprs[0])
            items.push(`(${ci.pcName}的${i.exprs[0]},${diceNum},${i.sanOld},${i.checkVal})`)
          }
          return `<dice>:${items.join(',')}`
          break
        }
      }
    }
    if (ci.rule === 'dnd5e') {
      switch (ci.cmd) {
        case 'st': {
          // {"cmd":"st","items":[{"attr":"hp","isInc":false,"modExpr":"3","type":"mod","valNew":7,"valOld":10}],"pcName":"海岸线","rule":"dnd5e"}
          let items = []
          let hasHp = false
          for (let i of ci.items || []) {
            if (i.attr == 'hp') {
              let maxNow = Math.max(i.valOld, i.valNew)
              items.push(`<hitpoint>:(${ci.pcName},${maxNow},${i.valOld},${i.valNew})`)
              hasHp = true
            }
          }
          let tip = ''
          if (hasHp) {
            let tip = '# 请注意，当前版本需要手动调整下方最大生命值(第二项)\n'
          }
          return tip + `${items.join('\n')}`
          break
        }
        case 'rc': {
          // {"cmd":"rc","items":[{"expr":"D20 + 体质豁免","reason":"体质豁免","result":15}],"pcName":"阿拉密尔•利亚顿","rule":"dnd5e"}
          let items = []

          let tip = ''
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.expr, 20)
            items.push(`(${ci.pcName}的${i.reason}检定,${diceNum},NA,${Math.max(i.result, 20)})`)
            tip = '# 请注意，DND的最大面数可能为 D20+各种加值，需要手动二次调整\n'
          }
          return tip + `<dice>:${items.join(',')}`
          break
        }
      }
    }

    switch (ci.cmd) {
      case 'roll': {
          // { "cmd": "roll", "items": [ { "dicePoints": 100, "expr": "D100", "result": 30 } ], "pcName": "木落" }
          let items = []
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.expr)
            items.push(`(${ci.pcName}的${i.expr},${diceNum},NA,${i.result})`)
          }
          return `<dice>:${items.join(',')}`
          break
        }
    }
    return ci
  }
}

const previewMessageSolve = (i: LogItem) => {
  let msg = i.message
  const prefix = (!store.exportOptions.timeHide ? `${timeSolve(i)}` : '') + nicknameSolve(i)
  if (i.isDice) {
    msg = nameReplace(msg)
  }

  const length = getTextWidth(prefix, getCanvasFontSize(preview.value as any))
  // return msg.replaceAll('<br />', '\n').replaceAll('\n', '<br /> ' + `<span style="color:white">${prefix}</span>`)
  return msg.replaceAll('<br />', '\n').replaceAll(/\n([^\n]+)/g, `<p style="margin-left: ${length}px; margin-top: 0; margin-bottom: 0">$1</p>`)
}

const nameReplace = (msg: string) => {
  for (let i of store.pcList) {
    msg = msg.replaceAll(`<${i.name}>`, `${i.name}`)
  }
  return msg
}

const trgMessageSolve = (i: LogItem) => {
  let msg = i.message
  let extra = ''
  if (i.isDice) {
    msg = nameReplace(msg)
    extra = '# '
  }
  msg = msg.replaceAll('"', '').replaceAll('\\', '') // 移除反斜杠和双引号
  const prefix = isAddVoiceMark.value ? '{*}' : ''
  return msg.replaceAll('<br />', '\n').replaceAll('\n', prefix + '<br /> ' + extra + nicknameSolve(i, 'trg'))
}

const bbsMessageSolve = (i: LogItem) => {
  let msg = i.message
  if (i.isDice) {
    msg = nameReplace(msg)
  }
  return msg.replaceAll('<br />', '\n').replaceAll('\n', '[/color]<br /> ' + (!store.exportOptions.timeHide ? `<span style='color:#aaa'>[color=#silver]${timeSolve(i)}[/color]</span>` : '') + `[color=${i.color||'#fff'}] ` + nicknameSolve(i, 'bbs'))
}

const previewClick = (mode: 'preview' | 'bbs' | 'trg') => {
  switch (mode) {
    case 'preview':
      isShowPreviewBBS.value = false
      isShowPreviewTRG.value = false
      break;
    case 'bbs':
      isShowPreview.value = false
      isShowPreviewTRG.value = false
      break;
    case 'trg':
      isShowPreview.value = false
      isShowPreviewBBS.value = false
      break;
  }
}

watch(() => isShowPreviewBBS.value, (val: any) => {
  if (isShowPreviewBBS.value) {
    exportOptions.imageHide = true

    nextTick(() => {
      new ClipboardJS('#btnCopyPreviewBBS')
    })
    showPreview()
  }
})

watch(() => isShowPreviewTRG.value, (val: any) => {
  if (isShowPreviewTRG.value) {
    exportOptions.commandHide = true
    exportOptions.imageHide = true
    showPreview()
    nextTick(() => {
      new ClipboardJS('#btnCopyPreviewTRG')
    })
  }
})


function setupUA() {
  const parser = new uaParser.UAParser()
  parser.setUA(navigator.userAgent)
  const deviceType = parser.getDevice()

  const browser = parser.getBrowser().name
  downloadUsableRank.value = 1

  isMobile.value = deviceType.type === 'mobile'
  if (deviceType.type === 'mobile') {
    // 经测可以使用的
    switch (browser) {
      // case '360 Browser': // 手机360 但是手机360无特征，自己是Chrome WebView
      // 手机:X浏览器 Chrome WebView无特征
      case 'Edge':
      case 'Chrome':
      case 'Chromium':
      case 'Firefox':
      case 'MIUI Browser':
      case 'Opera':
        downloadUsableRank.value = 2
    }

  // 经测无法使用的
    switch (browser) {
      case 'baiduboxapp': // 手机:百度浏览器
      case 'QQBrowser': // 手机:搜狗浏览器极速版，手机:QQ浏览器
      // 手机:万能浏览器，Chrome WebView无特征，会直接崩溃
      case 'UCBrowser': // 手机:UC浏览器
      case 'Quark': // 手机:夸克
      // 手机:Via浏览器，Chrome WebView无特征，会直接崩溃    
      case 'QQ': // 手机:QQ
      case 'WeChat':
        downloadUsableRank.value = 0
    }
  }
}

setupUA()

let findPC = (name: string) => {
    // return _pcDict[name]
    for (let i of store.pcList) {
      if (i.name === name) {
        return i
      }
    }
}

const nicknameSolve = (i: LogItem, mode: 'bbs' | 'trg' | undefined = undefined) => {
  let userid = '(' + i.IMUserId + ')'
  const options = store.exportOptions
  if (options.userIdHide) {
    userid = ''
  }
  if (mode === 'bbs') {
    return `<${i.nickname}${userid}>`
  }
  if (mode === 'trg') {
    const u = findPC(i.nickname)
    let kpFlag = u?.role === '主持人' ? ',KP' : ''
    return `[${i.nickname}${kpFlag}]:`
  }
  // [张安翔]:最基本的对话行

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

const browserAlert = () => {
  if (downloadUsableRank.value === 0) {
    ElMessageBox.alert('你目前所使用的浏览器无法下载文件，请更换对标准支持较好的浏览器。建议使用Chrome/Firefox/Edge', '注意！', {
      type: 'error',
    })
  }
  if (downloadUsableRank.value === 1) {
      if (isMobile.value) {
        ElMessageBox.alert('你目前所使用的浏览器可能在下载文件时遇到乱码，或无法下载文件，最好更换对标准支持较好的浏览器。建议使用Chrome/Firefox/Edge', '提醒！', {
        type: 'warning',
      })
    }
  }
  // 2 不做提示 因为兼容良好
}

onMounted(async () => {
  const params = new Proxy(new URLSearchParams(window.location.search), {
    get: (searchParams, prop) => searchParams.get(prop as any)
  })
  const key = (params as any).key
  const password = location.hash.slice(1)

  const showHl = () => {
    setTimeout(() => {
      if (!isMobile.value) {
        store.doEditorHighlight = true
        store.reloadEditor()
        store.reloadEditor2()      
      }
    }, 1000)
  }

  if (key && password) {
    const loading = ElLoading.service({
      lock: true,
      text: '正在试图加载远程记录 ...',
      fullscreen: true,
      background: 'rgba(0, 0, 0, 0.7)',
    })

    try {
      const record = await store.tryFetchLog(key, password)
      // await new Promise<void>((resolve) => {
      //   new setTimeout(() => { resolve() }, 1000)
      // })
      const log = unzlibSync(Uint8Array.from(atob(record.data), c => c.charCodeAt(0)))

      nextTick(() => {
        const text = strFromU8(log)
        store.pcList.length = 0
        store.editor.dispatch({
          changes: { from: 0, to: store.editor.state.doc.length, insert: text }
        })
      })

      onChange()
      loading.close()
      showHl()
    } catch (e) {
      ElNotification({
        title: 'Error',
        message: '加载日志失败，可能是序号或密码不正确',
        type: 'error',
      })
      loading.close()
      browserAlert()
      return true
    }
  } else {
    showHl()
  }

  // cminstance.value = cmRefDom.value?.cminstance;
  // cminstance.value?.focus();

  // console.log(cminstance.value)
  browserAlert()
  onChange()
});

function exportRecordRaw() {
  browserAlert()
  exportFileRaw(store.editor.state.doc.toString())
}

function exportRecordQQ() {
  browserAlert()
  const results = convertToLogItems(store.editor.state.doc.toString(), store.pcList, store.exportOptions)
  exportFileQQ(results, store.exportOptions)
}

function exportRecordIRC() {
  browserAlert()
  const results = convertToLogItems(store.editor.state.doc.toString(), store.pcList, store.exportOptions)
  exportFileIRC(results, store.exportOptions)
}

function exportRecordDOCX() {
  browserAlert()
  if (isMobile.value) {
    ElMessageBox.alert('你当前处于移动端环境，已知只有WPS能够查看生成的Word文件，且无法看图！使用PC打开可以查看图片。', '提醒！', {
      type: 'warning',
    })
  }

  // 其实是伪doc
  previewClick('preview')
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
  // console.log("222", items)
  let text = ""
  let changed = false
  store.itemById = {}
  for (let i of items) {
    if (await store.tryAddPcList(i)) {
      changed = true
    }

    if (i.commandInfo) {
      store.itemById[i.id] = i
      // console.log(222, store.itemById[i.id])
    }

    let idSuffix = ''
    if (i.isDice) {
      idSuffix = ` #${i.id}`
    }

    const timeText = dayjs.unix(i.time).format('YYYY/MM/DD HH:mm:ss')
    text += `${i.nickname}(${i.IMUserId}) ${timeText}${idSuffix}\n${i.message}\n\n`
  }

  store.editor.dispatch({
    changes: { from: 0, to: store.editor.state.doc.length, insert: text }
  })
  store.items = items
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

      // console.log(222, leftSafe, rightSafe, inLeft, inRight)
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
    let lastItem = null as any as LogItem
    for (let i of text.split('\n')) {
      // 遇到了这个问题，见：
      // https://blog.csdn.net/Jioho_chen/article/details/122510522
      const m = /^<(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d)>\s+\[?([^\]]+)\]?:\s+([^\n]+)$/g.exec(i)

      if (m) {
        const item = {} as LogItem
        nicknames.add(m[2])
        item.nickname = m[2]
        item.time = dayjs(m[1]).unix()
        item.message = m[3]
        item.IMUserId = startLength + nicknames.size
        items.push(item)
        lastItem = item
      } else {
        if (lastItem) {
          lastItem.message += '\n'+i
        }
      }
    }
    // console.log(222222, items)
    loadLog(items)
    store.pcNameRefresh()
    return true
  }

  // console.log('log解析: 并非塔骰格式，尝试下一种')
  return false
}

// 2022-05-10 11:28:25 名字(12345)
const reQQExportLine = /^(\d{4}-\d{2}-\d{2} \d{1,2}:\d{1,2}:\d{1,2})\s+(.+?)(\([^)]+\)|\<[^>]+\>)$/gm
const tryQQExportLog = (text: string) => {
  let isQQExportLog = false
  let testText = text
  if (text.length > 2000) {
    testText = text.slice(0, 2000)
  }

  if (reQQExportLine.test(testText)) {
    isQQExportLog = true
  }

  const nicknames = new Set<string>()

  if (isQQExportLog) {
    const items = [] as LogItem[]
    let lastItem = null as any as LogItem
    for (let i of text.split('\n')) {
      // 遇到了这个问题，见：
      // https://blog.csdn.net/Jioho_chen/article/details/122510522
      const m = /^(\d{4}-\d{2}-\d{2} \d{1,2}:\d{1,2}:\d{1,2})\s+(.+?)(\([^)]+\)|\<[^>]+\>)$/g.exec(i)

      if (m) {
        if (lastItem) {
          if (lastItem.message.endsWith('\n')) {
            // 消除最后一个\n，避免从文件复制出来多一行的问题
            lastItem.message = lastItem.message.slice(0, -1)
          }
        }
        const item = {} as LogItem
        nicknames.add(m[2])
        item.nickname = m[2]
        item.time = dayjs(m[1]).unix()
        item.message = ''
        item.IMUserId = 'QQ:' + m[3].slice(1, -1)
        items.push(item)
        lastItem = item
      } else {
        if (lastItem) {
          if (lastItem.message === '') lastItem.message += i
          else lastItem.message += '\n'+i
        }
      }
    }
    // console.log(222222, items)
    loadLog(items)
    store.pcNameRefresh()
    return true
  }

  // console.log('log解析: 并非塔骰格式，尝试下一种')
  return false
}

const trySealDice = (text: string) => {
  let isTrpgLog = false;
  try {
    const sealFormat = JSON.parse(text)
    if (sealFormat.items && sealFormat.items.length > 0) {
      const keys = Object.keys(sealFormat.items[0])
      isTrpgLog = keys.includes('isDice') && keys.includes('message')
    }

    // console.log(3333, isTrpgLog, sealFormat.items, sealFormat.items.length > 0)
    if (isTrpgLog) {
      loadLog(sealFormat.items)
      store.pcNameRefresh()
      return true
    }
  } catch (e) {
    // console.log('log解析: 并非seal格式，尝试下一种')
    // console.log('??????', e)
  }
  return false
}

let preventNext = false
const onChange = debounce(() => {
  if (preventNext) {
    preventNext = false
    return
  }
  const payloadText = store.editor.state.doc.toString()
  let isLog = false

  isLog = trySealDice(payloadText)
  if (isLog) {
    preventNext = true
    return
  }

  isLog = trySinaNyaLog(payloadText)
  if (isLog) {
    preventNext = true
    return
  }

  isLog = tryQQExportLog(payloadText)
  if (isLog) {
    preventNext = true
    return
  }

  let ret = (payloadText as string).matchAll(reNameLine2)
  const names = new Set();
  const namesAll = new Set();
  const namesToDelete = new Set();

  for (let i of store.pcList) {
    namesAll.add(i.name)
  }

  for (let i of ret) {
    store.tryAddPcList2(i[1])
    names.add(i[1])
  }

  for (let i of namesAll) {
    if (!names.has(i)) {
      namesToDelete.add(i)
    }
  }

  for (let i of namesToDelete) {
    store.tryRemovePC(i as any)
  }
}, 500)

const reloadFunc = debounce(() => {
  store.reloadEditor()
  store.reloadEditor2()
  showPreview()
}, 500)

const doEditorHighlightClick = (e: any) => {
  // 因为原生click事件会执行两次，第一次在label标签上，第二次在input标签上，故此处理
  if (e.target.tagName === 'INPUT') return;

  const doHl = () => {
    // 编辑器染色
    setTimeout(() => {
      store.reloadEditor()
      store.reloadEditor2()
    }, 500)    
  }

  if (!store.doEditorHighlight) {
    // 如果要开启
    if (isMobile.value) {
      ElMessageBox.confirm(
        '部分移动设备上的特定浏览器可能会因为兼容性问题而卡死，继续吗？',
        '开启编辑器染色？',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning',
        }
      ).then(async () => {
        doHl()
      }).catch(() => {
        // 重新关闭
        setTimeout(() => {
          store.doEditorHighlight = false
          store.reloadEditor()
        }, 500)    
      })
      return
    }
  }

  doHl()
}

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
  word-break: break-all;
  background: #fff;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04);
  position: relative;
}
</style>

<template>
  <div style="width: 1000px; margin: 0 auto; max-width: 100%;">
    <h2 style="text-align: center;display: flex; align-items: center; justify-content: center; flex-flow: wrap;">
      <span>海豹TRPG跑团Log着色器 V2.0.5</span>
      <a style="margin:0 1rem" href="https://github.com/sealdice/story-painter" target="_blank"><img src="./assets/github-mark.svg" style="width: 1.2rem;"/></a>
      <el-button type="primary" @click="backV1">返回V1</el-button>
    </h2>
    <div style="text-align: center;">SealDice骰QQ群 524364253 / 562897832</div>
    <!-- <div style="text-align: center;"><b><el-link type="primary" target="_blank" href="https://dice.weizaima.com/">新骰系测试中</el-link></b>，快来提需求！</div> -->
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
          <el-switch v-model="exportOptions.offTopicHide" />
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
        <div style="display: flex; align-items: center; width: 26rem;">
          <el-button style="padding: 0 1rem " @click="deletePc(index, i)"
            :disabled="isShowPreview || isShowPreviewBBS || isShowPreviewTRG">删除</el-button>

          <el-input :disabled="isShowPreview || isShowPreviewBBS || isShowPreviewTRG" v-model="i.name" class="w-50 m-2"
            :prefix-icon="UserFilled" @focus="nameFocus(i)" @change="nameChanged(i)" />

          <el-input :disabled="true" v-model="i.IMUserId" style="width: 18rem" />

          <el-select v-model="i.role" class="m-2 w-60" style="width: 18rem">
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
      style="margin-bottom: 1rem; margin-top: 1rem; display: flex; justify-content: center; align-items: center; flex-wrap: wrap;">
      <div>
        <el-button @click="exportRecordRaw">下载原始文件</el-button>
        <el-button v-show="false" @click="exportRecordQQ">下载QQ风格记录</el-button>
        <el-button v-show="false" @click="exportRecordIRC">下载IRC风格记录</el-button>
        <el-button @click="exportRecordDOC">下载Word</el-button>
      </div>
      <!-- <el-button @click="showPreview">预览</el-button> -->
      <div style="margin-left: 1rem; ">
        <el-checkbox label="预览" v-model="isShowPreview" :border="true" @click="previewClick('preview')" />
        <el-checkbox label="论坛代码" v-model="isShowPreviewBBS" :border="true" @click="previewClick('bbs')" />
        <el-checkbox label="回声工坊" v-model="isShowPreviewTRG" :border="true" @click="previewClick('trg')" />
        <!-- <el-checkbox label="回声工坊" v-model="isShowPreviewTRG" :border="true" @click="previewClick('trg')" /> -->
      </div>
    </div>

    <code-mirror v-show="!(isShowPreview || isShowPreviewBBS || isShowPreviewTRG)" ref="editor" @change="onChange">
      <div style="z-index: 1000; position: absolute; right: 1rem">
        <div>
          <el-button @click="clearText" id="btnCopyPreviewBBS" style="" size="large" type="primary">清空内容</el-button>
        </div>
        <div>
          <el-button @click="doFlush" style="" size="large" type="primary">调试:Flush</el-button>
        </div>
        <el-checkbox label="编辑器染色" v-model="store.doEditorHighlight" :border="false"
          @click.native="doEditorHighlightClick($event)" />
      </div>
    </code-mirror>

    <preview-main :is-show="isShowPreview" :preview-items="previewItems"></preview-main>
    <preview-bbs :is-show="isShowPreviewBBS" :preview-items="previewItems"></preview-bbs>
    <preview-trg :is-show="isShowPreviewTRG" :preview-items="previewItems"></preview-trg>
  </div>

  <div style="margin-bottom: 3rem"></div>
</template>

<script setup lang="ts">
import { nextTick, ref, onMounted, watch, h, render, renderList } from "vue";
import { useStore } from './store'
import { UserFilled } from '@element-plus/icons-vue'
import CodeMirror from './components/CodeMirror.vue'
import { EditorState, StateEffect } from '@codemirror/state';
import { debounce, delay } from 'lodash-es'
import { exportFileRaw, exportFileQQ, exportFileIRC, exportFileDoc } from "./utils/exporter";
import { ElLoading, ElMessageBox, ElNotification, ElMessage, ElButton, ElCheckbox, ElColorPicker, ElInput, ElOption, ElSelect, ElSwitch } from "element-plus";
import { strFromU8, unzlibSync } from 'fflate';
import uaParser from 'ua-parser-js'

import { logMan } from './logManager/logManager'
import { ViewUpdate } from "@codemirror/view";
import { TextInfo } from "./logManager/importers/_logImpoter";
import previewMain from "./components/previews/preview-main.vue";
import previewBbs from "./components/previews/preview-bbs.vue";
import previewTrg from "./components/previews/preview-trg.vue";
import PreviewItem from './components/previews/preview-main-item.vue'
import { LogItem, CharItem, packNameId } from "./logManager/types";
import { setCharInfo } from './logManager/importers/_logImpoter'
import { msgCommandFormat, msgImageFormat, msgIMUseridFormat, msgOffTopicFormat } from "./utils";

const isMobile = ref(false)
const downloadUsableRank = ref(0)

const isShowPreview = ref(false)
const isShowPreviewBBS = ref(false)
const isShowPreviewTRG = ref(false)

const backV1 = () => {
  // location.href = location.origin + '/v1/' + location.search + location.hash;
  location.href = 'https://log.weizaima.com' + '/v1/' + location.search + location.hash;
}

// 清空文本
const clearText = () => {
  store.editor.dispatch({
    changes: { from: 0, to: store.editor.state.doc.length, insert: '' }
  })
}

const doFlush = () => {
  logMan.flush();
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
      store.exportOptions.imageHide = true
      break;
    case 'trg':
      isShowPreview.value = false
      isShowPreviewBBS.value = false
      store.exportOptions.imageHide = true
      break;
  }
  showPreview();
}

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

        logMan.lastText = '';
        logMan.syncChange(text, [0, store.editor.state.doc.length], [0, text.length])
        // store.editor.dispatch({
        //   changes: { from: 0, to: store.editor.state.doc.length, insert: text }
        // })
      });

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
    store.editor.dispatch({
      changes: { from: 0, to: store.editor.state.doc.length, insert: store.editor.state.doc.toString() }
    })
    showHl()
  }

  // cminstance.value = cmRefDom.value?.cminstance;
  // cminstance.value?.focus();
  // console.log(cminstance.value)
  browserAlert()
});

function exportRecordRaw() {
  browserAlert()
  exportFileRaw(store.editor.state.doc.toString())
}

function exportRecordQQ() {
  browserAlert()
  showPreview()
  exportFileQQ(previewItems.value, store.exportOptions)
}

function exportRecordIRC() {
  browserAlert()
  showPreview()
  exportFileIRC(previewItems.value, store.exportOptions)
}

function exportRecordDOC() {
  browserAlert()
  if (isMobile.value) {
    ElMessageBox.alert('你当前处于移动端环境，已知只有WPS能够查看生成的Word文件，且无法看图！使用PC打开可以查看图片。', '提醒！', {
      type: 'warning',
    })
  }

  const solveImg = (el: Element) => {
    if (el.tagName === 'IMG') {
      let width = el.clientWidth;
      let height = el.clientHeight;
      if (width === 0) {
        width = 300;
        height = 300;
      }
      el.setAttribute('width', `${width}`)
      el.setAttribute('height', `${height}`)
    }
    for (let i = 0; i < el.children.length; i += 1) {
      solveImg(el.children[i])
    }
  }

  const map = store.pcMap;
  const el = document.createElement('span');
  const elRoot = document.createElement('div');
  const items = [];

  showPreview()
  for (let i of previewItems.value) {
    if (i.isRaw) continue;
    const id = packNameId(i);
    if (map.get(id)?.role === '隐藏') continue;

    const html = h(PreviewItem, { source: i });
    render(html, el);

    const c = el;
    solveImg(c);
    items.push(c.innerHTML);
  }

  exportFileDoc(items.join('\n'));
}

const previewItems = ref<LogItem[]>([])

function showPreview() {
  let tmp = []
  let index = 0;
  const offTopicHide = store.exportOptions.offTopicHide;

  for (let i of logMan.curItems) {
    if (i.isRaw) continue;

    // // 处理ot
    // if (offTopicHide && !i.isDice) {
    //   const msg = i.message.replaceAll(/^[(（].+?$/gm, '') // 【
    //   if (msg.trim() === '') continue;
    // }
    let msg = msgImageFormat(i.message, store.exportOptions);
    msg = msgOffTopicFormat(msg, store.exportOptions, i.isDice);
    msg = msgCommandFormat(msg, store.exportOptions);
    msg = msgIMUseridFormat(msg, store.exportOptions, i.isDice);
    if (msg.trim() === '') continue;

    i.index = index;
    tmp.push(i);
    index += 1;
  }
  previewItems.value = tmp;
}

const store = useStore()
const color2 = ref('#409EFF')

// 修改ot选项后重建items
watch(() => store.exportOptions.offTopicHide, showPreview)

const deletePc = (index: number, i: CharItem) => {
  const now = Date.now();
  if (now - lastNameChange < 100) return;
  lastNameChange = now;

  ElMessageBox.confirm(
    `即将删除角色 <b>${i.name}</b> 及其全部发言，确定吗？`,
    '操作确认',
    {
      dangerouslyUseHTMLString: true,
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
  store.pcList.splice(index, 1);
    logMan.deleteByCharItem(i);
  }).catch(() => {
    i.name = lastPCName;
  })
}

let lastPCName = ''

const nameFocus = (i: CharItem) => {
  lastPCName = i.name
}

let lastNameChange = 0;
const nameChanged = (i: CharItem) => {
  const now = Date.now();
  if (now - lastNameChange < 100) return;
  lastNameChange = now;

  const oldName = lastPCName; // 这样做的原因是，如果按回车确认，那么 nameFocus 会在promise触发前触发一遍导致无效
  const newName = i.name;
  if (oldName && newName) {
    const el = document.createElement('span');

    render(h('span', `${oldName}`), el);
    const name1 = el.innerHTML;

    render(h('span', `${newName}`), el);
    const name2 = el.innerHTML;

    render(h('span', `<${oldName}>`), el);
    const name1w = el.innerHTML;

    render(h('span', `<${newName}>`), el);
    const name2w = el.innerHTML;

    ElMessageBox.confirm(
      `即将进行名字变更 <b>${name1} -> ${name2}</b><br />将修改信息行，并在文本中进行批量替换( ${name1w} 替换为 ${name2w} )，确定吗？`,
      '操作确认',
      {
        dangerouslyUseHTMLString: true,
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    ).then(async () => {
      logMan.rename(i, oldName, newName)
    }).catch(() => {
      i.name = oldName;
    })
  }
}


logMan.ev.on('textSet', (text) => {
  store.editor.dispatch({
    changes: { from: 0, to: store.editor.state.doc.length, insert: text }
  });

  let m = new Map<string, CharItem>();
  for (let i of logMan.curItems) {
    if (i.isRaw) continue;
    setCharInfo(m, i);
  }
  store.updatePcList(m);
});

logMan.ev.on('parsed', (ti: TextInfo) => {
  store.updatePcList(ti.charInfo);
})

const onChange = (v: ViewUpdate) => {
  let payloadText = '';
  if (v) {
    if (v.docChanged) {
      // 有一种我不太清楚的特殊情况会导致二次调用，从而使得pclist清零
      // 看不出明显变化，只是一个隐藏参数flags为0
      // 破案了，是flush
      if (!v.viewportChanged && (v as any).flags === 0) {
        return;
      }

      const ranges = (v as any).changedRanges;
      if (ranges.length) {
        for (let i = ranges.length - 1; i >= 0; i--) {
          const payloadText = store.editor.state.doc.toString()

          const r1 = [ranges[i].fromA, ranges[i].toA];
          const r2 = [ranges[i].fromB, ranges[i].toB];

          console.log('XXX', v, r1, r2);
          if (r1[0] === 0 && r1[1] === logMan.lastText.length) {
            console.log('全部文本被删除，清除pc列表');
            store.pcList = [];
          }
          logMan.syncChange(payloadText, r1, r2);
        }
      }
    }
  }

  // payloadText = store.editor.state.doc.toString()
  // let isLog = false
}

const doEditorHighlightClick = (e: any) => {
  // 因为原生click事件会执行两次，第一次在label标签上，第二次在input标签上，故此处理
  if (e.target.tagName === 'INPUT') return;

  const doHl = () => {
    // 编辑器染色
    setTimeout(() => {
      store.reloadEditor()
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


const reloadFunc = debounce(() => {
  store.reloadEditor()
}, 500)

watch(store.pcList, reloadFunc, { deep: true })
watch(store.exportOptions, reloadFunc, { deep: true })

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

.options>div {
  width: 30rem;
  max-width: 30rem;
  margin-bottom: 2rem;
}

.options>div>.switch {
  display: flex;
  align-items: center;
  justify-content: center;

  &>h4 {
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


.list-dynamic {
  width: 100%;
  height: 500px;
  overflow-y: auto;
}

.list-item-dynamic {
  // display: flex;
  // align-items: center;
  padding: 0.5em 0;
  border-color: lightgray;
}

.scroller {
  height: 95vh;
}
</style>

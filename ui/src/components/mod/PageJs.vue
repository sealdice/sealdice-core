<template>
  <el-tabs v-model="mode" class="demo-tabs">
    <el-tab-pane label="控制台" name="console"></el-tab-pane>
    <el-tab-pane label="插件列表" name="list"></el-tab-pane>
  </el-tabs>

  <div v-show="mode == 'console'">
    <p style="color: #999"><small>注意: 延迟执行的代码，其输出不会立即出现</small></p>
    <div>
      <div style="word-break: break-all; margin-bottom: 1rem; white-space: pre-line;">
        <div v-for="i in jsLines">{{ i }}</div>
      </div>

      <div ref="editorBox">
      </div>

      <div>
        <div style="margin-top: 1rem">
          <!-- <el-button @click="doSave">上传牌堆(json/yaml/zip)</el-button> -->
          <el-button @click="doExecute">执行代码</el-button>
          <el-button @click="jsReload">重载JS</el-button>
        </div>
      </div>
    </div>
  </div>

  <div v-show="mode == 'list'">
    <el-space style="margin-bottom: 2rem;">
      <el-upload action="" multiple accept="application/javascript, .js" class="upload" :before-upload="beforeUpload"
        :file-list="uploadFileList">
        <el-button type="">上传插件</el-button>
      </el-upload>

      <!-- <el-button @click="jsVisitDir">浏览目录</el-button> -->
      <el-button @click="jsReload">重载JS</el-button>
      <el-button><el-link href="https://github.com/sealdice/javascript" target="_blank">获取插件</el-link></el-button>
    </el-space>

    <el-space direction="vertical" :fill="true" wrap style="width: 100%">
      <div v-for="i, index in jsList">
        <el-descriptions :title="i.name" :border="false" class="js-item">
          <template #title>
            <span>{{ i.name }}</span>
            <el-button style="float:right" @click="doDelete(i, index)">删除</el-button>
          </template>
          <el-descriptions-item label="作者">{{ i.author || '<佚名>' }}</el-descriptions-item>
          <el-descriptions-item label="版本">{{ i.version || '<未定义>' }}</el-descriptions-item>
          <el-descriptions-item label="安装时间">{{ dayjs.unix(i.installTime).fromNow() }}</el-descriptions-item>
          <el-descriptions-item label="许可协议">{{ i.license || '<暂无>' }}</el-descriptions-item>
          <el-descriptions-item label="主页">{{ i.homepage || '<暂无>' }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ i.updateTime ? dayjs.unix(i.updateTime).fromNow() : '' || '<暂无>' }}</el-descriptions-item>
          <el-descriptions-item label="介绍" :span="3">{{ i.desc || '<暂无>' }}</el-descriptions-item>
          <el-descriptions-item label="报错信息" :span="3" v-if="i.errText">{{ i.errText }}</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-space>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount, onMounted, ref } from 'vue';
import { useStore } from '~/store'
import { urlBase } from '~/backend'
import filesize from 'filesize'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as dayjs from 'dayjs'

// import type { UploadProps, UploadUserFile } from 'element-plus'
import {
  Location,
  Document,
  Menu as IconMenu,
  Setting,
  CirclePlusFilled,
  CircleClose,
  QuestionFilled,
  BrushFilled
} from '@element-plus/icons-vue'
import { EditorView, basicSetup } from "codemirror"
import { javascript } from "@codemirror/lang-javascript"


const store = useStore()
const editorBox = ref(null);
const mode = ref('console');
let editor: EditorView

const jsLines = ref([] as string[])

const defaultText = [
  "// 学习制作可以看这里：https://github.com/sealdice/javascript/tree/main/examples",
  "// 下载插件可以看这里: https://github.com/sealdice/javascript/tree/main/scripts",
  "// 使用TypeScript，编写更容易 https://github.com/sealdice/javascript/tree/main/examples_ts",
  "// 目前可用于: 创建自定义指令，自定义COC房规，发送网络请求",
  "",
  "console.log('这是测试控制台');",
  "console.log('可以这样来查看变量详情：');",
  "console.log(Object.keys(seal));",
  "console.log('更多内容正在制作中...')",
  "console.log('注意: 测试版！API仍然可能发生重大变化！')",
  "// 写在这里的所有变量都是临时变量，如果你希望全局变量，使用 globalThis",
  "// 但是注意，全局变量在进程关闭后失效，想保存状态请存入硬盘。",
  "globalThis._test = 123;",
  "",
  ""
]

/** 执行指令 */
const doExecute = async () => {
  jsLines.value = [];
  const txt = editor.state.doc.toString();
  const data = await store.jsExec(txt);

  // 优先填充print输出
  const lines = []
  if (data.outputs) {
    lines.push(...data.outputs)
  }
  // 填充err或ret
  if (data.err) {
    lines.push(data.err)
  } else {
    lines.push(data.ret);
    try {
      (window as any).lastJSValue = data.ret;
      (globalThis as any).lastJSValue = data.ret;
    } catch (e) { }
  }
  jsLines.value = lines
}


let timerId: number

onMounted(async () => {
  const el = editorBox.value as any as HTMLElement;
  editor = new EditorView({
    extensions: [basicSetup, javascript(), EditorView.lineWrapping,],
    parent: el,
    doc: defaultText.join('\n'),
  })
  el.onclick = () => {
    editor.focus();
  }
  try {
    (globalThis as any).editor = editor;
  } catch (e) { }

  await refreshList();
  if (jsList.value.length > 0) {
    mode.value = 'list'
  }

  timerId = setInterval(async () => {
    console.log('refresh')
    const data = await store.jsGetRecord();

    if (data.outputs) {
      jsLines.value.push(...data.outputs)
    }
  }, 3000) as any;
})

onBeforeUnmount(() => {
  clearInterval(timerId)
})


const jsList = ref<JsScriptInfo[]>([]);
const uploadFileList = ref<any[]>([]);

const jsVisitDir = async () => {
  // 好像webui上没啥效果，先算了
  // await store.jsVisitDir();
}

const refreshList = async () => {
  const lst = await store.jsList();
  jsList.value = lst;
}

const jsReload = async () => {
  const ret = await store.jsReload()
  if (ret && ret.testMode) {
    ElMessage.success('展示模式无法重载脚本')
  } else {
    ElMessage.success('已重载')
    await refreshList()
  }
}

const beforeUpload = async (file: any) => { // UploadRawFile
  let fd = new FormData()
  fd.append('file', file)
  await store.jsUpload({ form: fd })
  refreshList();
  ElMessage.success('上传完成，请在全部操作完成后，手动重载插件')
}

const doDelete = async (data: any, index: number) => {
  ElMessageBox.confirm(
    `删除插件《${data.name}》，确定吗？`,
    '删除',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async (data) => {
    await store.jsDelete({ index })
    setTimeout(() => {
      // 稍等等再重载，以免出现没删掉
      refreshList()
    }, 1000);
    ElMessage.success('插件已删除，请手动重载后生效')
  })
}
</script>

<style lang="scss">
.cm-editor {
  /* height: v-bind("$props.initHeight"); */
  height: 20rem;
  // font-size: 18px;

  outline: 0 !important;
  /* height: 50rem; */
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04);
}

@media screen and (max-width: 700px) {
  .bak-item {
    flex-direction: column;

    &>span {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
  }
}

.deck-keys {
  display: flex;
  flex-flow: wrap;

  &>span {
    margin-right: 1rem;
    // width: fit-content;
  }
}

.upload {
  >ul {
    display: none;
  }
}

.js-item {
  .el-descriptions__label {
    font-weight: bolder;
  }

  .el-descriptions__title {
    flex: 1;
  }

  >.el-descriptions__body {
    padding: 1.5rem;
  }
}
</style>

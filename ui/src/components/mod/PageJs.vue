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
          <el-descriptions-item label="网站">{{ i.website || '<暂无>' }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ i.updateTime || '<暂无>' }}</el-descriptions-item>
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
  "console.log('这是测试控制台');",
  "console.log('可以这样来查看变量详情：');",
  "console.log(Object.keys(seal));",
  "console.log('更多内容正在制作中...')",
  "console.log('注意: 测试版！API仍然可能发生重大变化！')",
  "// 想要制作可以看这里：https://github.com/sealdice/javascript/tree/main/examples",
  "// 下载插件可以看这里: https://github.com/sealdice/javascript/tree/main/scripts",
  "// 使用TypeScript，编写更容易 https://github.com/sealdice/javascript/tree/main/examples_ts",
  "",
  "// 写在这里的所有变量都是临时变量，如果你希望全局变量，使用 globalThis",
  "// 但是注意，全局变量在进程关闭后失效，想保存状态请存入硬盘。",
  "globalThis._test = 123;",
  "",
  "",
  "// 如何建立一个扩展",
  "if (!seal.ext.find('test')) {",
  "  const ext = seal.ext.new('test', '木落', '1.0.0');",
  "  // 创建一个命令",
  "  const cmdSeal = seal.ext.newCmdItemInfo();",
  "  cmdSeal.name = 'seal';",
  "  cmdSeal.help = '召唤一只海豹，可用.seal <名字> 命名';",
  "  cmdSeal.solve = (ctx, msg, cmdArgs) => {",
  "    let val = cmdArgs.getArgN(1);",
  "    switch (val) {",
  "      case 'help': {",
  "        const ret = seal.ext.newCmdExecuteResult(true);",
  "        ret.showHelp = true;",
  "        return ret;",
  "      }",
  "      default: {",
  "        if (!val) val = '氪豹';",
  "        seal.replyToSender(ctx, msg, `你抓到一只海豹！取名为${val}\\n它的逃跑意愿为${Math.ceil(Math.random() * 100)}`)",
  "        return seal.ext.newCmdExecuteResult(true);",
  "      }",
  "    }",
  "  }",
  "  // 注册命令",
  "  ext.cmdMap['seal'] = cmdSeal;",
  "  // 注册扩展",
  "  seal.ext.register(ext);",
  "}",
  "",
  "// 写一个自定义COC规则",
  "rule = seal.coc.newRule()",
  "rule.index = 20 // 自定义序号必须大于等于20，可用.setcoc 20切换",
  "rule.key = '测试' // 可用 .setcoc 测试 切换",
  "rule.name = '自设规则' // 已切换至规则 name: desc",
  "rule.desc = '出1大成功\\n出100大失败'",
  "",
  "// d100 为出目，checkValue 为技能点数",
  "rule.check = (ctx, d100, checkValue) => {",
  "  let successRank = 0",
  "  const criticalSuccessValue = 1",
  "  const fumbleValue = 100",
  "",
  "  if (d100 <= checkValue) {",
  "    successRank = 1",
  "  } else {",
  "    successRank = -1",
  "  }",
  "",
  "  // 成功判定",
  "  if (successRank == 1) {",
  "    // 区分大成功、困难成功、极难成功等",
  "    if (d100 <= checkValue/2) {",
  "      //suffix = \"成功(困难)\"",
  "      successRank = 2",
  "    }",
  "    if (d100 <= checkValue/5) {",
  "      //suffix = \"成功(极难)\"",
  "      successRank = 3",
  "    }",
  "    if (d100 <= criticalSuccessValue) {",
  "      //suffix = \"大成功！\"",
  "      successRank = 4",
  "    }",
  "  } else {",
  "    if (d100 >= fumbleValue) {",
  "      //suffix = \"大失败！\"",
  "      successRank = -2",
  "    }",
  "  }",
  "",
  "  let ret = seal.coc.newRuleCheckResult()",
  "  ret.successRank = successRank",
  "  ret.criticalSuccessValue = criticalSuccessValue",
  "  return ret",
  "}",
  "",
  "// 返回值为bool，代表成功或失败，失败一般是name或index重复",
  "seal.coc.registerRule(rule)",
  "",
  "",
  "// 不支持 async/await 但支持promise",
  "// 推荐使用ts编译到js使用",
  "console.log('\\n发送网络请求:')",
  "fetch('https://jsonplaceholder.typicode.com/users').then((resp) => {",
  "  resp.json().then((users) => {",
  "    for (let i of users.slice(0, 3)) {",
  "      console.log(i.name);",
  "    }",
  "  });",
  "})",
  "console.log('网络请求文本可能延迟出现，会在日志界面显示。');",
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

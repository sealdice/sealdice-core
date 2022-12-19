<template>
  <h2>控制台</h2>
  <div>
    <div style="word-break: break-all; margin-bottom: 1rem; white-space: pre-line;">
      <div v-for="i in jsLines">{{i}}</div>
    </div>

    <div ref="editorBox">
    </div>

    <div>
      <div style="margin-top: 1rem">
        <!-- <el-button @click="doSave">上传牌堆(json/yaml/zip)</el-button> -->
        <el-button @click="doExecute">执行代码</el-button>
        <el-button @click="doReload">重载JS</el-button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount, onMounted, ref } from 'vue';
import { useStore } from '~/store'
import { urlBase } from '~/backend'
import filesize from 'filesize'
import { ElMessage, ElMessageBox } from 'element-plus'
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
let editor: EditorView

const jsLines = ref([] as string[])

const doReload = async () => {
  await store.jsReload();
}

const defaultText = [
    "console.log('这是测试控制台');",
    "console.log('可以这样来查看变量详情：');",
    "console.log(Object.keys(seal));",
    "console.log('更多内容正在制作中...')",
    "console.log('注意: 测试版！API仍然可能发生重大变化！')",
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
    } catch (e) {}
  }
  jsLines.value = lines
}


onMounted(async () => {
  const el = editorBox.value as any as HTMLElement;
  editor = new EditorView({
    extensions: [basicSetup, javascript(), EditorView.lineWrapping, ],
    parent: el,
    doc: defaultText.join('\n'),
  })
  el.onclick = () => {
    editor.focus();
  }
  try {
    (globalThis as any).editor = editor;
  } catch (e) {}
})
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
    & > span {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
  }
}

.deck-keys {
  display:flex;
  flex-flow: wrap;

  & > span {
    margin-right: 1rem;
    // width: fit-content;
  }
}
</style>

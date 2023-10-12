<template>
  <header class="page-header">
    <el-switch v-model="jsEnable" active-text="启用" inactive-text="关闭" />
    <el-button v-show="jsEnable" @click="jsReload" type="primary" :icon="Refresh">重载JS</el-button>
  </header>

  <el-affix :offset="70" v-if="needReload">
    <div class="tip-danger">
      <el-text type="danger" size="large" tag="strong">存在修改，需要重载后生效！</el-text>
    </div>
  </el-affix>

  <el-row>
    <el-col :span="24">
      <el-tabs v-model="mode" class="demo-tabs" :stretch=true>
        <el-tab-pane label="控制台" name="console">
          <div>
            <div ref="editorBox">
            </div>
            <div>
              <div style="margin-top: 1rem">
                <el-button @click="doExecute" type="success" :icon="CaretRight" :disabled="!jsEnable">执行代码</el-button>
              </div>
            </div>
            <el-text type="danger" tag="p" style="padding: 1rem 0;">注意：延迟执行的代码，其输出不会立即出现</el-text>
            <div style="word-break: break-all; margin-bottom: 1rem; white-space: pre-line;">
              <div v-for="i in jsLines">{{ i }}</div>
            </div>
          </div>
        </el-tab-pane>
        <el-tab-pane label="插件列表" name="list">
          <header class="js-list-header">
            <el-space>
              <el-upload action="" multiple accept="application/javascript, .js" class="upload"
                :before-upload="beforeUpload" :file-list="uploadFileList">
                <el-button type="primary" :icon="Upload">上传插件</el-button>
              </el-upload>
              <el-button type="info" :icon="Search" size="small" link tag="a" target="_blank"
                style="text-decoration: none;" href="https://github.com/sealdice/javascript">获取插件</el-button>
            </el-space>
          </header>
          <main class="js-list-main">
            <el-card class="js-item" v-for="i, index in jsList" :key="index" shadow="hover">
              <template #header>
                <div class="js-item-header">
                  <el-space>
                    <el-switch v-model="i.enable" @change="changejsScriptStatus(i.name, i.enable)"
                      style="--el-switch-on-color: #67C23A; --el-switch-off-color: #F56C6C" />
                    <el-text size="large" tag="b">{{ i.name }}</el-text>
                    <el-text>{{ i.version || '&lt;未定义>' }}</el-text>
                  </el-space>
                  <el-space>
                    <el-popconfirm v-if="i.updateUrls && i.updateUrls.length > 0" width="220"
                                   confirm-button-text="确认"
                                   cancel-button-text="取消"
                                   @confirm="doCheckUpdate(i, index)"
                                   title="更新地址由插件作者提供，是否确认要检查该插件更新？">
                      <template #reference>
                        <el-button :icon="Download" type="success" size="small" plain :loading="diffLoading">更新</el-button>
                      </template>
                    </el-popconfirm>
<!--                    <el-button :icon="Setting" type="primary" size="small" plain @click="showSettingDialog = true">设置</el-button>-->
                    <el-button @click="doDelete(i, index)" :icon="Delete" type="danger" size="small" plain>删除</el-button>
                  </el-space>
                </div>
              </template>
              <el-descriptions>
                <el-descriptions-item :span="3" label="作者">{{ i.author || '&lt;佚名>' }}</el-descriptions-item>
                <el-descriptions-item :span="3" label="介绍">{{ i.desc || '&lt;暂无>' }}</el-descriptions-item>
                <el-descriptions-item :span="3" label="主页">{{ i.homepage || '&lt;暂无>' }}</el-descriptions-item>
                <el-descriptions-item label="许可协议">{{ i.license || '&lt;暂无>' }}</el-descriptions-item>
                <el-descriptions-item label="安装时间">{{ dayjs.unix(i.installTime).fromNow() }}</el-descriptions-item>
                <el-descriptions-item label="更新时间">
                  {{ i.updateTime ? dayjs.unix(i.updateTime).fromNow() : '' || '&lt;暂无>' }}
                </el-descriptions-item>
                <el-descriptions-item label="报错信息" :span="3" v-if="i.errText">{{ i.errText }}</el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-dialog v-model="showDiff" title="插件内容对比" class="diff-dialog">
              <diff-viewer lang="javascript" :old="jsCheck.old" :new="jsCheck.new"/>
              <template #footer>
                <el-space wrap>
                  <el-button @click="showDiff = false">取消</el-button>
                  <el-button v-if="!(jsCheck.old === jsCheck.new)" type="success" :icon="DocumentChecked" @click="jsUpdate">确认更新</el-button>
                </el-space>
              </template>
            </el-dialog>

<!--            <el-dialog v-model="showSettingDialog" title="设置项">-->
<!--              <el-form :model="settingForm">-->
<!--                <el-form-item v-for="p of settingForm.props" :key="p.key" :label="p.name ?? p.key">-->
<!--                  <el-input v-model="p.value"/>-->
<!--                </el-form-item>-->
<!--              </el-form>-->
<!--              <template #footer>-->
<!--            <span class="dialog-footer">-->
<!--              <el-button @click="showSettingDialog = false">取消</el-button>-->
<!--              <el-button type="primary" @click="showSettingDialog = false">-->
<!--                提交-->
<!--              </el-button>-->
<!--            </span>-->
<!--              </template>-->
<!--            </el-dialog>-->
          </main>
        </el-tab-pane>
      </el-tabs>
    </el-col>

  </el-row>
</template>

<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useStore } from '~/store'
import { ElMessage, ElMessageBox } from 'element-plus'
import {Refresh, CaretRight, Upload, Search, Delete, Setting, Download, DocumentChecked} from '@element-plus/icons-vue'
import * as dayjs from 'dayjs'
import { EditorView, basicSetup } from "codemirror"
import { javascript } from "@codemirror/lang-javascript"
import DiffViewer from "~/components/mod/diff-viewer.vue";

const store = useStore()
const jsEnable = ref(false)
const editorBox = ref(null);
const mode = ref('console');
const needReload = ref(false)
let editor: EditorView

const jsLines = ref([] as string[])

const defaultText = [
  "// 学习制作可以看这里：https://github.com/sealdice/javascript/tree/main/examples",
  "// 下载插件可以看这里: https://github.com/sealdice/javascript/tree/main/scripts",
  "// 使用TypeScript，编写更容易 https://github.com/sealdice/javascript/tree/main/examples_ts",
  "// 目前可用于: 创建自定义指令，自定义COC房规，发送网络请求，读写本地数据",
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
  "let ext = seal.ext.find('test');",
  "if (!ext) {",
  "  ext = seal.ext.new('test', '木落', '1.0.0');",
  "  seal.ext.register(ext);",
  "}",
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
  jsEnable.value = await jsStatus()
  watch(jsEnable, async (newStatus, oldStatus) => {
    console.log("new:", newStatus, " old:", oldStatus)
    if (oldStatus !== undefined) {
      if (newStatus) {
        console.log("reload")
        await jsReload()
      } else {
        console.log("shutdown")
        await jsShutdown()
      }
    }
  })

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

const jsStatus = async () => {
  return store.jsStatus()
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
    needReload.value = false
  }
  jsEnable.value = await jsStatus()
}

const jsShutdown = async () => {
  const ret = await store.jsShutdown()
  if (ret?.testMode) {
    ElMessage.success('展示模式无法关闭')
  } else if (ret?.result === true) {
    ElMessage.success('已关闭JS支持')
    jsLines.value = []
    await refreshList()
  }
  jsEnable.value = await jsStatus()
}

const beforeUpload = async (file: any) => { // UploadRawFile
  let fd = new FormData()
  fd.append('file', file)
  await store.jsUpload({ form: fd })
  refreshList();
  ElMessage.success('上传完成，请在全部操作完成后，手动重载插件')
  needReload.value = true
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
    needReload.value = true
  })
}

const changejsScriptStatus = async (name: string, status: boolean) => {
  if (status) {
    const ret = await store.jsEnable({ name })
    setTimeout(() => {
      refreshList()
    }, 1000);
    if (ret.result) {
      ElMessage.success('插件已启用，请手动重载后生效')
    }
  } else {
    const ret = await store.jsDisable({ name })
    setTimeout(() => {
      refreshList()
    }, 1000);
    if (ret.result) {
      ElMessage.success('插件已禁用，请手动重载后生效')
    }
  }
  needReload.value = true
  return true
}

const showSettingDialog = ref<boolean>(false)

interface DeckProp {
  key:  string
  value: string

  name?: string
  desc?: string
  required?: boolean
  default?: string
}

const settingForm = ref({
  props: [{key: "name", value: "test props"}] as DeckProp[]
})

const showDiff = ref<boolean>(false)
const diffLoading = ref<boolean>(false)

interface JsCheckResult {
  old: string,
  new: string,
  tempFileName: string,
  index: number,
}

const jsCheck = ref<JsCheckResult>({
  old: "",
  new: "",
  tempFileName: "",
  index: -1
})

const doCheckUpdate = async (data: any, index: number) => {
  diffLoading.value = true
  const checkResult = await store.jsCheckUpdate({ index });
  diffLoading.value = false
  if (checkResult.result) {
    jsCheck.value = { ...checkResult, index }
    showDiff.value = true
  } else {
    ElMessage.error('检查更新失败！' + checkResult.err)
  }
}

const jsUpdate = async () => {
  const res = await store.jsUpdate(jsCheck.value);
  if (res.result) {
    showDiff.value = false
    needReload.value = true
    setTimeout(() => {
      refreshList()
    }, 1000)
    ElMessage.success('更新成功，请手动重载后生效')
  } else {
    showDiff.value = false
    ElMessage.error('更新失败！' + res.err)
  }
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

.js-list-header {
  margin-bottom: 1rem;
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}

.js-list-main {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem
}

.js-item-header {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}

.js-item-header {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}

.js-item {
  min-width: 100%;
}
</style>

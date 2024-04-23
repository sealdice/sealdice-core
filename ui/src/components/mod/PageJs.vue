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
  <el-affix :offset="70" v-if="jsConfigEdited">
    <div class="tip-danger">
      <el-text type="danger" size="large" tag="strong">配置内容已修改，不要忘记保存！</el-text>
        <el-button class="button" type="primary" :icon="DocumentChecked" :disabled="!jsConfigEdited" @click="doJsConfigSave()">点我保存</el-button>
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
            <el-card class="js-item" v-for="(i, index) of jsList" :key="index" shadow="hover">
              <template #header>
                <div class="js-item-header">
                  <template v-if="!i.errText">
                    <el-space>
                      <el-switch v-model="i.enable" @change="changejsScriptStatus(i.name, i.enable)" :disabled="i.errText !== ''"
                                 style="--el-switch-on-color: var(--el-color-success); --el-switch-off-color: var(--el-color-danger)" />
                      <el-text size="large" tag="b">{{ i.name }}</el-text>
                      <el-text>{{ i.version || '&lt;未定义>' }}</el-text>
                      <el-tag v-if="i.official" size="small" type="success">官方</el-tag>
                    </el-space>
                    <el-space>
                      <el-button v-if="i.official && i.updateUrls && i.updateUrls.length > 0"
                                 :icon="Download" type="success" size="small" plain :loading="diffLoading">更新</el-button>
                      <el-popconfirm v-else-if="i.updateUrls && i.updateUrls.length > 0" width="220"
                                     confirm-button-text="确认"
                                     cancel-button-text="取消"
                                     @confirm="doCheckUpdate(i, index)"
                                     title="更新地址由插件作者提供，是否确认要检查该插件更新？">
                        <template #reference>
                          <el-button :icon="Download" type="success" size="small" plain :loading="diffLoading">更新</el-button>
                        </template>
                      </el-popconfirm>
                      <!--                    <el-button :icon="Setting" type="primary" size="small" plain @click="showSettingDialog = true">设置</el-button>-->
                      <el-button v-if="i.builtin && i.builtinUpdated" @click="doDelete(i, index)" :icon="Delete" type="danger" size="small" plain>卸载更新</el-button>
                      <el-button v-else-if="!i.builtin" @click="doDelete(i, index)" :icon="Delete" type="danger" size="small" plain>删除</el-button>
                    </el-space>
                  </template>
                  <template v-else>
                    <el-space alignment="center">
                      <el-icon size="20" color="var(--el-color-danger)"><circle-close/></el-icon>
                      <del>
                        <el-text size="large" tag="b">{{ i.filename }}</el-text>
                      </del>
                    </el-space>
                    <el-space>
                      <el-button v-if="i.builtin && i.builtinUpdated" @click="doDelete(i, index)" :icon="Delete" type="danger" size="small">卸载更新</el-button>
                      <el-button v-else-if="!i.builtin" @click="doDelete(i, index)" :icon="Delete" type="danger" size="small">删除</el-button>
                    </el-space>
                  </template>
                </div>
              </template>

              <el-descriptions style="white-space:pre-line;">
                <template v-if="!i.errText">
                  <el-descriptions-item v-if="!i.official" :span="3" label="作者">{{ i.author || '&lt;佚名>' }}</el-descriptions-item>
                  <el-descriptions-item :span="3" label="介绍">{{ i.desc || '&lt;暂无>' }}</el-descriptions-item>
                  <el-descriptions-item v-if="!i.official" :span="3" label="主页">{{ i.homepage || '&lt;暂无>' }}</el-descriptions-item>
                  <el-descriptions-item label="许可协议">{{ i.license || '&lt;暂无>' }}</el-descriptions-item>
                  <el-descriptions-item label="安装时间">{{ dayjs.unix(i.installTime).fromNow() }}</el-descriptions-item>
                  <el-descriptions-item label="更新时间">
                    {{ i.updateTime ? dayjs.unix(i.updateTime).fromNow() : '' || '&lt;暂无>' }}
                  </el-descriptions-item>
                </template>
                <template v-else>
                  <el-descriptions-item label="错误信息">
                    <el-text type="danger">{{ i.errText }}</el-text>
                  </el-descriptions-item>
                </template>
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
          </main>
        </el-tab-pane>

        <el-tab-pane label="插件设置" name="config">
          <main>
            <div v-if="size(jsConfig as Map<any,any>) === 0" style="display: flex; justify-content: center">
              <el-text size="large" tag="strong">暂无设置项</el-text>
            </div>
            <el-collapse v-else class="js-list-main" style="margin-top: 0.5rem;">
              <el-collapse-item class="js-item" v-for="(config, i) in jsConfig" :key="i">
                <template #title>
                  <div class="js-item-header">
                    <el-space>
                      <el-text size="large" tag="strong" style="margin-left: 1rem">{{ (config as unknown as JsPluginConfig)['pluginName'] }}</el-text>
                    </el-space>
                  </div>
                </template>
                <el-card shadow="never" style="border: 0;">
                  <el-form v-for="(c, index) in (config as unknown as JsPluginConfig)['configs']" :key="index">
                    <template #header>
                      <div class="js-item-header">
                        <el-space>
                          <el-text size="large">{{ (c as unknown as JsPluginConfigItem).key }}</el-text>
                        </el-space>
                      </div>
                    </template>
                    <el-form-item v-if="(c as unknown as JsPluginConfigItem).type == 'string'" style="width: 100%; margin-bottom: .5rem;">
                      <el-form-item label="字符串配置项:">{{(c as unknown as JsPluginConfigItem).key}}</el-form-item><br/>
                      <div style="width: 100%"><el-text>{{ (c as unknown as JsPluginConfigItem).description }}</el-text></div>
                      <div style="width: 100%; margin-bottom: .5rem;">
                        <el-input type="textarea" v-model="(c as unknown as JsPluginConfigItem).value" @change="doJsConfigChanged()"></el-input>
                      </div>
                      <template v-if="(c as unknown as JsPluginConfigItem).value !== (c as unknown as JsPluginConfigItem).defaultValue">
                        <el-tooltip content="重置为初始值" placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doResetJsConfig((config as unknown as JsPluginConfig)['pluginName'],(c as unknown as JsPluginConfigItem).key)">
                            <brush-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <template v-if="(c as unknown as JsPluginConfigItem).deprecated">
                        <el-tooltip content="移除 - 这个配置在新版的默认配置中不被使用，<br />但升级而来时仍可能被使用，请确认无用后删除" raw-content
                                    placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doDeleteUnusedConfig((config as unknown as JsPluginConfig)['pluginName'], (c as unknown as JsPluginConfigItem).key)">
                            <delete-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                    </el-form-item>
                    <el-form-item v-if="(c as unknown as JsPluginConfigItem).type == 'int'">
                      <el-form-item label="整数配置项:">{{(c as unknown as JsPluginConfigItem).key}}</el-form-item><br/>
                      <div style="width: 100%"><el-text>{{ (c as unknown as JsPluginConfigItem).description }}</el-text></div>
                      <el-form-item :span="30">
                        <div style="margin-left: 1rem;">
                          <el-input-number v-model="(c as unknown as JsPluginConfigItem).value" type="number" @change="doJsConfigChanged()"></el-input-number>
                        </div>
                      </el-form-item>
                      <template v-if="(c as unknown as JsPluginConfigItem).value !== (c as unknown as JsPluginConfigItem).defaultValue">
                        <el-tooltip content="重置为初始值" placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doResetJsConfig((config as unknown as JsPluginConfig)['pluginName'],(c as unknown as JsPluginConfigItem).key)">
                            <brush-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <template v-if="(c as unknown as JsPluginConfigItem).deprecated">
                        <el-tooltip content="移除 - 这个配置在新版的默认配置中不被使用，<br />但升级而来时仍可能被使用，请确认无用后删除" raw-content
                                    placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doDeleteUnusedConfig((config as unknown as JsPluginConfig)['pluginName'], (c as unknown as JsPluginConfigItem).key)">
                            <delete-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                    </el-form-item>
                    <el-form-item v-if="(c as unknown as JsPluginConfigItem).type == 'float'">
                      <el-form-item label="浮点数配置项:">{{(c as unknown as JsPluginConfigItem).key}}</el-form-item><br/>
                      <div style="width: 100%"><el-text>{{ (c as unknown as JsPluginConfigItem).description }}</el-text></div>
                      <el-form-item :span="30">
                        <div style="margin-left: 1rem;">
                          <el-input-number v-model="(c as unknown as JsPluginConfigItem).value" type="number" @change="doJsConfigChanged()"></el-input-number>
                        </div>
                      </el-form-item>
                      <template v-if="(c as unknown as JsPluginConfigItem).value !== (c as unknown as JsPluginConfigItem).defaultValue">
                        <el-tooltip content="重置为初始值" placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doResetJsConfig((config as unknown as JsPluginConfig)['pluginName'],(c as unknown as JsPluginConfigItem).key)">
                            <brush-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <template v-if="(c as unknown as JsPluginConfigItem).deprecated">
                        <el-tooltip content="移除 - 这个配置在新版的默认配置中不被使用，<br />但升级而来时仍可能被使用，请确认无用后删除" raw-content
                                    placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doDeleteUnusedConfig((config as unknown as JsPluginConfig)['pluginName'], (c as unknown as JsPluginConfigItem).key)">
                            <delete-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                    </el-form-item>
                    <el-form-item v-if="(c as unknown as JsPluginConfigItem).type == 'bool'">
                      <el-form-item label="布尔配置项:">{{(c as unknown as JsPluginConfigItem).key}}</el-form-item><br/>
                      <div style="width: 100%"><el-text>{{ (c as unknown as JsPluginConfigItem).description }}</el-text></div>
                      <el-form-item :span="30" >
                        <div style="margin-left: 1rem;">
                          <el-switch v-model="(c as unknown as JsPluginConfigItem).value" @change="doJsConfigChanged()"></el-switch>
                        </div>
                      </el-form-item>
                      <template v-if="(c as unknown as JsPluginConfigItem).value !== (c as unknown as JsPluginConfigItem).defaultValue">
                        <el-tooltip content="重置为初始值" placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doResetJsConfig((config as unknown as JsPluginConfig)['pluginName'],(c as unknown as JsPluginConfigItem).key)">
                            <brush-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <template v-if="(c as unknown as JsPluginConfigItem).deprecated">
                        <el-tooltip content="移除 - 这个配置在新版的默认配置中不被使用，<br />但升级而来时仍可能被使用，请确认无用后删除" raw-content
                                    placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doDeleteUnusedConfig((config as unknown as JsPluginConfig)['pluginName'], (c as unknown as JsPluginConfigItem).key)">
                            <delete-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                    </el-form-item>
                    <el-form-item v-if="(c as unknown as JsPluginConfigItem).type == 'template'" style="width: 100%; margin-bottom: .5rem;">
                      <el-form-item label="模板配置项:" style="width: 100%; margin-bottom: .5rem;">{{(c as unknown as JsPluginConfigItem).key}}</el-form-item><br/>
                      <div style="width: 100%"><el-text>{{ (c as unknown as JsPluginConfigItem).description }}</el-text></div>
                      <template v-if="!isEqual((c as unknown as JsPluginConfigItem).value, (c as unknown as JsPluginConfigItem).defaultValue)">
                        <el-tooltip content="重置为初始值" placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doResetJsConfig((config as unknown as JsPluginConfig)['pluginName'],(c as unknown as JsPluginConfigItem).key)">
                            <brush-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <template v-if="(c as unknown as JsPluginConfigItem).deprecated">
                        <el-tooltip content="移除 - 这个配置在新版的默认配置中不被使用，<br />但升级而来时仍可能被使用，请确认无用后删除" raw-content
                                    placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doDeleteUnusedConfig((config as unknown as JsPluginConfig)['pluginName'], (c as unknown as JsPluginConfigItem).key)">
                            <delete-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <el-form-item style="width: 100%; margin-bottom: .5rem;">
                        <div v-for="(d, index) in (c as unknown as JsPluginConfigItem).value" :key="index" style="width: 100%; margin-bottom: .5rem;">
                          <!-- 这里面是单条修改项 -->
                          <el-row>
                            <el-col style="width: 100%; margin-bottom: .5rem;">
                            <span style="width: 100%;">
                              <el-input type="textarea" v-model="((c as unknown as JsPluginConfigItem).value)[index]" :autosize="true" @change="doJsConfigChanged()"></el-input>
                            </span>
                            </el-col>
                            <el-col :span="5">
                              <div style="display: flex; align-items: center; width: 1.3rem; margin-left: 1rem; margin-top: .5rem">
                                <el-tooltip :content="index === 0 ? '点击添加一项' : '点击删除你不想要的配置项'" placement="bottom-start">
                                  <el-icon>
                                    <circle-plus-filled v-if="index == 0" @click="doJsConfigAddItem((c as unknown as JsPluginConfigItem).value)" />
                                    <circle-close v-else @click="doJsConfigRemoveItemAt((c as unknown as JsPluginConfigItem).value, index)" />
                                  </el-icon>
                                </el-tooltip>
                              </div>
                            </el-col>
                          </el-row>
                        </div>
                      </el-form-item>
                    </el-form-item>
                    <el-form-item v-if="(c as unknown as JsPluginConfigItem).type == 'option'">
                      <el-form-item label="选项配置项:" style="width: 100%; margin-bottom: .5rem;">{{(c as unknown as JsPluginConfigItem).key}}</el-form-item>
                      <div style="width: 100%"><el-text>{{ (c as unknown as JsPluginConfigItem).description }}</el-text></div>
                      <template v-if="(c as unknown as JsPluginConfigItem).value !== (c as unknown as JsPluginConfigItem).defaultValue">
                        <el-tooltip content="重置为初始值" placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doResetJsConfig((config as unknown as JsPluginConfig)['pluginName'],(c as unknown as JsPluginConfigItem).key)">
                            <brush-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <template v-if="(c as unknown as JsPluginConfigItem).deprecated">
                        <el-tooltip content="移除 - 这个配置在新版的默认配置中不被使用，<br />但升级而来时仍可能被使用，请确认无用后删除" raw-content
                                    placement="bottom-end">
                          <el-icon style="float: right; margin-left: 1rem;" @click="doDeleteUnusedConfig((config as unknown as JsPluginConfig)['pluginName'], (c as unknown as JsPluginConfigItem).key)">
                            <delete-filled />
                          </el-icon>
                        </el-tooltip>
                      </template>
                      <div style="width: 100%; margin-bottom: .5rem;">
                        <el-select v-model="(c as unknown as JsPluginConfigItem).value" @change="doJsConfigChanged()">
                          <el-option v-for="s in (c as unknown as JsPluginConfigItem).option" :key="s" :value="s">{{s}}</el-option>
                        </el-select>
                      </div>
                    </el-form-item>
                  </el-form>
                </el-card>
              </el-collapse-item>
            </el-collapse>
          </main>
        </el-tab-pane>
      </el-tabs>
    </el-col>

  </el-row>
</template>

<script lang="ts" setup>
import {onBeforeUnmount, onMounted, ref, watch} from 'vue';
import {useStore} from '~/store'
import {ElMessage, ElMessageBox} from 'element-plus'
import {
  BrushFilled,
  CaretRight,
  CircleClose,
  CirclePlusFilled,
  Delete,
  DeleteFilled,
  DocumentChecked,
  Download,
  Refresh,
  Search,
  Upload
} from '@element-plus/icons-vue'
import * as dayjs from 'dayjs'
import {basicSetup, EditorView} from "codemirror"
import {javascript} from "@codemirror/lang-javascript"
import DiffViewer from "~/components/mod/diff-viewer.vue";
import {isEqual, size} from "lodash-es";

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

let jsConfigEdited = ref(false)
const doJsConfigChanged = () => {
  jsConfigEdited.value = true
}

const doDeleteUnusedConfig = (pluginName: any, key: any) => {
  ElMessageBox.confirm(
    `删除插件 ${pluginName} 的配置项 ${key} ，确定吗？`,
    '删除',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async (data) => {
    await store.jsDeleteUnusedConfig(pluginName, key )
    setTimeout(() => {
      // 稍等等再重载，以免出现没删掉
      refreshConfig()
    }, 1000);
    ElMessage.success('配置项已删除')
  })
}

const doResetJsConfig = (plginName: string, key: string) => {
  ElMessageBox.confirm(
      '重置这条配置项回默认状态，确定吗？',
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
  ).then(async () => {
    await store.jsResetConfig(plginName, key)
    ElMessage({
      type: 'success',
      message: '成功!',
    })
    setTimeout(() => {
      refreshConfig()
    }, 1000);
  })
}
const doJsConfigAddItem = (arr: any[]) => {
  arr.push("");
  doJsConfigChanged()
  return arr;
}
const doJsConfigRemoveItemAt = <T>(arr: T[], index: number) => {
  if (index < 0 || index >= arr.length) {
    return arr;
  }
  arr.splice(index, 1);
  doJsConfigChanged()
  return arr;
}

const doJsConfigSave = async () => {
  await store.jsSetConfig(jsConfig.value)
    jsConfigEdited.value = false
    ElMessage.success('已保存')
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
  await refreshConfig();

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
const jsConfig = ref<Map<string, JsPluginConfig>>(new Map<string, JsPluginConfig>());
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

const refreshConfig = async () => {
  jsConfig.value = await store.jsGetConfig();
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

const doDelete = async (data: JsScriptInfo, index: number) => {
  ElMessageBox.confirm(
    data.official ? `卸载官方插件《${data.name}》的更新，确定吗？` : `删除插件《${data.name}》，确定吗？`,
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

.js-item {
  min-width: 100%;
}
</style>

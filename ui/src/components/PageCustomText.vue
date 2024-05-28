<template>
  <el-affix :offset="70" v-if="modified">
    <div class="page-header tip-danger">
      <el-text type="danger" size="large" tag="strong">内容已修改，不要忘记保存！</el-text>
      <el-button class="button" type="primary" :icon="DocumentChecked" @click="save"
        :disabled="!modified">点我保存</el-button>
    </div>
  </el-affix>

  <div class="tip">
    <el-collapse class="helptips">
      <el-collapse-item name="1">
        <template #title>
          <el-text tag="strong">查看帮助</el-text>
        </template>

        <el-text tag="p">
          <div>此处可以对骰子返回的文本进行修改。最终返回的文本将为多个条目中随机抽取的一个。</div>
          <div>随机文本:默认一种显示结果，如果需要多种反馈结果，使用＋添加条目，使用-删除条目</div>
          <!-- 权重选择:默认1，权重—致则没有优先级。数字越小，优先级越高 -->
          <!-- <div>文件备份:已修改的指令统一存在于路径/路径1/路径2/文件名，如有需要替换文件即可</div> -->
          <div>遇到有此标记(<el-icon><brush-filled /></el-icon>)的条目，说明和默认值不同，是一个自定义条目</div>
          <div style="margin-top: 1rem;">文本下方的 <el-tag size="small">标签</el-tag> 代表了被默认文本所使用的特殊变量，你可以使用 {变量名} 来插入他们，例如
            {$t判定值} </div>
          <div>除此之外，这些变量可以在所有文本中使用：
            <el-space size="small" wrap>
              <el-tag size="small" disable-transitions
                v-for="i in ['$t玩家', '$tQQ昵称', '$t个人骰子面数', '$tQQ', '$t骰子帐号', '$t骰子昵称', '$t群号', '$t群名']">{{ i }}</el-tag>
            </el-space>
          </div>
          <div>
            <span>以及，所有的自定义文本都可以嵌套使用，例如：</span>
            <div>
              <b>这里是{核心:骰子名字}，我是一个示例</b>
            </div>
            <div>默认会被解析为：</div>
            <div>
              <b>这里是海豹bot，我是一个示例</b>
            </div>
            <div>注意！千万不要递归嵌套，会发生很糟糕的事情。</div>
          </div>

          <div style="margin-top: 1rem;">
            <div>此外，支持插入图片，将图片放在骰子的适当目录，再写这样一句话即可:</div>
            <div><b>[图:data/images/sealdice.png]</b></div>
            <div>可以参考 核心:骰子进群 词条</div>
            <div>同样的，可以使用CQ码插入图片和其他内容，关于CQ码，请参阅onebot项目文档</div>
          </div>

          <div style="margin-top: 1rem;">
            <b>COC的“判定-常规”和“判定-简短”主要区别是，多重检定会默认使用简短版本(.ra 3#射击)</b>
            <b>进行调整后，可以在左侧面板“指令测试”中进行测试！</b>
          </div>
        </el-text>
      </el-collapse-item>
    </el-collapse>
  </div>

  <div style="margin-top: 1rem; display: flex; justify-content: space-between; align-items: center;">
    <div>
      <el-text>搜索：</el-text>
      <el-input :prefix-icon="Search" style="display: inline;" v-model="currentFilterName" clearable></el-input>
    </div>
    <!-- 这个按钮颜色还是淡一些不然喧宾夺主 -->
    <!-- 偷偷加个淡点的颜色回来（） -->
    <el-button type="primary" plain @click="dialogImportVisible = true">导入/导出</el-button>
  </div>

  <el-space style="margin: 1rem 0" wrap>
    <el-radio-group v-model="filterMode" @change="handleFilterModeChange">
      <el-radio v-for="mode of filterModes" :key="mode.value" :label="mode.value">
        <template #default>{{ mode.desc }}</template>
      </el-radio>
    </el-radio-group>
    <div v-if="filterMode === 'group'">
      <el-text>分组：</el-text>
      <el-select v-model="currentFilterGroup" filterable allow-create>
        <el-option v-for="group of filterGroups" :key="group" :label="group" :value="group"></el-option>
      </el-select>
    </div>
  </el-space>

  <el-collapse class="text-collapse" v-model="activeGroups">
    <custom-text-box v-for="[group, values] in reactive(doSort(category))" :group="group">
      <template #values>
        <el-col :xs="24" :span="12" v-for="[k, v] in values">
          <el-form ref="form" label-width="auto" label-position="top">
            <el-form-item>
              <template #label>
                <div>
                  <el-tag effect="dark" type="info" style="margin-right: .5rem;" disable-transitions>
                    {{ (store.curDice.customTextsHelpInfo[category][k.toString()]).subType ||
                  ((store.curDice.customTextsHelpInfo[category][k.toString()]).notBuiltin ? '旧版文本' : '其它') }}
                  </el-tag>

                  <span>
            <span>{{ k.toString() }}</span>
            <el-tooltip v-if="store.curDice.customTextsHelpInfo[category][k.toString()].extraText"
                        :content="store.curDice.customTextsHelpInfo[category][k.toString()].extraText" raw-content>
              <el-icon><question-filled /></el-icon>
            </el-tooltip>
          </span>

                  <template v-if="(store.curDice.customTextsHelpInfo[category][k.toString()]).notBuiltin">
                    <el-tooltip content="移除 - 这个文本在新版的默认配置中不被使用，<br />但升级而来时仍可能被使用，请确认无用后删除" raw-content
                                placement="bottom-end">
                      <el-icon style="float: right; margin-left: 1rem;" @click="askDeleteValue(category, k.toString())">
                        <delete-filled />
                      </el-icon>
                    </el-tooltip>
                  </template>

                  <template v-if="store.curDice.customTextsHelpInfo[category][k.toString()].modified">
                    <el-tooltip content="重置为初始值" placement="bottom-end">
                      <el-icon style="float: right; margin-left: 1rem;" @click="askResetValue(category, k.toString())">
                        <brush-filled />
                      </el-icon>
                    </el-tooltip>
                  </template>
                  <!-- <el-tooltip content="效果预览" placement="bottom-end">
                    <el-icon @click="askResetValue(category, k.toString())" style="float: right;"><video-play /></el-icon>
                  </el-tooltip> -->
                </div>
              </template>

              <div v-for="(k2, index) in v" style="width: 100%; margin-bottom: .5rem;">
                <!-- 这里面是单条修改项 -->
                <el-row>
                  <el-col :span="2">
                    <el-tooltip :content="index === 0 ? '点击添加一个回复语，SealDice将会随机抽取一个回复' : '点击删除你不想要的回复语'"
                                placement="bottom-start">
                      <el-icon>
                        <circle-plus-filled v-if="index == 0" @click="addItem(k)" />
                        <circle-close v-else @click="removeItem(v, index)" />
                      </el-icon>
                    </el-tooltip>
                  </el-col>
                  <el-col :span="22">
                    <!-- :suffix-icon="Management" -->
                    <el-input type="textarea" autosize v-model="k2[0]" @change="doChanged(category, k.toString())"></el-input>
                  </el-col>
                </el-row>
              </div>
              <el-space size="small" wrap>
                <el-tag size="small" disable-transitions
                        v-for="i in store.curDice.customTextsHelpInfo[category][k.toString()].vars">{{ i }}</el-tag>
                <!-- {{ store.curDice.customTextsHelpInfo[category][k.toString()] }} -->
              </el-space>
            </el-form-item>
          </el-form>
        </el-col>
      </template>
    </custom-text-box>
  </el-collapse>

  <el-dialog v-model="dialogImportVisible" title="导入导出" :close-on-click-modal="false" :close-on-press-escape="false"
    :show-close="true" :fullscreen="true" class="the-dialog">
    <el-checkbox v-model="importOnlyCurrent">仅当前页面(勾选)/全部自定义文案</el-checkbox>
    <el-checkbox v-model="importImpact">紧凑</el-checkbox>

    <!-- <template > -->
    <div>以下为导出内容，可以复制给别人:</div>
    <el-input placeholder="填入数据" type="textarea" :autosize="{ minRows: 4 }" class="import-edit" id="import-edit"
      v-model="configForImport"></el-input>
    <!-- </template> -->

    <template #footer>
      <span class="dialog-footer">
        <!-- <el-button @click="dialogImportVisible = false">取消</el-button> -->
        <el-button @click="configForImport = ''">清空</el-button>
        <el-button data-clipboard-target="#import-edit" @click="copied" id="btnCopy1">复制</el-button>
        <el-button type="primary" @click="doImport" :disabled="configForImport === ''">导入并保存</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import {reactive} from "vue";
import {useStore} from '~/store'
import {
  BrushFilled,
  CircleClose,
  CirclePlusFilled,
  DeleteFilled,
  DocumentChecked,
  QuestionFilled,
  Search
} from '@element-plus/icons-vue'
import { cloneDeep, filter, groupBy, map, sortBy, startsWith, trim, uniq, entries, mapValues } from 'lodash-es'
import ClipboardJS from 'clipboard'

const store = useStore()
const props = defineProps<{ category: string }>();

const configForImport = ref('')
const importOnlyCurrent = ref(true)
const importImpact = ref(true)
const dialogImportVisible = ref(false)

const activeGroups = ref(['__others__'])
const doSort = (category: string) => {
  let items = Object.entries(store.curDice.customTexts[category]);
  const helpInfo = store.curDice.customTextsHelpInfo[category];

  if (currentFilterName.value != "") {
    items = items.filter(item => item[0].includes(currentFilterName.value)
      || helpInfo[item[0]].subType.includes(currentFilterName.value));
  }


  switch (filterMode.value) {
    case 'all':
      break
    case 'unmodified':
      items = items.filter(item => !helpInfo[item[0]].modified)
      break
    case 'modified':
      items = items.filter(item => helpInfo[item[0]].modified)
      break
    case 'deprecated':
      items = items.filter(item => helpInfo[item[0]].notBuiltin)
      break
    case 'group':
      filterGroups.value = sortBy(uniq(Object.values(helpInfo).map(info => trim(info.subType)).filter(subType => subType !== "")))
      items = items.filter(item => startsWith(trim(helpInfo[item[0]].subType), currentFilterGroup.value))
      break
  }

  // 子类别元素超过 4 个的，展示上需要打包
  const boxedGroups = map(filter(entries(groupBy(map(items, item => {
    const subType = helpInfo[item[0]].subType;
    return subType.split(' ')[0];
  }), group => group)), group => group[1].length >= 4), group => group[0]);

  const outMap = entries(mapValues(groupBy(items, item => {
    const group = helpInfo[item[0]].subType.split(' ')[0];
    if (boxedGroups.includes(group)) {
      return group
    } else {
      return '__others__'
    }
  }), items => items.sort((a, b) => {
    const ia = helpInfo[a[0]];
    const ib = helpInfo[b[0]];

    if (ia.topOrder !== ib.topOrder) {
      return ib.topOrder - ia.topOrder;
    }

    if (ia.subType !== ib.subType) {
      return ib.subType.localeCompare(ia.subType);
    }

    return 0;
  }))).sort( (a, b) => {
    const [aGroup, _] = a
    const [bGroup, _2] = b
    if (aGroup === '__others__') {
      return -1
    } else if (bGroup === '__others__') {
      return 1
    } else {
      return 0
    }
  })

  return outMap;
}

const copied = () => {
  ElMessage.success('进行了复制！')
}

const importRefresh = () => {
  const indent = !importImpact.value ? 2 : 0
  if (importOnlyCurrent.value) {
    configForImport.value = JSON.stringify({
      title: '某人的自定义配置',
      items: {
        [props.category]: store.curDice.customTexts[props.category]
      }
    }, null, indent)
  } else {
    configForImport.value = JSON.stringify(store.curDice.customTexts, null, indent)
  }
}

const doImport = async () => {
  try {
    const data = JSON.parse(configForImport.value)
    if (!(data.title && data.items)) {
      ElMessage.error('格式不正确')
      return
    }

    for (let [k, v] of Object.entries(data.items)) {
      if (store.curDice.customTexts[k]) {
        store.curDice.customTexts[k] = v as any
      }
      await store.customTextSave(k)
    }
    await store.getCustomText()
    modified.value = false
    ElMessage.success('已保存')
    dialogImportVisible.value = false
  } catch (e) {
    ElMessage.error('格式不正确')
  }
}

watch(() => dialogImportVisible.value, (newValue, oldValue) => {
  if (newValue) {
    importRefresh()
    nextTick(() => {
      new ClipboardJS(document.getElementById('btnCopy1') as any)
    })
  }
})

watch(() => [importImpact.value, importOnlyCurrent.value], (newValue) => {
  importRefresh()
})


watch(() => props.category, (newValue, oldValue) => { //直接监听
  modified.value = false
});

const addItem = (k: any) => {
  store.curDice.customTexts[props.category][k].push(['', 1 as any])
  modified.value = true
}

const doChanged = (category: string, keyName: string) => {
  modified.value = true
  const helpInfo = store.curDice.customTextsHelpInfo[category][keyName]
  helpInfo.modified = true
}

const removeItem = (v: any[], index: number) => {
  v.splice(index, 1)
  modified.value = true
}

const save = async () => {
  modified.value = false
  await store.customTextSave(props.category)
  await store.getCustomText()
  modified.value = false
  ElMessage.success('已保存')
}

const deleteValue = async (category: string, keyName: string) => {
  const helpInfo = store.curDice.customTextsHelpInfo[category][keyName]
  delete store.curDice.customTexts[category][keyName]
  modified.value = true
}


const askDeleteValue = async (category: string, keyName: string) => {
  ElMessageBox.confirm(
    '删除这条文本，确定吗？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    deleteValue(category, keyName)
    ElMessage({
      type: 'success',
      message: '成功!',
    })
  })
}

const resetValue = async (category: string, keyName: string) => {
  const helpInfo = store.curDice.customTextsHelpInfo[category][keyName]
  store.curDice.customTexts[category][keyName] = cloneDeep(helpInfo.origin)
  helpInfo.modified = false
  modified.value = true
}

const askResetValue = async (category: string, keyName: string) => {
  ElMessageBox.confirm(
    '重置这条文本回默认状态，确定吗？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    resetValue(category, keyName)
    ElMessage({
      type: 'success',
      message: '成功!',
    })
  })
}

const modified = ref(false)

interface FilterMode {
  value: string,
  desc: string,
}

const filterModes: FilterMode[] = [
  { value: "all", desc: "全部" },
  { value: "unmodified", desc: "默认文案" },
  { value: "modified", desc: "修改过" },
  { value: "group", desc: "指定分组" },
  { value: "deprecated", desc: "旧版文本" },
]
const filterMode = ref<string>("all")
const filterGroups = ref<string[]>([])
const currentFilterGroup = ref<string>("")
const currentFilterName = ref<string>("")

const handleFilterModeChange = (newMode: any) => {
  if (newMode === 'group') {
    nextTick(() => {
      currentFilterGroup.value = filterGroups.value[0]
      currentFilterName.value = ''
    })
  } else {
    currentFilterGroup.value = ''
    currentFilterName.value = ''
  }
}

onBeforeMount(async () => {
  modified.value = false
})

watch(props, () => {
  filterMode.value = 'all'
})
</script>

<style scoped>
.import-edit>textarea {
  max-height: 65vh;
}

.helptips {
  background-color: #f3f5f7;
}

.helptips :deep(.el-collapse-item__header) {
  background-color: #f3f5f7;
}

.helptips :deep(.el-collapse-item__wrap) {
  background-color: #f3f5f7;
}

.text-collapse {
  width: 100%;
  background-color: #f3f5f7;

  :deep(.el-collapse-item__header) {
    background-color: #f3f5f7;
  }

  :deep(.el-collapse-item__wrap) {
    background-color: #f3f5f7;
  }
}
</style>

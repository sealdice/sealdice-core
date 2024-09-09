<template>
  <header class="page-header">
    <el-switch @change="switchClick" v-model="replyEnable" active-text="启用" inactive-text="关闭">总开关</el-switch>
    <el-button @click="doSave" :icon="DocumentChecked" type="primary"
      v-if="store.curDice.config.customReplyConfigEnable">保存</el-button>
  </header>

  <el-affix :offset="70" v-if="modified">
    <div class="tip-danger">
      <el-text type="danger" size="large" tag="strong">内容已修改，不要忘记保存！</el-text>
    </div>
  </el-affix>

  <div class="tip">
    <el-collapse class="helptips" v-model="activeTip">
      <el-collapse-item name="basic">
        <template #title>
          <el-text tag="strong">基础帮助</el-text>
        </template>
        <el-text tag="p">每项自定义回复由一个&lt;条件>触发，产生一系列&lt;结果><br />一旦一个&lt;条件>被满足，将立即停止匹配，并执行&lt;结果></el-text>
      </el-collapse-item>
      <el-collapse-item name="advanced">
        <template #title>
          <el-text tag="strong">进阶内容</el-text>
        </template>
        <el-text tag="p">
          越靠前的项具有越高的优先级，可以拖动来调整优先顺序！<br />
          为了避免滥用和无限互答，自定义回复的响应间隔最低为5s<br />
          匹配到的文本将被存入变量$t0，正则匹配的组将被存入$t1 $t2 ....<br />
          若存在组名，如(?P&lt;A&gt;cc)，将额外存入$tA
        </el-text>
      </el-collapse-item>
    </el-collapse>
  </div>

  <el-divider />

  <main class="reply-main">
    <header style="display: flex; flex-wrap: wrap; justify-content: space-between;"
      v-if="store.curDice.config.customReplyConfigEnable">
      <el-space class="current-reply" direction="vertical">
        <el-space style="justify-content: center" wrap>
          <strong>当前文件</strong>
          <el-select v-model="curFilename" style="width: 10rem">
            <el-option v-for="item in fileItems" :key="item.filename" :label="item.filename" :value="item.filename" />
          </el-select>
          <el-checkbox-button :class="cr.enable ? `reply-file-status-open` : `reply-file-status-close`"
            style="margin-left: 1rem" v-model="cr.enable">
            {{ cr.enable ? '已启用' : '未启用' }}
          </el-checkbox-button>
        </el-space>
        <el-space style="margin-top: 0.5rem;" warp>
          <el-button @click="customReplyFileDelete" type="danger" size="small" plain :icon="Delete">删除</el-button>
          <el-button type="primary" size="small" plain :icon="Download" tag="a" style="text-decoration: none;"
            :href="`${urlBase}/sd-api/configs/custom_reply/file_download?name=${encodeURIComponent(curFilename)}&token=${encodeURIComponent(store.token)}`">下载
          </el-button>
        </el-space>
        <el-text class="mt-2" v-if="!cr.enable" type="warning">注意：启用后该文件中的自定义回复才会生效</el-text>
      </el-space>
      <div class="mt-4 sm:mt-0 reply-operation">
        <div>
          <el-tooltip content="新建一个自定义回复文件。">
            <el-button type="success" plain :icon="DocumentAdd" @click="customReplyFileNew">新建</el-button>
          </el-tooltip>
        </div>
        <div>
          <el-tooltip content="通过粘贴/编辑文本来导入自定义回复。">
            <el-button type="primary" plain :icon="Tickets" @click="dialogFormVisible = true">解析</el-button>
          </el-tooltip>
        </div>
        <el-tooltip content="上传自定义回复的 .yaml 文件。">
          <el-upload action="" multiple accept=".yaml" :before-upload="beforeUpload" :file-list="uploadFileList">
            <el-button type="primary" plain :icon="Upload">上传</el-button>
          </el-upload>
        </el-tooltip>
      </div>
    </header>

    <el-divider v-if="store.curDice.config.customReplyConfigEnable" />

    <template v-if="!store.curDice.config.customReplyConfigEnable">
      <div></div>
      <el-text type="danger" size="large" style="font-size: 1.5rem;">请先启用总开关！</el-text>
    </template>
    <template v-else>
      <foldable-card ref="commonConditionsRef" type="div" :default-fold="true"
                     :class="cr.enable ? '' : 'disabled'">
        <template #title>
          <el-space size="large" wrap>
            <el-space size="small">
              <strong>公共条件</strong>
              <el-button type="success" size="small" plain :icon="Plus" @click="addOneCondition(conditions)">添加一项</el-button>
            </el-space>
            <el-text type="info" size="small">该文件下所有的回复的执行，都需要先满足以下公共条件（需同时满足，即and）。</el-text>
          </el-space>
        </template>

        <template v-if="conditions && conditions.length > 0">
          <custom-reply-conditions v-model="conditions"/>
        </template>
        <template v-else>
          <el-text type="info">当前无公共条件</el-text>
        </template>

        <template #unfolded-extra>
          <template v-if="conditions && conditions.length > 0">
            <el-text type="info">公共条件数量：{{ conditions.length }}</el-text>
          </template>
          <template v-else>
            <el-text type="info">当前无公共条件</el-text>
          </template>
        </template>
      </foldable-card>

      <el-divider/>

      <nested-draggable :tasks="list" :class="cr.enable ? '' : 'disabled'" />
      <div style="display: flex; justify-content: space-between;">
        <el-button type="success" plain :icon="Plus" @click="addOne(list)">添加一项</el-button>
        <el-button @click="doSave" :icon="DocumentChecked" type="primary">保存</el-button>
      </div>
    </template>
  </main>

  <el-dialog v-model="dialogFormVisible" title="导入配置" :close-on-click-modal="false" :close-on-press-escape="false"
    :show-close="false" class="the-dialog">
    <!-- <template > -->
    <el-input placeholder="支持格式: 关键字/回复语" class="reply-text" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }"
      v-model="configForImport"></el-input>
    <!-- </template> -->

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogFormVisible = false">取消</el-button>
        <el-button type="primary" @click="doImport" :disabled="configForImport === ''">下一步</el-button>
      </span>
    </template>
  </el-dialog>

  <el-dialog v-model="dialogLicenseVisible" title="许可协议" :close-on-click-modal="false" :close-on-press-escape="false"
    :show-close="false" class="the-dialog">
    <pre style="white-space: pre-wrap;">尊敬的用户，欢迎您选择由木落等研发的海豹骰点核心（SealDice），在您使用自定义功能前，请务必仔细阅读使用须知，当您使用我们提供的服务时，即代表您已同意使用须知的内容。

  您需了解，海豹核心官方版只支持TRPG功能，娱乐功能定制化请自便，和海豹无关。
  您清楚并明白您对通过骰子提供的全部内容负责，包括自定义回复、非自带的插件、牌堆。海豹骰不对非自身提供以外的内容合法性负责。您不得在使用海豹骰服务时，导入包括但不限于以下情形的内容:
  (1) 反对中华人民共和国宪法所确定的基本原则的；
  (2) 危害国家安全，泄露国家秘密，颠覆国家政权，破坏国家统一的;
  (3) 损害国家荣誉和利益的;
  (4) 煽动民族仇恨、民族歧视、破坏民族团结的；
  (5) 破坏国家宗教政策,宣扬邪教和封建迷信的;
  (6) 散布谣言，扰乱社会秩序，破坏社会稳定的；
  (7) 散布淫秽、色情、赌博、暴力、凶杀、恐怖或者教唆犯罪的；
  (8) 侮辱或者诽谤他人，侵害他人合法权益的；
  (9) 宣扬、教唆使用外挂、私服、病毒、恶意代码、木马及其相关内容的；
  (10)侵犯他人知识产权或涉及第三方商业秘密及其他专有权利的；
  (11)散布任何贬损、诋毁、恶意攻击海豹骰及开发人员、海洋馆工作人员、mod编写人员、关联合作者的；
  (12)含有中华人民共和国法律、行政法规、政策、上级主管部门下发通知中所禁止的其他内容的。

  一旦查实您有以上禁止行为，请立即停用海豹骰。同时我们也会主动对你进行举报。</pre>
    <!-- 一旦查实您有以上禁止行为，我们有权进行核查、修改和/或删除您导入的内容，而不需事先通知。 -->

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogLicenseVisible = false">我同意</el-button>
        <el-button @click="licenseRefuse">我拒绝</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { urlBase } from "~/backend";
import { useStore } from '~/store';
import nestedDraggable from "../utils/nested.vue";
import {
  DocumentChecked,
  Delete,
  Download,
  DocumentAdd,
  Tickets,
  Upload,
  Plus
} from '@element-plus/icons-vue'

const store = useStore()
const dialogFormVisible = ref(false)
const dialogLicenseVisible = ref(false)
const configForImport = ref('')

const replyEnable = computed({
  get: () => store.curDice.config.customReplyConfigEnable,
  set: (value) => {
    store.diceConfigSet({ customReplyConfigEnable: value })
    if (!store.curDice.config.customReplyConfigEnable) {
      dialogLicenseVisible.value = true
    }
  }
})

watch(replyEnable, async (newStatus, oldStatus) => {
  if (newStatus != oldStatus) {
  }
})

const activeTip = ref('basic')

const curFilename = ref('reply.yaml')

const conditions = ref<any>([])

const commonConditionsRef = ref<any>(null)

const list = ref<any>([
  // {"enable":true,"condition":{"condType":"match","matchType":"match_exact","value":"asd"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
  // {"enable":true,"condition":{"condType":"match","matchType":"match_fuzzy","value":"ccc"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
])

const fileItems = ref<any>([
  // {"enable":true,"condition":{"condType":"match","matchType":"match_exact","value":"asd"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
  // {"enable":true,"condition":{"condType":"match","matchType":"match_fuzzy","value":"ccc"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
])

const uploadFileList = ref<any[]>([])

const cr = ref<any>({ enable: true })

const switchClick = () => {
  if (!store.curDice.config.customReplyConfigEnable) {
    dialogLicenseVisible.value = true
  }

  store.diceConfigSet({ customReplyConfigEnable: !store.curDice.config.customReplyConfigEnable })
}

const licenseRefuse = () => {
  dialogLicenseVisible.value = false
  store.curDice.config.customReplyConfigEnable = false
  store.diceConfigSet({ customReplyConfigEnable: false })
}

const modified = ref(false)

watch(() => cr.value, (newValue, oldValue) => { //直接监听
  modified.value = true
}, { deep: true });

watch(() => curFilename.value, (newValue, oldValue) => { //直接监听
  nextTick(() => {
    refreshCurrent();
  });
});

const customReplyFileNew = () => {
  ElMessageBox.prompt(
    '创建一个新的回复文件',
    '',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'info',
      inputPlaceholder: 'reply2.yaml',
      inputValue: `reply${Math.ceil(Math.random() * 10000)}.yaml`,
    }
  ).then(async (data) => {
    if (!data.value) {
      data.value = `reply${Math.ceil(Math.random() * 10000)}.yaml`;
    }
    const ret = await store.customReplyFileNew(data.value);
    let ret2 = await store.customReplyFileList() as any;
    fileItems.value = ret2.items;
    curFilename.value = ret2.items[0].filename;

    ElMessage({
      type: 'success',
      message: (ret as any).success ? '成功!' : '失败',
    })
  })
}

const customReplyFileDelete = () => {
  ElMessageBox.confirm(
    '是否删除此文件？',
    '',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    const ret = await store.customReplyFileDelete(curFilename.value);

    if ((ret as any).success) {
      let ret = await store.customReplyFileList() as any;
      fileItems.value = ret.items;
      curFilename.value = ret.items[0].filename;
      await refreshCurrent();
    }

    ElMessage({
      type: 'success',
      message: '完成',
    })
  })
}

const beforeUpload = async (file: any) => { // UploadRawFile
  let fd = new FormData()
  fd.append('file', file)
  try {
    const resp = await store.customReplyFileUpload({ form: fd });
    ElMessage.success('上传完成');
    let ret = await store.customReplyFileList() as any;
    fileItems.value = ret.items;
    curFilename.value = ret.items[0].filename;
    await refreshCurrent();
  } catch (e) {
    ElMessage.error('上传失败，可能有同名文件！')
  }
}

const addOneCondition = (lst: any) => {
  lst.push({ "condType": "textMatch", "matchType": "matchExact", "value": "要匹配的文本" })
  commonConditionsRef.value.open()
}

const addOne = (lst: any) => {
  lst.push({ "enable": true, "conditions": [{ "condType": "textMatch", "matchType": "matchExact", "value": "要匹配的文本" }], "results": [{ "resultType": "replyToSender", "delay": 0, "message": [["说点什么", 1]] }] },)
}

const deleteAnyItem = (lst: any[], index: number) => {
  lst.splice(index, 1);
}

const doSave = async () => {
  try {
    for (let i of cr.value.items) {
      for (let j of i.conditions) {
        if (j.condType === 'textLenLimit') {
          j.value = parseInt(j.value) || 0;
        }
      }

      for (let j of i.results) {
        if (j.delay) {
          j.delay = parseFloat(j.delay)
          if (j.delay < 0) {
            j.delay = 0
          }
        }
        if (!j.delay) j.delay = 0
      }
    }
    cr.value.filename = curFilename.value;
    cr.value.conditions = conditions.value;
    for (let cond of cr.value.conditions) {
      if (cond.condType === 'textLenLimit') {
        cond.value = parseInt(cond.value) || 0;
      }
    }
    await store.setCustomReply(cr.value)
    ElMessage.success('已保存')
    modified.value = false
  } catch (e) {
    ElMessage.error('保存失败！！')
  }
}

function parseString(str: string): [string[], string[], string] {
  const leftArr: string[] = [];
  const rightArr: string[] = [];
  let restIndex = 0;

  let currentStr = "";
  let isLeft = true;
  let isEscaped = false;

  for (let i = 0; i < str.length; i++) {
    const char = str[i];
    restIndex = i;
    if (isEscaped) {
      if (char !== "\r" && char !== "\n" && char !== "/") {
        currentStr += "\\"; // 如果是被转义的字符，将反斜杠保留
      }
      if (char === 'n' || char === 'r') {
        currentStr = currentStr.slice(0, -1) + '\n'
      } else {
        currentStr += char;
      }
      isEscaped = false;
      continue;
    }
    if (char == '\n') {
      break;
    }
    if (char === "\\") {
      isEscaped = true;
      continue;
    }
    if (char === "|") {
      if (isLeft) {
        leftArr.push(currentStr);
      } else {
        rightArr.push(currentStr);
      }
      currentStr = "";
    } else if (char === "/") {
      if (i < str.length - 1 && str[i + 1] === "|") {
        currentStr += char;
      } else {
        if (isLeft) {
          leftArr.push(currentStr);
        } else {
          rightArr.push(currentStr);
        }
        currentStr = "";
        isLeft = false;
      }
    } else {
      currentStr += char;
    }
  }
  // 处理最后一个字符串
  if (isLeft) {
    leftArr.push(currentStr);
  } else {
    rightArr.push(currentStr);
  }

  return [leftArr, rightArr, str.slice(restIndex + 1)];
}

const doImport = () => {
  const ret = []
  let count = 0
  let text = configForImport.value

  while (true) {
    const [a, b, rest] = parseString(text);
    if (a.length && b.length) {
      const replies = b.map((v) => [v, 1]);
      list.value.push({
        "enable": true, "conditions": [
          { "condType": "textMatch", "matchType": "matchMulti", "value": a.join('|') }], "results": [{ "resultType": "replyToSender", "delay": 0, "message": replies }]
      });
      count += 1;
    }
    text = rest;
    if (!rest) break;
  }

  ElMessage.success('导入成功!')
  configForImport.value = ''
}

onBeforeMount(async () => {
  let ret = await store.customReplyFileList() as any;
  fileItems.value = ret.items;
  curFilename.value = ret.items[0].filename;
  await store.diceConfigGet();
  await refreshCurrent();
})

const refreshCurrent = async () => {
  console.log('load', curFilename.value);
  const ret = await store.getCustomReply(curFilename.value) as any
  cr.value = ret
  conditions.value = ret.conditions
  list.value = ret.items
  await nextTick(() => {
    modified.value = false
  })
}

onBeforeUnmount(() => {
  // clearInterval(timerId)
})
</script>

<style scoped>
.reply-text>textarea {
  max-height: 65vh;
}

.helptips {
  background-color: #f3f5f7;

  :deep(.el-collapse-item__header) {
    background-color: #f3f5f7;
  }

  :deep(.el-collapse-item__wrap) {
    background-color: #f3f5f7;
  }
}
</style>

<style scoped lang="css">
@media screen and (max-width: 700px) {
  .bak-item {
    flex-direction: column;

    &>span {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
  }

  .current-reply {
    width: 100%;
  }

  .reply-operation {
    width: 100%;
    justify-content: space-around;
  }

  .reply-operation div:not(:last-child) {
    margin-right: 1rem;
  }
}

@media screen and (min-width: 700px) {
  .reply-operation {
    flex-direction: column;
    margin-right: 0.5rem;
  }

  .reply-operation div:not(:last-child) {
    margin-bottom: 0.5rem;
  }
}

.reply-operation {
  display: flex;
  justify-content: center;
  align-items: self-start;
}

.disabled {
  filter: grayscale(1);
  /* pointer-events: none; */
  cursor: not-allowed;
  user-select: none;
}

.img-box {
  height: 250px;
  margin-right: 3rem;
  float: left;

  img {
    height: 200px;
  }
}

.about {
  background-color: #fff;
  padding: 2rem;
  line-height: 2rem;
  text-align: left;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04)
}

.subtitle {
  margin-bottom: 1rem;
  font-weight: bold;
}
</style>
<template>
  <el-affix :offset="60" v-if="modified">
    <div class="tip">
      <!-- <p class="title">TIP</p> -->
      <div style="display: flex; justify-content: space-between; align-content: center; align-items: center">
        <span>内容已修改，不要忘记保存</span>
        <!-- <el-button class="button" type="primary" @click="save" :disabled="!modified">点我保存</el-button> -->
      </div>
    </div>
  </el-affix>

  <div>
    <el-checkbox @click="switchClick" v-model="cr.enable">总开关</el-checkbox>
    <el-button style="float: right" @click="dialogFormVisible = true">导入</el-button>
    <div>每项自定义回复由一个“条件”触发，产生一系列“结果”</div>
    <div>一旦一个“条件”被满足，将立即停止匹配，并执行“结果”</div>
    <div>越靠前的项具有越高的优先级，可以拖动来调整优先顺序！</div>
    <div>为了避免滥用和无限互答，自定义回复的响应间隔最低为5s</div>
    <div>匹配到的文本将被存入变量$t0，正则匹配的组将被存入$t1 $t2 ....</div>
    <div>若存在组名，如(?P&lt;A&gt;cc)，将额外存入$tA</div>
    <nested-draggable :tasks="list" />
    <div>
      <el-button @click="addOne(list)">添加一项</el-button>
      <el-button @click="doSave">保存</el-button>
    </div>
  </div>

  <el-dialog v-model="dialogFormVisible" title="导入配置" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false" custom-class="the-dialog">
    <!-- <template > -->
    <el-input placeholder="支持格式: 关键字/回复语" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" v-model="configForImport"></el-input>
    <!-- </template> -->

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogFormVisible = false">取消</el-button>
        <el-button type="primary" @click="doImport" :disabled="configForImport === ''">下一步</el-button>
      </span>
    </template>
  </el-dialog>

  <el-dialog v-model="dialogLicenseVisible" title="许可协议" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false" custom-class="the-dialog">
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
import { computed, onBeforeMount, onBeforeUnmount, ref, watch, nextTick } from 'vue';
import { useStore } from '~/store';
import imgHaibao from '~/assets/haibao1.png'
import nestedDraggable from "./nested.vue";
import { ElMessage, ElMessageBox } from 'element-plus'
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
import { cloneDeep } from "lodash-es";

const store = useStore()
const dialogFormVisible = ref(false)
const dialogLicenseVisible = ref(false)
const configForImport = ref('')

const list = ref<any>([
  // {"enable":true,"condition":{"condType":"match","matchType":"match_exact","value":"asd"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
  // {"enable":true,"condition":{"condType":"match","matchType":"match_fuzzy","value":"ccc"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
])

const cr = ref<any>({})

const switchClick = () => {
  if (!cr.value.enable) {
    dialogLicenseVisible.value = true
  }
}

const licenseRefuse = () => {
  dialogLicenseVisible.value = false
  cr.value.enable = false
}

const modified = ref(false)

watch(() => cr.value, (newValue, oldValue) => { //直接监听
  modified.value = true
}, {deep: true});

const addOne = (lst: any) => {
  lst.push({"enable":true,"conditions":[{"condType":"textMatch","matchType":"matchExact","value":"要匹配的文本"}],"results":[{"resultType":"replyToSender","delay":0,"message":[ ["说点什么", 1] ]}]},)
}

const doSave = async () => {
  try {
    for (let i of cr.value.items) {
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
    await store.setCustomReply(cr.value)
    ElMessage.success('已保存')
    modified.value = false
  } catch (e) {
    ElMessage.error('保存失败！！')
  }
}

const doImport = () => {
  const ret = []
  let count = 0
  const text = configForImport.value

  for (let i of text.matchAll(/^([^/\n]+)\//gm)) {
    ret.push([i[1], i.index, 0])
    if (count !== 0) {
      ret[count-1][2] = i.index
    }
    count += 1
  }

  if (ret.length) {
    ret[ret.length-1][2] = text.length
  }

  for (let i of ret) {
    const word = i[0] as string
    const reply = text.slice((i[1] as number) + word.length + 1, i[2] as number)
    list.value.push({"enable":true,"conditions":[{"condType":"textMatch","matchType":"matchExact","value":word}],"results":[{"resultType":"replyToSender","delay":0,"message": [ [reply, 1] ]}]},)
  }

  ElMessage.success('导入成功!')
  configForImport.value = ''
}

onBeforeMount(async () => {
  const ret = await store.getCustomReply() as any
  cr.value = ret
  list.value = ret.items
  nextTick(() => {
    modified.value = false
  })
})

onBeforeUnmount(() => {
  // clearInterval(timerId)
})
</script>

<style scoped lang="scss">
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

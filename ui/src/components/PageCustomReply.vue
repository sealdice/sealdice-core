<template>
  <div>
    <el-checkbox v-model="cr.enable">总开关</el-checkbox>
    <el-button style="float: right" @click="dialogFormVisible = true">导入</el-button>
    <div>每项自定义回复由一个“条件”触发，产生一系列“结果”</div>
    <div>一旦一个“条件”被满足，将立即停止匹配，并执行“结果”</div>
    <div>越靠前的项具有越高的优先级，可以拖动来调整优先顺序！</div>
    <div>为了避免滥用和无限互答，自定义回复的响应间隔最低为5s，可调整(尚未实装)</div>
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
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount, ref } from 'vue';
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
const configForImport = ref('')

const list = ref<any>([
  // {"enable":true,"condition":{"condType":"match","matchType":"match_exact","value":"asd"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
  // {"enable":true,"condition":{"condType":"match","matchType":"match_fuzzy","value":"ccc"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
])

const cr = ref<any>({})

const addOne = (lst: any) => {
  lst.push({"enable":true,"conditions":[{"condType":"textMatch","matchType":"matchExact","value":"要匹配的文本"}],"results":[{"resultType":"replyToSender","delay":0,"message":[ ["说点什么", 1] ]}]},)
}

const doSave = () => {
  store.setCustomReply(cr.value)
  ElMessage.success('已保存')
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

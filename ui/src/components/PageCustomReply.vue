<template>
  <div>
    <el-checkbox v-model="cr.enable">总开关</el-checkbox>
    <nested-draggable :tasks="list" />
    <div>
      <el-button @click="addOne(list)">添加一项</el-button>
      <el-button @click="doSave">保存</el-button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount, ref } from 'vue';
import { useStore } from '~/store';
import imgHaibao from '~/assets/haibao1.png'
import nestedDraggable from "./nested.vue";

const store = useStore()

const list = ref([
  // {"enable":true,"condition":{"condType":"match","matchType":"match_exact","value":"asd"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
  // {"enable":true,"condition":{"condType":"match","matchType":"match_fuzzy","value":"ccc"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]},
])

const cr = ref<any>({})

const addOne = (lst: any) => {
  lst.push({"enable":true,"condition":{"condType":"match","matchType":"match_exact","value":"要匹配的文本"},"results":[{"resultType":"replyToSender","delay":0,"message":"说点什么"}]},)
}

const doSave = () => {
  store.setCustomReply(cr.value)
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

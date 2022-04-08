<template>
  <el-form label-width="120px">
    未完成界面

    <el-form-item label="UI界面地址">
      <el-input v-model="config.serveAddress" />
    </el-form-item>

    <el-form-item label="Master解锁码">
      <div>{{config.masterUnlockCode}}</div>
    </el-form-item>

    <el-form-item label="UI界面密码">
      <el-input type="password" v-model="config.uiPassword" />
    </el-form-item>

    <el-form-item label="QQ回复延迟(秒)">
      <el-input v-model="config.messageDelayRangeStart" type="number" style="width: 6rem;" />
      <span style="margin: 0 1rem">-</span>
      <el-input v-model="config.messageDelayRangeEnd" type="number" style="width: 6rem;" />
    </el-form-item>

    <el-form-item label="指令前缀">
      <div style="width: 100%; margin-bottom: .5rem;">
      </div>
    </el-form-item>

    <el-form-item label="日志仅记录指令">
        <el-checkbox label="群信息" v-model="config.onlyLogCommandInGroup"/>
        <el-checkbox label="私聊" v-model="config.onlyLogCommandInPrivate"/>
    </el-form-item>
  </el-form>

</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount } from 'vue';
import { useStore } from '~/store';
const store = useStore()

onBeforeMount(async () => {
  await store.diceConfigGet()
  // await store.logFetchAndClear()
})

onBeforeUnmount(() => {
  // clearInterval(timerId)
})

const config = computed(() => {
  return store.curDice.config
})

const onSubmit = () => {
  console.log('submit!')
}
</script>

<style scoped lang="scss">
.about {
  background-color: #fff;
  padding: 2rem;
  line-height: 2rem;
  text-align: left;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04)
}

.top-item {
  flex: 1 0 50%;
  flex-grow: 0;
}
</style>

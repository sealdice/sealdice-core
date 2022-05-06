<template>
  <h2>群管理</h2>
  <div>
    <span style="margin-right: 1rem;">平台:</span>
    <el-checkbox v-model="showPlatformQQ">QQ</el-checkbox>
    <el-checkbox v-model="showPlatformQQCH">QQ频道</el-checkbox>
  </div>
  <div>
    <span style="margin-right: 1rem;">其他:</span>
    <el-checkbox v-model="orderByTimeDesc">按最后使用时间降序</el-checkbox>    
  </div>
  <div>
    <span style="margin-right: 1rem;">搜索:</span>
    <el-input v-model="searchBy" style="max-width: 15rem;" placeholder="请输入帐号或群名的一部分"></el-input>
  </div>

  <div style="margin-top: 2rem;">
    <div v-for="i, index in groupItems" style="margin-bottom: 2rem;border: 1px solid #ccc; border-radius: .2rem; padding: .5rem; background-color: #fff;">
      <el-checkbox v-model="i.active" @click="i.changed = true">启用</el-checkbox>
      <div><span class="left">群名:</span> {{ i.groupName || '未获取到' }}</div>
      <div><span class="left">群号:</span> {{i.groupId}}</div>
      <div><span class="left">上次指令:</span> {{ i.recentCommandTime ? dayjs.unix(i.recentCommandTime).fromNow() : '从未' }}</div>
      <!-- <div>玩家数量(执行过指令): {{i.tmpPlayerNum}}</div> -->
      <div><span class="left">Log状态:</span> {{i.logOn ? '开启' : '关闭'}}</div>
      <div><span class="left">邀请人:</span> {{ i.inviteUserId || '未知' }}</div>
      <div><span class="left">入群时间:</span> {{ i.enteredTime ? dayjs.unix(i.enteredTime).fromNow() : '未知' }}</div>
      <div><span class="left">开启迎新:</span> {{i.showGroupWelcome ? '开启' : '关闭'}}</div>
      <div><span class="left">启用扩展:</span> {{i.tmpExtList.join(', ')}}</div>
      <!-- <div>欢迎语: <el-input type="textarea" v-model="i.groupWelcomeMessage" autosize /> </div> -->
      <!-- <div>{{i}}</div> -->
      <el-button :disabled="!i.changed" @click="saveOne(i, index)">保存</el-button>
      <el-button @click="quitGroup(i, index)">退群</el-button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount, onMounted, ref } from 'vue';
import { useStore } from '~/store'
import { urlBase } from '~/backend'
import filesize from 'filesize'
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
import * as dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { cloneDeep, sortBy } from 'lodash-es'

dayjs.extend(relativeTime)

const store = useStore()

const data = ref<{
  items: any[]
}>({
  items: []
})

const groupList = ref<any>({})

const showPlatformQQ = ref(true)
const showPlatformQQCH = ref(false)
const orderByTimeDesc = ref(true)
const searchBy = ref('')

const groupItems = computed<any[]>(() => {
  if (groupList.value.items) {
    // const groupListItems = cloneDeep(groupList.value.items)
    let items = []
    for (let i of groupList.value.items) {
      let ok = false
      if (i.groupId.startsWith('QQ-CH-Group:') && showPlatformQQCH.value) {
        ok = true
      }
      if (i.groupId.startsWith('QQ-Group:') && showPlatformQQ.value) {
        ok = true
      }

      if (ok && searchBy.value !== '') {
        let a = false
        let b = false
        if (i.groupId.indexOf(searchBy.value) !== -1) {
          a = true
        }
        if (i.groupName.indexOf(searchBy.value) !== -1) {
          b = true
        }
        ok = a || b
      }

      if (ok) items.push(i)
    }
    
    items = sortBy(items, ['recentCommandTime'])
    if (orderByTimeDesc.value) {
      items = items.reverse()
    }
    return items
  }
  return []
})

const refreshList = async () => {
  const data = await store.groupList()
  groupList.value = data
}

const saveOne = async (i: any, index: number) => {
  // await store.backupConfigSave(cfg.value)
  // console.log(222, i, index)
  await store.groupSetOne(i)
  i.changed = false
  ElMessage.success('已保存')
}

const quitGroup = async (i: any, index: number) => {
  // await store.backupConfigSave(cfg.value)
  // console.log(222, i, index)
  ElMessage.success('已保存')
}

onBeforeMount(async () => {
  await refreshList()
})
</script>

<style lang="scss">
span.left {
  display: inline-block;
  min-width: 5rem;
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
</style>
<template>
  <h2>群管理</h2>
  <div style="display: flex;">
    <span style="margin-right: 1rem; white-space: nowrap;">平台:</span>
    <span style="margin-right: -.2rem;">
      <el-checkbox-group v-model="checkList">
        <el-checkbox label="QQ-Group:">QQ群</el-checkbox>
        <el-checkbox label="QQ-CH-Group:">QQ频道</el-checkbox>
        <el-checkbox label="DISCORD-CH-Group:">Discord频道</el-checkbox>
        <el-checkbox label="DODO-Group:">Dodo频道</el-checkbox>
        <el-checkbox label="KOOK-CH-Group:">KOOK频道</el-checkbox>
        <el-checkbox label="DINGTALK-Group:">钉钉群</el-checkbox>
        <el-checkbox label="SLACK-CH-Group:">Slack频道</el-checkbox>
        <el-checkbox label="TG-Group:">TG群</el-checkbox>
        <el-checkbox label="SEALCHAT-Group:">Sealchat频道</el-checkbox>
      </el-checkbox-group>
    </span>
  </div>
  <div>
    <span style="margin-right: 1rem;">其他:</span>
    <el-checkbox v-model="orderByTimeDesc">按最后使用时间降序</el-checkbox>
    <el-checkbox v-model="filter30daysUnused">30天未使用</el-checkbox>
  </div>
  <div>
    <span style="margin-right: 1rem;">搜索:</span>
    <el-input v-model="searchBy" style="max-width: 15rem;" placeholder="请输入帐号或群名的一部分"></el-input>
  </div>

  <div style="margin-top: 2rem;">
    <div v-bind="containerProps" style="height: calc(100vh - 22.5rem)">
      <div v-bind="wrapperProps">
        <div v-for="item in list" :key="item.index" style="">
          <foldable-card style="margin-top: 1rem;">
            <template #title>
              <el-space class="item-header" size="large" alignment="center">
                <el-switch v-model="item.data.active" @click="item.data.changed = true"
                  style="--el-switch-on-color: var(--el-color-success); --el-switch-off-color: var(--el-color-danger)" />
                <el-space size="small" wrap>
                  <el-text style="max-width: 23rem; overflow: hidden; white-space: nowrap; text-overflow: ellipsis;" size="large" tag="strong">{{ item.data.groupId }}</el-text>
                  <el-text>「{{ item.data.groupName || '未获取到' }}」</el-text>
                </el-space>
              </el-space>
            </template>

            <template #title-extra>
              <el-button type="success" size="small" :icon="DocumentChecked" plain v-if="item.data.changed"
                         @click="saveOne(item.data, item.index)">保存</el-button>
              <template v-if="item.data.groupId.startsWith('QQ-Group:')">
                <el-tooltip v-for="_, j in item.data.diceIdExistsMap" :key="j" raw-content
                            :content="j.toString() + '<br>有二次确认'">
                  <el-button type="danger" size="small" :icon="Close" plain
                             @click="quitGroup(item.data, item.index, j.toString())">退出
                    {{ j.toString().slice(-4) }}</el-button>
                </el-tooltip>
              </template>
            </template>

            <el-descriptions>
              <el-descriptions-item label="上次使用">{{ item.data.recentDiceSendTime ?
                dayjs.unix(item.data.recentDiceSendTime).fromNow() :
                '从未'
              }}</el-descriptions-item>
              <el-descriptions-item label="入群时间">{{ item.data.enteredTime ? dayjs.unix(item.data.enteredTime).fromNow() :
                '未知'
              }}</el-descriptions-item>
              <el-descriptions-item label="邀请人">{{ item.data.inviteUserId || '未知' }}</el-descriptions-item>
              <el-descriptions-item label="Log状态">{{ item.data.logOn ? '开启' : '关闭' }}</el-descriptions-item>
              <el-descriptions-item label="迎新">{{ item.data.showGroupWelcome ? '开启' : '关闭' }}</el-descriptions-item>
              <el-descriptions-item />
              <el-descriptions-item :span="3" label="启用扩展">
                <span v-if="item.data.tmpExtList">
                  <el-tag size="small" v-for="group of item.data.tmpExtList" style="margin-right: 0.5rem;"
                    disable-transitions>{{
                      group
                    }}</el-tag>
                </span>
                <el-text v-else>'未知'</el-text>
              </el-descriptions-item>
            </el-descriptions>

            <template #unfolded-extra>
              <el-descriptions>
                <el-descriptions-item :span="2" label="上次使用">{{ item.data.recentDiceSendTime ?
                    dayjs.unix(item.data.recentDiceSendTime).fromNow() :
                    '从未'
                  }}
                </el-descriptions-item>
                <el-descriptions-item label="邀请人">{{ item.data.inviteUserId || '未知' }}</el-descriptions-item>
              </el-descriptions>
            </template>
          </foldable-card>
        </div>
      </div>
    </div>

    <!-- <el-card shadow="hover" v-for="(i, index) in groupItems" style="margin-top: 1rem;">
      <template #header>
        <div class="item-header">
          <el-space size="large" alignment="center">
            <el-switch v-model="i.active" @click="i.changed = true"
              style="--el-switch-on-color: var(--el-color-success); --el-switch-off-color: var(--el-color-danger)" />
            <el-space size="small" wrap>
              <el-text size="large" tag="strong">{{ i.groupId }}</el-text>
              <el-text>「{{ i.groupName || '未获取到' }}」</el-text>
            </el-space>
          </el-space>
          <el-space>
            <el-button type="success" size="small" :icon="DocumentChecked" plain v-if="i.changed"
              @click="saveOne(i, index)">保存</el-button>
            <el-tooltip v-for="_, j in i.diceIdExistsMap" raw-content :content="j.toString() + '<br>有二次确认'">
              <el-button type="danger" size="small" :icon="Close" plain @click="quitGroup(i, index, j.toString())">退出
                {{ j.toString().slice(-4) }}</el-button>
            </el-tooltip>
          </el-space>
        </div>
      </template>
      <el-descriptions>
        <el-descriptions-item label="上次使用">{{ i.recentDiceSendTime ? dayjs.unix(i.recentDiceSendTime).fromNow() : '从未'
        }}</el-descriptions-item>
        <el-descriptions-item label="入群时间">{{ i.enteredTime ? dayjs.unix(i.enteredTime).fromNow() : '未知'
        }}</el-descriptions-item>
        <el-descriptions-item label="邀请人">{{ i.inviteUserId || '未知' }}</el-descriptions-item>
        <el-descriptions-item label="Log状态">{{ i.logOn ? '开启' : '关闭' }}</el-descriptions-item>
        <el-descriptions-item label="迎新">{{ i.showGroupWelcome ? '开启' : '关闭' }}</el-descriptions-item>
        <el-descriptions-item />
        <el-descriptions-item :span="3" label="启用扩展">
          <span v-if="i.tmpExtList">
            <el-tag size="small" v-for="group of i.tmpExtList" style="margin-right: 0.5rem;" disable-transitions>{{ group
            }}</el-tag>
          </span>
          <el-text v-else>'未知'</el-text>
        </el-descriptions-item>
      </el-descriptions>
    </el-card> -->
  </div>
</template>

<script lang="ts" setup>
import { useStore } from '~/store'
import {
  DocumentChecked,
  Close,
} from '@element-plus/icons-vue'
import * as dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { now, sortBy } from 'lodash-es'

dayjs.extend(relativeTime)

const store = useStore()

const checkList = ref<string[]>(['QQ-Group:'])

const groupList = ref<any>({})

const orderByTimeDesc = ref(true)
const filter30daysUnused = ref(false);
const searchBy = ref('')

const groupItems = computed<any[]>(() => {
  if (groupList.value.items) {
    // const groupListItems = cloneDeep(groupList.value.items)
    let items = []
    for (let i of groupList.value.items) {
      let ok = false
      for (let f of checkList.value) {
        if (i.groupId.startsWith(f)) {
          ok = true
        }
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

      if (ok) {
        const t = Math.max(i.enteredTime || 0, i.recentCommandTime || 0, i.recentDiceSendTime || 0);
        if (filter30daysUnused.value) {
          if (now() / 1000 - t < 30 * 24 * 60 * 60) {
            ok = false;
          }
        }
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

const { list, containerProps, wrapperProps } = useVirtualList(
  groupItems,
  {
    itemHeight: 230,
  },
)

const quitTextSave = ref(false);

const saveOne = async (i: any, index: number) => {
  // await store.backupConfigSave(cfg.value)
  // console.log(222, i, index)
  await store.groupSetOne(i)
  i.changed = false
  ElMessage.success('已保存')
}

const quitGroup = async (i: any, index: number, diceId: string) => {
  const quitGroupText = localStorage.getItem('quitGroupText') || '因长期不使用等原因，骰主后台操作退出';
  ElMessageBox.prompt(
    '会进行退出留言“因长期不使用等原因，骰主后台操作退出”，输入英文大写NO则静默退出，写别的则为附加留言',
    '退出此群？',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
      inputValue: quitGroupText,
      message: h('div', null, [
        h('p', null, '会进行退出留言“因长期不使用等原因，骰主后台操作退出”，输入英文大写NO则静默退出，写别的则为附加留言'),
        h('label', {
          onInput: (e: any) => {
            quitTextSave.value = e.target.checked;
          }
        }, [
          h('input', {
            value: quitTextSave.value,
            type: 'checkbox',
          }),
          h('span', null, '设为默认'),
        ])
      ])
    }
  ).then(async (data) => {
    await store.setGroupQuit({
      groupId: i.groupId,
      diceId,
      silence: data.value === 'NO',
      extraText: data.value
    })
    if (quitTextSave.value) {
      localStorage.setItem('quitGroupText', data.value);
    }

    await refreshList()
    ElMessage.success('退出完成')

    ElMessage({
      type: 'success',
      message: '成功!',
    })
  })
}

onBeforeMount(async () => {
  await refreshList()
})
</script>

<style lang="css">
span.left {
  display: inline-block;
  min-width: 5rem;
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

.item-header {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}</style>

<template>
  <h2>黑白名单</h2>
  <div>
    <!-- <div>用户列表</div> -->
    <!-- <div>
      <div>昵称</div>
      <div>用户ID</div>
      <div>事发地点</div>
      <div>原因</div>
      <div>怒气值</div>
      <div>到期时间</div>
    </div>
    <div>群组列表</div> -->

    <el-collapse>
      <el-collapse-item name="1">
        <template #title>
          <div style="padding: 0 1rem">设置选项</div>
        </template>

        <div style="padding: 0 1rem;">
          <h3>基本设置</h3>
          <div>黑名单惩罚</div>
          <el-checkbox v-model="banConfig.banBehaviorRefuseReply">拒绝回复</el-checkbox>
          <el-checkbox v-model="banConfig.banBehaviorRefuseInvite">拒绝邀请</el-checkbox>
          <el-checkbox v-model="banConfig.banBehaviorQuitLastPlace">退出事发群</el-checkbox>
          <!-- <div>自动拉黑时长(分钟): <el-input style="max-width: 5rem;" type="number" v-model="banConfig.autoBanMinutes"></el-input></div> -->

          <h3>怒气值设置</h3>
          <small style="display: block; margin-bottom: 1rem;">说明: 海豹的黑名单使用积分制，每当用户做出恶意行为，其积分上涨一定数值，到达阈值后自动进入黑名单。会通知邀请者、通知列表、事发群(如果可能)</small>
          <div style="margin: .2rem 0;">警告阈值: <el-input style="max-width: 5rem;" type="number" v-model="banConfig.thresholdWarn"></el-input></div>
          <div style="margin: .2rem 0;">拉黑阈值: <el-input style="max-width: 5rem;" type="number" v-model="banConfig.thresholdBan"></el-input></div>

          <div style="margin: .2rem 0;margin-top:1rem">禁言增加: <el-input style="max-width: 5rem;" type="number" v-model="banConfig.scoreGroupMuted">50</el-input></div>
          <div style="margin: .2rem 0;">踢出增加: <el-input style="max-width: 5rem;" type="number" v-model="banConfig.scoreGroupKicked">50</el-input></div>
          <!-- <div>短时间多次拉群: <el-input>50</el-input></div> -->
          <!-- <div style="margin: .2rem 0;">连续指令(≥10/分钟): <el-input style="max-width: 5rem;" type="number" v-model="banConfig.scoreTooManyCommand"></el-input></div> -->

          <div style="margin: .2rem 0;margin-top:1rem"><span style="display: inline-block;min-width: 6rem">每分钟下降:</span> <el-input style="max-width: 5rem;" v-model="banConfig.scoreReducePerMinute"></el-input></div>
          <div style="margin: .2rem 0;"><span style="display: inline-block;min-width: 6rem">群组连带责任:</span> <el-input style="max-width: 5rem;" type="number" v-model="banConfig.jointScorePercentOfGroup"></el-input></div>
          <div style="margin: .2rem 0;"><span style="display: inline-block;min-width: 6rem">邀请人连带责任:</span> <el-input style="max-width: 5rem;" type="number" v-model="banConfig.jointScorePercentOfInviter"></el-input></div>

          <el-button style="margin-top: 1rem;" type="primary" @click="banConfigSave">保存设置</el-button>
        </div>
      </el-collapse-item>
    </el-collapse>

  </div>
  <div>

  </div>

  <h2>列表</h2>
  <!-- <div>
    <span style="margin-right: 1rem;">平台:</span>
    <el-checkbox v-model="showPlatformQQ">QQ</el-checkbox>
    <el-checkbox v-model="showPlatformQQCH">QQ频道</el-checkbox>
  </div> -->
  <div>
    <span style="margin-right: 1rem;">级别:</span>
    <el-checkbox v-model="showBanned">拉黑</el-checkbox>
    <el-checkbox v-model="showWarn">警告</el-checkbox>
    <el-checkbox v-model="showTrusted">信任</el-checkbox>
    <el-checkbox v-model="showOthers">其它</el-checkbox>
  </div>
  <!-- <div>
    <span style="margin-right: 1rem;">其他:</span>
    <el-checkbox v-model="orderByTimeDesc">按最后使用时间降序</el-checkbox>    
  </div> -->
  <div>
    <span style="margin-right: 1rem;">搜索:</span>
    <el-input v-model="searchBy" style="max-width: 15rem;" placeholder="请输入帐号或名字的一部分"></el-input>
  </div>

  <div style="margin-top: 2rem;">
    <div v-for="i, index in groupItems" style="margin-bottom: 2rem;border: 1px solid #ccc; border-radius: .2rem; padding: .5rem; background-color: #fff;">
      <div><span class="left">状态:</span> {{ banRankText.get(i.rank) }}</div>
      <div><span class="left">帐号:</span> {{ i.ID }}</div>
      <div><span class="left">名字:</span> {{ i.name }}</div>
      <div><span class="left">怒气值:</span> {{ i.score }}</div>
      <div>
        <span class="left">原因:</span>
        <div style="margin-left: 2rem">
          <div v-for="j, index in i.reasons">
            <el-tooltip raw-content :content="dayjs.unix(i.times[index]).format('YYYY-MM-DD HH:mm:ssZ[Z]')">
              <span>{{ dayjs.unix(i.times[index]).fromNow() }}</span>
            </el-tooltip>
            <span>，地点“{{ i.places[index] }}”，具体原因: {{j}}</span>
          </div>
        </div>
      </div>
      <el-button @click="deleteOne(i, index)">删除</el-button>
    </div>
  </div>
  <el-button @click="dialogAddShow = true">添加</el-button>

  <el-dialog v-model="dialogAddShow" title="添加用户/群组" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false" custom-class="the-dialog">
    <div>
      <span>用户ID(必填):</span>
      <el-input v-model="addData.id" placeholder="必须为 QQ:12345 或 QQ-Group:12345 格式"></el-input>
    </div>
    <div>
      <span>名称:</span>
      <el-input v-model="addData.name" placeholder="自动"></el-input>
    </div>
    <div>
      <span>原因:</span>
      <el-input v-model="addData.reason" placeholder="骰主后台设置"></el-input>
    </div>
    <div>
      <div>身份:</div>
      <el-select v-model="addData.rank">
        <el-option
          v-for="item in [{'label': '禁用', value: -30}, {'label': '信任', value: 30}]"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>
    </div>

    <template #footer>
      <span class="dialog-footer">
          <el-button @click="doAdd">添加</el-button>
          <el-button @click="dialogAddShow = false">取消</el-button>
      </span>
    </template>
  </el-dialog>
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

const groupList = ref<any>({})

const showBanned = ref(true)
const showWarn = ref(true)
const showTrusted = ref(true)
const showOthers = ref(true)
const dialogAddShow = ref(false)

const banRankText = new Map<number, string>()
banRankText.set(-30, '禁止')
banRankText.set(-10, '警告')
banRankText.set(+30, '信任')
banRankText.set(0, '常规')


const addData = ref<{ id: string, rank: number, name:string, reason: string }>({
  id: '',
  rank: -30,
  reason: '',
  name: ''
});

const doAdd = async () => {
  if (addData.value.id === '') return
  await store.banConfigMapAddOne(addData.value.id, addData.value.rank, addData.value.name, addData.value.reason)
  await refreshList()
  ElMessage.success('已保存')
  dialogAddShow.value = false
}

const searchBy = ref('')

const groupItems = computed<any[]>(() => {
  if (groupList.value) {
    // const groupListItems = cloneDeep(groupList.value.items)
    let items = []
    for (let [k, _v] of Object.entries(groupList.value)) {
      const v = _v as any
      let ok = false
      if (v.rank === -30 && showBanned.value) {
        ok = true
      }
      if (v.rank === -10 && showWarn.value) {
        ok = true
      }
      if (v.rank === 30 && showTrusted.value) {
        ok = true
      }
      if (v.rank === 0 && showOthers.value) {
        ok = true
      }

      if (ok && searchBy.value !== '') {
        let a = false
        let b = false
        if (v.ID.indexOf(searchBy.value) !== -1) {
          a = true
        }
        if (v.name.indexOf(searchBy.value) !== -1) {
          b = true
        }
        ok = a || b
      }

      if (ok) items.push(v)
    }

    // items = sortBy(items, ['recentCommandTime'])
    // if (orderByTimeDesc.value) {
    //   items = items.reverse()
    // }
    return items
  }
  return []
})

const banConfigSave = async () => {
  for (let [k, v] of Object.entries(banConfig.value)) {
    let vVal = parseFloat(v as any)
    if (!isNaN(vVal)) {
      banConfig.value[k] = vVal
    }
  }
  await store.banConfigSet(banConfig.value)
  await configGet()
  ElMessage.success('已保存')
}

const refreshList = async () => {
  const lst = await store.banConfigMapGet()
  groupList.value = lst
}

const configGet = async () => {
  banConfig.value = await store.banConfigGet()
}

const deleteOne = async (i: any, index: number) => {
  await store.banConfigMapDeleteOne(i)
  await refreshList()
  ElMessage.success('已保存')
}

const banConfig = ref<any>({})

onBeforeMount(async () => {
  await configGet()
  await refreshList()
})
</script>

<style lang="scss">
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
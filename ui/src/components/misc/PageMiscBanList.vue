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

    <el-collapse class="tip setting-tips">
      <el-collapse-item name="1">
        <template #title>
          <el-text tag="strong">设置选项</el-text>
        </template>

        <h4>基本设置</h4>
        <el-space wrap>
          <el-text>黑名单惩罚：</el-text>
          <el-checkbox v-model="banConfig.banBehaviorRefuseReply">拒绝回复</el-checkbox>
          <el-checkbox v-model="banConfig.banBehaviorRefuseInvite">拒绝邀请</el-checkbox>
          <el-checkbox v-model="banConfig.banBehaviorQuitLastPlace">退出事发群</el-checkbox>
          <!-- <div>自动拉黑时长(分钟): <el-input style="max-width: 5rem;" type="number" v-model="banConfig.autoBanMinutes"></el-input></div> -->
          <el-checkbox v-model="banConfig.banBehaviorQuitPlaceImmediately">使用时立即退出群</el-checkbox>
          <el-checkbox v-model="banConfig.banBehaviorQuitIfAdmin">使用者为管理员立即退群，为普通群员进行通告</el-checkbox>
        </el-space>

        <h4>怒气值设置</h4>
        <div style="margin-bottom: 1rem">
          <el-text size="small" type="warning" tag="p">说明：海豹的黑名单使用积分制，每当用户做出恶意行为，其积分上涨一定数值，到达阈值后自动进入黑名单。会通知邀请者、通知列表、事发群（如果可能）。</el-text>
        </div>
        <el-form size="small">
          <el-form-item label="警告阈值">
            <el-input-number v-model="banConfig.thresholdWarn" :min="0" :step="1" step-strictly></el-input-number>
          </el-form-item>
          <el-form-item label="拉黑阈值">
            <el-input-number v-model="banConfig.thresholdBan" :min="0" :step="1" step-strictly></el-input-number>
          </el-form-item>
          <el-divider />
          <el-form-item label="禁言增加">
            <el-input-number v-model="banConfig.scoreGroupMuted" :min="0" :step="1" step-strictly></el-input-number>
          </el-form-item>
          <el-form-item label="踢出增加">
            <el-input-number v-model="banConfig.scoreGroupKicked" :min="0" :step="1" step-strictly></el-input-number>
          </el-form-item>
          <el-form-item label="刷屏增加">
            <el-input-number v-model="banConfig.scoreTooManyCommand" :min="0" :step="1" step-strictly></el-input-number>
          </el-form-item>
          <el-form-item label="每分钟下降">
            <el-input-number v-model="banConfig.scoreReducePerMinute" :min="0" :step="1" step-strictly></el-input-number>
          </el-form-item>
          <el-divider />
          <el-form-item label="群组连带责任">
            <el-input-number v-model="banConfig.jointScorePercentOfGroup" :min="0" :max="1" :step="0.1"></el-input-number>
          </el-form-item>
          <el-form-item label="邀请人连带责任">
            <el-input-number v-model="banConfig.jointScorePercentOfInviter" :min="0" :max="1" :step="0.1"></el-input-number>
          </el-form-item>
        </el-form>
        <el-button type="primary" :icon="DocumentChecked" @click="banConfigSave">保存设置</el-button>
      </el-collapse-item>
    </el-collapse>

  </div>

  <header style="display: flex; justify-content: space-between; flex-wrap: wrap;">
    <h3>列表</h3>

    <el-space>
      <el-button type="success" :icon="Plus" plain @click="dialogAddShow = true">添加</el-button>
      <el-upload action="" multiple accept="application/json,.json" :show-file-list="false"
                 :before-upload="beforeUpload" style="display: flex; align-items: center;">
        <el-button type="success" :icon="Upload" plain>导入</el-button>
      </el-upload>
      <el-button type="primary" :icon="Download" plain tag="a" target="_blank"
                 :href="`${urlBase}/sd-api/banconfig/export`" style="text-decoration: none;">
        导出
      </el-button>
    </el-space>
    <!-- <div>
      <span style="margin-right: 1rem;">平台:</span>
      <el-checkbox v-model="showPlatformQQ">QQ</el-checkbox>
      <el-checkbox v-model="showPlatformQQCH">QQ频道</el-checkbox>
    </div> -->
  </header>
  <div style="margin: 0.5rem 0;">
    <span style="margin-right: 0.5rem;">级别：</span>
    <el-checkbox v-model="showBanned">拉黑</el-checkbox>
    <el-checkbox v-model="showWarn">警告</el-checkbox>
    <el-checkbox v-model="showTrusted">信任</el-checkbox>
    <el-checkbox v-model="showOthers">其它</el-checkbox>
  </div>
  <!-- <div>
    <span style="margin-right: 1rem;">其他:</span>
    <el-checkbox v-model="orderByTimeDesc">按最后使用时间降序</el-checkbox>
  </div> -->
  <div style="margin: 1rem 0;">
    <span style="margin-right: 0.5rem;">搜索：</span>
    <el-input v-model="searchBy" style="max-width: 15rem;" placeholder="请输入帐号或名字的一部分"></el-input>
  </div>

  <main style="margin-top: 2rem;">
    <el-space fill size="small">
      <el-card v-for="i, index in groupItems" :key="i.ID" shadow="hover">
        <template #header>
          <div class="ban-item-header">
            <el-space alignment="center">
              <el-tag v-if="i.rankText === '禁止'" type="danger" disable-transitions>{{ i.rankText }}</el-tag>
              <el-tag v-else-if="i.rankText === '警告'" type="warning" disable-transitions>{{ i.rankText }}</el-tag>
              <el-tag v-else-if="i.rankText === '信任'" type="success" disable-transitions>{{ i.rankText }}</el-tag>
              <el-tag v-else disable-transitions>{{ i.rankText }}</el-tag>
              <el-space size="small" alignment="center" wrap>
                <el-text size="large" tag="strong">{{ i.ID }}</el-text>
                <el-text>「{{ i.name }}」</el-text>
                <el-text size="small" tag="em">怒气值：{{ i.score }}</el-text>
              </el-space>
            </el-space>
            <el-space>
              <el-button :icon="Delete" type="danger" size="small" plain @click="deleteOne(i, index)">删除</el-button>
            </el-space>
          </div>
        </template>
        <el-space style="display: block;" direction="vertical">
          <div v-for="(j, index) in i.reasons" :key="index">
            <el-space size="small" wrap>
              <el-tooltip raw-content :content="dayjs.unix(i.times[index]).format('YYYY-MM-DD HH:mm:ssZ[Z]')">
                <el-tag size="small" type="info" disable-transitions>{{ dayjs.unix(i.times[index]).fromNow() }}</el-tag>
              </el-tooltip>
              <el-text>在&lt;{{ i.places[index] }}>，原因：「{{j}}」</el-text>
            </el-space>
          </div>
        </el-space>
      </el-card>
    </el-space>
  </main>

  <el-dialog v-model="dialogAddShow" title="添加用户/群组" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false" class="the-dialog">
    <el-form label-width="70px">
      <el-form-item label="用户ID" required>
        <el-input v-model="addData.id" placeholder="必须为 QQ:12345 或 QQ-Group:12345 格式"></el-input>
      </el-form-item>
      <el-form-item label="名称">
        <el-input v-model="addData.name" placeholder="自动"></el-input>
      </el-form-item>
      <el-form-item label="原因">
        <el-input v-model="addData.reason" placeholder="骰主后台设置"></el-input>
      </el-form-item>
      <el-form-item label="身份">
        <el-select v-model="addData.rank">
          <el-option
              v-for="item in [{'label': '禁用', value: -30}, {'label': '信任', value: 30}]"
              :key="item.value"
              :label="item.label"
              :value="item.value"
          />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
          <el-button @click="dialogAddShow = false">取消</el-button>
          <el-button type="success" @click="doAdd">添加</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import type {UploadUserFile} from "element-plus";
import { useStore } from '~/store'
import { urlBase } from '~/backend'
import {
  DocumentChecked,
  Download,
  Delete,
  Plus, Upload,
} from '@element-plus/icons-vue'
import * as dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

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

      v.rankText = banRankText.get(v.rank)

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
  const res = await ElMessageBox.confirm(
      '是否删除此记录？',
      '删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
  );
  if (res) {
    await store.banConfigMapDeleteOne(i)
    await refreshList()
    ElMessage.success('已保存')
  }
}

const banConfig = ref<any>({})

const beforeUpload = async (file: UploadUserFile) => {
  let fd = new FormData()
  fd.append('file', file as unknown as Blob)

  const c = await store.banUpload({form: fd})
  if (c.result) {
    ElMessage.success('导入黑白名单完成')
    await nextTick(async () => {
      await refreshList()
    })
  } else {
    ElMessage.error('导入黑白名单失败！' + c.err)
  }
}

onBeforeMount(async () => {
  await configGet()
  await refreshList()
})
</script>

<style scoped>
.setting-tips {
  background-color: #f3f5f7;
}

.setting-tips :deep(.el-collapse-item__header) {
  background-color: #f3f5f7;
}

.setting-tips :deep(.el-collapse-item__wrap) {
  background-color: #f3f5f7;
}
</style>

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

.ban-item-header {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}
</style>

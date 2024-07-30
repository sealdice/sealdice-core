<template>
  <Teleport v-if="store.curDice.logs.length" to="#root">
    <el-button type="default" class="btn-scrolldown" :icon="CaretBottom" circle @click="scrollDown" content="最新日志"></el-button>
  </Teleport>

  <div style="display: flex; justify-content: flex-end; align-items: center">
    <div style="display: flex; flex-direction: column;">
      <el-tooltip v-if="store.curDice.baseInfo.versionCode < store.curDice.baseInfo.versionNewCode && store.curDice.baseInfo.containerMode"
                  content="容器模式下禁止直接更新，请手动拉取最新镜像">
        <el-button type="primary" disabled>升级新版</el-button>
      </el-tooltip>
      <el-button v-else-if="store.curDice.baseInfo.versionCode < store.curDice.baseInfo.versionNewCode"
                 type="primary" @click="upgradeDialogVisible = true">
        升级新版
      </el-button>
    </div>
  </div>

  <h4>状态</h4>
  <div class="flex flex-col justify-center gap-4">
    <div class="flex items-center flex-wrap gap-1">
      <span>内存占用：</span>
      <span class="mr-2">{{filesize(store.curDice.baseInfo.memoryUsedSys || 0)}}</span>
      <el-text size="small" type="info">理论内存占用，数值偏大。系统任务管理器中的「活动内存」才是实际使用的系统内存。</el-text>
    </div>

    <div class="flex items-center flex-wrap gap-1" @click="refreshNetworkHealth">
      <el-tooltip raw-content content="点击重新进行检测">
        <span>网络质量：</span>
      </el-tooltip>

      <el-text type="primary" v-if="networkHealth.timestamp === 0">检测中…… 🤔</el-text>
      <el-text type="success" v-else-if="networkHealth.total !== 0 && networkHealth.total === networkHealth.ok?.length">优 😄</el-text>
      <el-text type="primary" v-else-if="networkHealth.ok?.includes('sign') && networkHealth.ok?.includes('seal')">一般 😐️</el-text>
      <el-text type="danger" v-else-if="networkHealth.total !== 0 && (networkHealth.ok ?? []).length === 0">网络中断 😱</el-text>
      <template v-else>
        <el-text type="warning" class="mr-4">差 ☹️</el-text>
        <el-text type="warning" size="small">这意味着你可能无法正常使用内置客户端/Lagrange 连接 QQ 平台，有时会出现消息无法正常发送的现象。</el-text>
      </template>
    
      <el-tooltip v-if="networkHealth.timestamp !== 0">
        <template #content>
          {{ dayjs.unix(networkHealth.timestamp).format('YYYY-MM-DD HH:mm:ss') }}
        </template>
        <el-text class="ml-auto" type="info" size="small">检测于 {{ dayjs.unix(networkHealth.timestamp).from(now) }}</el-text>
      </el-tooltip>
    </div>

    <div v-if="networkHealth.timestamp !== 0" class="mx-2 flex items-center gap-4">
      <el-text size="small">官网 <component :is="getWebsiteHealthComponent(networkHealth.ok?.includes('seal'))"></component></el-text>
      <el-text size="small">Lagrange Sign <component :is="getWebsiteHealthComponent(networkHealth.ok?.includes('sign'))"></component></el-text>
      <el-text size="small">Google <component :is="getWebsiteHealthComponent(networkHealth.ok?.includes('google'))"></component></el-text>
      <el-text size="small">GitHub <component :is="getWebsiteHealthComponent(networkHealth.ok?.includes('github'))"></component></el-text>
    </div>
  </div>

  <div class="flex justify-between items-center">
    <h4>日志</h4>
    <el-checkbox v-model="autoRefresh">保持刷新</el-checkbox>
  </div>

  <el-divider class="latest-log-warn">
    <el-text type="warning" size="small" class="hover:cursor-pointer" @click="scrollDown">点击下拉到底查看最新日志</el-text>
  </el-divider>

  <div class="hidden md:block p-0 logs">
    <el-table :data="store.curDice.logs"
              :row-class-name="getLogRowClassName" :header-cell-style="{backgroundColor: '#f3f5f7'}">
      <el-table-column label="时间" width="90" >
        <template #default="scope">
          <div style="display: flex; align-items: center">
            <el-icon v-if="scope.row.msg.startsWith('onebot | ')" color="var(--el-color-warning)"><timer /></el-icon>
            <el-icon v-else-if="scope.row.msg.startsWith('发给')" color="var(--el-color-primary)"><timer /></el-icon>
            <el-icon v-else-if="scope.row.level === 'warn'" color="var(--el-color-warning)"><timer /></el-icon>
            <el-icon v-else-if="scope.row.level === 'error'" color="var(--el-color-danger)"><timer /></el-icon>
            <el-icon v-else><timer /></el-icon>
            <span style="margin-left: 0.3rem">
              <span v-if="scope.row.msg.startsWith('onebot | ')" style="color: var(--el-color-warning)">{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
              <span v-else-if="scope.row.msg.startsWith('发给')" style="color: var(--el-color-primary)">{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
              <span v-else-if="scope.row.level === 'warn'" style="color: var(--el-color-warning)">{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
              <span v-else-if="scope.row.level === 'error'" style="color: var(--el-color-danger)">{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
              <span v-else>{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
            </span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="level" label="级别" width="55">
        <template #default="scope">
          <el-text v-if="scope.row.msg.startsWith('onebot | ')" type="warning">{{ scope.row.level }}</el-text>
          <el-text v-else-if="scope.row.msg.startsWith('发给')" type="primary">{{ scope.row.level }}</el-text>
          <el-text v-else-if="scope.row.level === 'warn'" type="warning">{{ scope.row.level }}</el-text>
          <el-text v-else-if="scope.row.level === 'error'" type="danger">{{ scope.row.level }}</el-text>
          <el-text v-else>{{ scope.row.level }}</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="msg" label="信息">
        <template #default="scope">
          <span v-if="scope.row.msg.startsWith('onebot | ')" style="color: var(--el-color-warning)">{{ scope.row.msg }}</span>
          <span v-else-if="scope.row.msg.startsWith('发给')" style="color: var(--el-color-primary)">{{ scope.row.msg }}</span>
          <span v-else-if="scope.row.level === 'warn'" style="color: var(--el-color-warning)">{{ scope.row.msg }}</span>
          <span v-else-if="scope.row.level === 'error'" style="color: var(--el-color-danger)">{{ scope.row.msg }}</span>
          <span v-else>{{ scope.row.msg }}</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
  <el-table :data="store.curDice.logs" class="md:hidden w-full logs"
            :row-class-name="getLogRowClassName" :header-cell-style="{backgroundColor: '#f3f5f7'}">
    <el-table-column label="时间" width="60" >
      <template #default="scope">
        <div style="display: flex; align-items: center">
          <span v-if="scope.row.msg.startsWith('onebot | ')" style="color: var(--el-color-warning)">{{ dayjs.unix(scope.row.ts).format('HH:mm') }}</span>
          <span v-else-if="scope.row.msg.startsWith('发给')" style="color: var(--el-color-primary)">{{ dayjs.unix(scope.row.ts).format('HH:mm') }}</span>
          <span v-else-if="scope.row.level === 'warn'" style="color: var(--el-color-warning)">{{ dayjs.unix(scope.row.ts).format('HH:mm') }}</span>
          <span v-else-if="scope.row.level === 'error'" style="color: var(--el-color-danger)">{{ dayjs.unix(scope.row.ts).format('HH:mm') }}</span>
          <span v-else>{{ dayjs.unix(scope.row.ts).format('HH:mm') }}</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column prop="msg" label="信息">
      <template #default="scope">
        <span v-if="scope.row.msg.startsWith('onebot | ')" style="color: var(--el-color-warning)">{{ scope.row.msg }}</span>
        <span v-else-if="scope.row.msg.startsWith('发给')" style="color: var(--el-color-primary)">{{ scope.row.msg }}</span>
        <span v-else-if="scope.row.level === 'warn'" style="color: var(--el-color-warning)">{{ scope.row.msg }}</span>
        <span v-else-if="scope.row.level === 'error'" style="color: var(--el-color-danger)">{{ scope.row.msg }}</span>
        <span v-else>{{ scope.row.msg }}</span>
      </template>
    </el-table-column>
  </el-table>

  <el-dialog v-model="upgradeDialogVisible" title="升级新版本" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="true" class="the-dialog">
    <!-- <el-checkbox v-model="importOnlyCurrent">仅当前页面(勾选)/全部自定义文案</el-checkbox> -->
    <!-- <el-checkbox v-model="importImpact">紧凑</el-checkbox> -->

    <el-link style="font-size: 16px; font-weight: bolder;" type="primary" href="https://dice.weizaima.com/changelog" target="_blank">查看更新日志</el-link>

    <div>请及时更新海豹到最新版本，这意味着功能增加和BUG修复。</div>
    <div>当然，在更新前最好看看右上角的海豹新闻，通常会很有帮助。</div>
    <div>在操作之前，最好能确保你目前可以接触到服务器，以防万一需要人工干预。</div>
    <div><b>如果升级后无法启动，请删除海豹目录中的"update"、"auto_update.exe"并手动进行升级</b></div>
    <div><b>进一步的内容请查阅届时自动生成的“升级失败指引”或加群询问。</b></div>

    <el-button style="margin: 1rem 0" type="primary" @click="doUpgrade">确认升级到 {{store.curDice.baseInfo.versionNew}} </el-button>
    
    <div>{{store.curDice.baseInfo.versionNewNote}}</div>
    <div>注意: 升级成功后界面不会自动刷新，请在重连完成后手动刷新</div>
    <div><b>当前Win11 22H2无法自动重启，建议此系统用户手动更新</b></div>
    <div>不要连续多次执行</div>

    <template #footer>
      <span class="dialog-footer">
        <!-- <el-button @click="dialogImportVisible = false">取消</el-button> -->
        <!-- <el-button @click="configForImport = ''">清空</el-button> -->
        <!-- <el-button data-clipboard-target="#import-edit" @click="copied" id="btnCopy1">复制</el-button> -->
        <!-- <el-button type="primary" @click="doImport" :disabled="configForImport === ''">导入并保存</el-button> -->
      </span>
    </template>
  </el-dialog>

  <!-- <div v-for="i in store.curDice.logs">
    {{i}}
  </div> -->
</template>

<script lang="tsx" setup>
import { Timer, CaretBottom } from '@element-plus/icons-vue'
import { useStore } from '~/store';
import dayjs from 'dayjs';
import {filesize} from 'filesize'
import {
  CircleCheckFilled,
  CircleCloseFilled,
} from '@element-plus/icons-vue'

const store = useStore()

const upgradeDialogVisible = ref(false)
const autoRefresh = ref(true)
const now = ref<dayjs.Dayjs>(dayjs())
const networkHealth = ref({
  total: 0,
  ok: [],
  timestamp: 0
} as {
  total: number,
  ok: string[],
  timestamp: number
})

let timerId: number
let checkTimerId: number

const doUpgrade = async () => {
  upgradeDialogVisible.value = false
  ElMessageBox.alert('开始下载更新，请等待……<br>完成后将自动重启海豹，并进入更新流程', '升级', { dangerouslyUseHTMLString: true })
  try {
    const ret = await store.upgrade()
    ElMessageBox.alert((ret as any).text + '<br>如果几分钟后服务没有恢复，检查一下海豹目录', '升级', { dangerouslyUseHTMLString: true })
  } catch (e) {
    // ElMessageBox.alert('升级失败', '升级')
  }
}

const scrollDown = () => {
  const panel = document.querySelector<HTMLElement>('.logs')?.parentElement;
  if (panel) {
    panel.scrollTop = panel.scrollHeight;
  }
}

const getColorByLevel = (level: string) => {
  switch (level) {
    case 'warn':
      return 'var(--el-color-warning)'
    case 'error':
      return 'var(--el-color-danger)'
    default:
      return ''
  }
}

const getLogRowClassName = ({ row }: { row: any }) => {
  switch (row.level) {
    case 'warn':
      return 'no-hover warning-row'
    case 'error':
      return 'no-hover danger-row'
    default:
      return 'no-hover normal-row'
  }
}

const getWebsiteHealthComponent = (ok: boolean): VNode => <>
  {ok ? <el-icon color={'var(--el-color-success)'}><CircleCheckFilled /></el-icon> : <el-icon color={'var(--el-color-danger)'}><CircleCloseFilled /></el-icon>}
</>

const refreshNetworkHealth = async () => {
  networkHealth.value.timestamp = 0
  const ret = await store.checkNetworkHealth()
  if (ret.result) {
    networkHealth.value = ret
  }
}

onBeforeMount(async () => {
  if (autoRefresh.value) {
    await store.logFetchAndClear()
    await refreshNetworkHealth()
  }

  timerId = setInterval(() => {
    if (autoRefresh.value) {
      store.logFetchAndClear()
    }
    now.value = dayjs()
  }, 5000) as any
  checkTimerId = setInterval(async () => {
    await refreshNetworkHealth()
  }, 5 * 60 * 1000) as any // 5 min 一次
})

onBeforeUnmount(() => {
  clearInterval(timerId)
  clearInterval(checkTimerId)
})
</script>


<style scoped>
.btn-scrolldown {
  position: absolute;
  right: 40px; 
  bottom: 60px;
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
  box-shadow: 0 0 6px rgba(0,0,0,.12);
  cursor: pointer;
  z-index: 5;
  opacity: .4;
}

.btn-scrolldown:hover {
  transition: all .3s;
  opacity: 1;  
}

.latest-log-warn {
  margin-top: 0;
  margin-bottom: 1rem;
  :deep(.el-divider__text) {
    background: #f3f4f6;
  }
}
</style>

<style lang="css">
.el-table .warning-row {
  --el-table-tr-bg-color: var(--el-color-warning-light-8);
  &:hover {
    --el-table-tr-bg-color: var(--el-color-warning-light-9);
  }
}

.el-table .danger-row {
  --el-table-tr-bg-color: var(--el-color-danger-light-8);
  &:hover {
    --el-table-tr-bg-color: var(--el-color-danger-light-9);
  }
}

.el-table .normal-row {
  --el-table-tr-bg-color: #f3f5f7;
  &:hover {
    --el-table-tr-bg-color: var(--el-color-primary-light-9);
  }
}

.no-hover:hover > td {
  background-color: initial !important;
}
</style>
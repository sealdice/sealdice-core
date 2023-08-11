<template>
  <Teleport v-if="store.curDice.logs.length" to="#root">
    <el-button type="default" class="btn-scrolldown" :icon="CaretBottom" circle @click="scrollDown"></el-button>
  </Teleport>

  <p style="display: flex;justify-content: space-between;">
    <span style="display: flex; align-items: center;">
      <span>内存占用: </span>
      <span>{{filesize(store.curDice.baseInfo.memoryUsedSys || 0)}}</span>
      <el-tooltip raw-content content="理论内存占用，偏大。任务管理器中的“活动内存”为实际使用的系统内存">
        <el-icon><question-filled /></el-icon>
      </el-tooltip>
    </span>

    <span style="display: flex; align-self: flex-end; flex-direction: column;">
      <el-button v-if="store.curDice.baseInfo.versionCode < store.curDice.baseInfo.versionNewCode" type="primary" @click="upgradeDialogVisible = true">升级新版</el-button>
      <el-checkbox v-model="autoRefresh">保持刷新</el-checkbox>
    </span>
  </p>
  <div style=" padding: 0 1rem; background: #fff;" class="hidden-xs-only">
    <el-table :data="store.curDice.logs">
      <el-table-column label="时间" width="110" >
        <template #default="scope">
          <div style="display: flex; align-items: center">
            <el-icon><timer /></el-icon>
            <span style="margin-left: 10px">{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="level" label="级别" width="85" />
      <el-table-column prop="msg" label="信息">
        <template #default="scope">
          <span v-if="scope.row.msg.startsWith('onebot | ')" style="color: #DB7E44">{{ scope.row.msg }}</span>
          <!-- <span v-else-if="scope.row.msg.startsWith('收到') && scope.row.msg.includes('的指令')" style="color: #445ddb">{{ scope.row.msg }}</span> -->
          <span v-else-if="scope.row.msg.startsWith('发给')" style="color: #445ddb">{{ scope.row.msg }}</span>
          <span v-else-if="scope.row.level === 'error'" style="color: #c00000">{{ scope.row.msg }}</span>
          <span v-else>{{ scope.row.msg }}</span>
        </template>
      </el-table-column>
    </el-table>
  </div>

  <el-table :data="store.curDice.logs" style="width: 100%;" class="hidden-sm-and-up">
    <el-table-column label="时间" width="65" >
      <template #default="scope">
        <div style="display: flex; align-items: center">
          <!-- <el-icon><timer /></el-icon> -->
          <span>{{ dayjs.unix(scope.row.ts).format('HH:mm') }}</span>
        </div>
      </template>
    </el-table-column>
    <!-- <el-table-column prop="level" label="级别" width="60" /> -->
    <el-table-column prop="msg" label="信息">
      <template #default="scope">
        <span v-if="scope.row.msg.startsWith('onebot | ')" style="color: #DB7E44">{{ scope.row.msg }}</span>
        <span v-else-if="scope.row.msg.startsWith('发给')" style="color: #445ddb">{{ scope.row.msg }}</span>
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

<script lang="ts" setup>
import { Timer, CaretBottom } from '@element-plus/icons-vue'
import { computed, onBeforeMount, onBeforeUnmount, onMounted, ref } from 'vue';
import { useStore } from '~/store';
import * as dayjs from 'dayjs'
import filesize from 'filesize'
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
import { ElMessage, ElMessageBox } from 'element-plus'

const store = useStore()

const upgradeDialogVisible = ref(false)
const autoRefresh = ref(true)

let timerId: number

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
  const panel = document.querySelector<HTMLElement>('.main-container')?.parentElement;
  if (panel) {
    panel.scrollTop = panel.scrollHeight;
  }
}

onBeforeMount(async () => {
  if (autoRefresh.value) {
    await store.logFetchAndClear()
  }

  timerId = setInterval(() => {
    if (autoRefresh.value) {
      store.logFetchAndClear()
    }
  }, 5000) as any
})

onBeforeUnmount(() => {
  clearInterval(timerId)
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
</style>
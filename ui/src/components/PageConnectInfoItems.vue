<template>
  <!-- <div style="position: relative;"> -->
  <div style="position: absolute; right: 40px; bottom: 60px; z-index: 10">
    <el-button type="primary" class="btn-add" :icon="Plus" circle @click="addOne"></el-button>
  </div>
  <!-- </div> -->

  <div v-if="(!store.curDice.conns) || (store.curDice.conns && store.curDice.conns.length === 0)">
    <span style="vertical-align: middle;">似乎还没有账号，</span>
    <el-link style="font-size: 16px; font-weight: bolder;" type="primary" @click="addOne">点我添加一个</el-link>
  </div>

  <div style="display: flex; flex-wrap: wrap;">
    <div v-for="i, index in reactive(store.curDice.conns)" style="width: 20rem; flex: 1 0 50%; flex-grow: 0;">
      <el-card class="box-card" style="margin-right: 1rem; margin-bottom: 1rem; position: relative">
        <template #header>
          <div class="card-header">
            <span>{{i.nickname || '<未知>'}}({{i.userId}})</span>
            <!-- <el-button class="button" type="text"  @click="doModify(i, index)">修改</el-button> -->
            <el-button class="button" type="text"  @click="doRemove(i)">删除</el-button>
          </div>
        </template>

        <div style="position: absolute; width: 17rem; height: 14rem; background: #fff; z-index: 1;" v-if="i.adapter?.inPackGoCqHttpQrcodeReady && store.curDice.qrcodes[i.id]">
          <div style="margin-left: 2rem">需要同账号的手机QQ扫码登录:</div>
          <img style="width: 10rem; height:10rem; margin-left: 3.5rem; margin-top: 2rem;" :src="store.curDice.qrcodes[i.id]" />
        </div>

        <el-form ref="formRef" :model="i" label-width="100px">
          <!-- <el-form-item label="帐号">
            <el-input v-model="i.account"></el-input>
            <div>123456789<el-tag size="small">{{i.platform}}</el-tag></div>
          </el-form-item>

          <el-form-item label="昵称">
            <div>阮鸫</div>
          </el-form-item> -->

          <el-form-item label="状态">
            <el-space>
              <div v-if="i.state === 0"><el-tag type="danger">断开</el-tag></div>
              <div v-if="i.state === 2"><el-tag type="warning">连接中</el-tag></div>
              <div v-if="i.state === 1"><el-tag type="success">已连接</el-tag></div>
              <el-tooltip :content="`看到这个标签是因为最近20分钟内有风控警告，将在重新登录后临时消除。触发时间: ` + dayjs.unix(i.adapter?.inPackGoCqHttpLastRestricted).fromNow()" v-if="Math.round(new Date().getTime()/1000) - i.adapter?.inPackGoCqHttpLastRestricted < 30 * 60">
                <el-tag type="warning">风控</el-tag>
              </el-tooltip>
            </el-space>
          </el-form-item>

          <el-form-item label="在线时长">
            <div>{{i.onlineTotalTime}} 未实现</div>
          </el-form-item>

          <el-form-item label="群组数量">
            <div>{{i.groupNum}}</div>
          </el-form-item>

          <el-form-item label="累计响应指令">
            <div>{{i.cmdExecutedNum}}</div>
          </el-form-item>

          <el-form-item label="上次执行指令">
            <div v-if="i.cmdExecutedLastTime">{{dayjs.unix(i.cmdExecutedLastTime).fromNow()}}</div>
            <div v-else>从未</div>
          </el-form-item>

          <el-form-item label="连接地址">
            <!-- <el-input v-model="i.connectUrl"></el-input> -->
            <div>{{i.adapter?.connectUrl}}</div>
          </el-form-item>

          <!-- <el-form-item label="密码">
            <el-input type="password" v-model="i.password"></el-input>
          </el-form-item> -->

          <!-- <el-form-item label="启用">
            <el-switch v-model="i.enable"></el-switch>
          </el-form-item> -->

          <!-- <el-form-item label=""> -->
          <div style="display: flex;justify-content: center; margin-bottom: 1rem;">
            <el-button-group>
              <el-tooltip content="如果日志中出现帐号被风控，可以试试这个功能" placement="bottom-start">
                <el-button type="warning" @click="askGocqhttpReLogin(i)">重新登录</el-button>
              </el-tooltip>
              <el-tooltip content="离线/启用此账号，重启骰子后仍将保持离线/启用状态" placement="bottom-start">
                <el-button type="warning" @click="askSetEnable(i, false)" v-if="i.enable">禁用</el-button>
                <el-button type="warning" @click="askSetEnable(i, true)" v-else>启用</el-button>
              </el-tooltip>
            </el-button-group>
          </div>
          <!-- </el-form-item> -->

        </el-form>
      </el-card>
    </div>
  </div>

  <el-dialog v-model="dialogFormVisible" title="帐号登录" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false" custom-class="the-dialog">
    <template v-if="form.step === 1">
      <el-form :model="form">
        <el-form-item label="类型" :label-width="formLabelWidth">
          <div>QQ账号</div>
        </el-form-item>

        <el-form-item label="设备" :label-width="formLabelWidth" required>
          <el-select v-model="form.protocol">
            <el-option label="iPad 协议" :value="0"></el-option>
            <el-option label="Android 协议 - 稳定协议，建议！" :value="1"></el-option>
            <el-option label="Android 手表协议 - 可以和手机QQ共存" :value="2"></el-option>
            <!-- <el-option label="MacOS" :value="3"></el-option> -->
          </el-select>
        </el-form-item>

        <el-form-item label="账号" :label-width="formLabelWidth" required>
          <el-input v-model="form.account" type="number" autocomplete="off"></el-input>
        </el-form-item>

        <el-form-item label="密码" :label-width="formLabelWidth">
          <el-input v-model="form.password" type="password" autocomplete="off"></el-input>
          <small>
            <div>提示: 新设备首次登录多半需要手机版扫码，建议先准备好</div>
            <div>能够进行扫码登录（不填写密码即可），但注意扫码登录不支持自动重连。</div>
            <div>如果出现“要求同一WIFI扫码”可以本地登录后备份，复制到服务器上。</div>
            <div v-if="form.protocol != 1" style="color: #aa4422;">提示: iPad或者Android手表协议非常容易出问题，如果失败，可以删掉重登多试几次！</div>
          </small>
        </el-form-item>

      </el-form>
    </template>
    <template v-else-if="form.step === 2">
      <el-timeline style="min-height: 260px;">
        <el-timeline-item
          v-for="(activity, index) in activities"
          :key="index"
          :type="activity.type"
          :color="activity.color"
          :size="activity.size"
          :hollow="activity.hollow"
        >
          <div>{{ activity.content }}</div>
          <div v-if="index === 2 && curConn.adapter?.inPackGoCqHttpQrcodeReady">
            <div>登录需要滑条验证码, 请使用登录此账号的手机QQ扫描二维码以继续登录:</div>
            <img :src="store.curDice.qrcodes[curConn.id]" style="width: 12rem; height: 12rem" />
          </div>
          <div v-if="index === 2 && curConn.adapter?.inPackGoCqHttpLoginDeviceLockUrl">
            <div>账号已开启设备锁，请访问此链接进行验证：</div>
            <div>
              <el-link :href="curConn.adapter?.inPackGoCqHttpLoginDeviceLockUrl" target="_blank">{{curConn.adapter?.inPackGoCqHttpLoginDeviceLockUrl}}</el-link>
            </div>
            <div>
              <div>确认验证完成后，点击此按钮：</div>
              <div>
                <el-button type="warning" @click="gocqhttpReLogin(curConn)" :disabled="duringRelogin">下一步</el-button>
              </div>
            </div>
          </div>
          <div v-if="index === 2 && (!curConn.adapter?.inPackGoCqHttpRunning && !curConn.adapter?.inPackGoCqHttpLoginDeviceLockUrl) && (!isRecentLogin)">
            <div>
              <div>登录失败!可能是以下原因：</div>
              <ul>
                <li>密码错误(注意检查大小写)</li>
                <li>二维码获取失败(日志中会有“二维码错误”)</li>
                <li>扫二维码超时过期(约2分钟)</li>
                <li>海豹发生了故障</li>
              </ul>
              <div>具体请参见日志。如果不出现“确定”按钮，请直接刷新。</div>
              <div>若删除账号重复尝试无果，请回报bug给开发者。</div>
            </div>
          </div>
        </el-timeline-item>
      </el-timeline>
    </template>
    <template v-else-if="form.step === 3">
      <el-result
        icon="success"
        title="成功啦!"
        sub-title="现在账号状态应该是“已连接”了，去试一试骰子吧！"
      >
        <!-- <template #extra></template> -->
      </el-result>
    </template>

    <template #footer>
      <span class="dialog-footer">
        <template v-if="form.step === 1">
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" @click="goStepTwo" :disabled="form.account==''">下一步</el-button>
        </template>
        <template v-if="form.isEnd">
          <el-button @click="formClose">确定</el-button>
        </template>
      </span>
    </template>
  </el-dialog>

</template>

<script lang="ts" setup>
import { reactive, onBeforeMount, onBeforeUnmount, ref, nextTick } from 'vue';
import { useStore } from '~/store';
import type { DiceConnection } from '~/store';
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { sleep } from '~/utils'
import { delay } from 'lodash-es'
import * as dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

dayjs.extend(relativeTime)

const fullActivities = [
  {
    content: '正在生成虚拟设备信息',
    size: 'large',
    type: 'primary',
    hollow: true,
  },
  {
    content: '正在生成登录配置文件',
    size: 'large',
    color: '#0bbd87',
    hollow: true,
  },
  {
    content: '进行登录……',
    size: 'large',
    flag: true
  },
  {
    content: '完成!',
    type: 'primary',
    hollow: true,
  },

  {
    content: '进行重新登录流程',
    type: 'large',
    hollow: true,
  },
  {
    content: '如果卡在这里不出二维码，可以尝试1分钟后刷新页面，再次点击登录。如果还不行请删除此账号重新添加',
    type: 'large',
    hollow: true,
  },
]
const activities = ref([] as typeof fullActivities)

const store = useStore()

const isRecentLogin = ref(false)
const duringRelogin = ref(false)
const dialogFormVisible = ref(false)
const formLabelWidth = '100px'

const curConn = ref({} as DiceConnection);
const curConnId = ref('');

const setRecentLogin = () => {
  isRecentLogin.value = true
  setTimeout(() => {
    isRecentLogin.value = false
  }, 3000)
}

const goStepTwo = async () => {
  form.step = 2
  curConnId.value = '';
  setRecentLogin()

  store.addImConnection(form).then((conn) => {
    curConnId.value = conn.id;
  }).catch(e => {
    console.log(e)
    ElMessageBox.alert('似乎已经添加了这个账号！', '添加失败')
    formClose()
  })
  activities.value = []
  await sleep(500)
  activities.value.push(fullActivities[0])
  await sleep(1000)
  activities.value.push(fullActivities[1])
  await sleep(1000)
  activities.value.push(fullActivities[2])
}

const formClose = async () => {
  curConnId.value = ''
  dialogFormVisible.value = false;
  form.step = 1;
  form.isEnd = false;
}

const setEnable = async (i: DiceConnection, val: boolean) => {
  const ret = await store.getImConnectionsSetEnable(i, val)
  i.enable = ret.enable
  ElMessage.success('状态修改完成')
  if (val) {
    setRecentLogin()
    // 若是启用骰子，走登录流程
    curConnId.value = '' // 先改掉这个，以免和当前连接一致，导致被瞬间重置
    nextTick(() => {
      curConnId.value = i.id
    })
    // store.gocqhttpReloginImConnection(i).then(theConn => {
    //   curConnId.value = i.id;
    // })

    // 重复登录时，也打开这个窗口
    activities.value = []
    dialogFormVisible.value = true
    form.step = 2

    activities.value.push(fullActivities[4])
    activities.value.push(fullActivities[5])
    activities.value.push(fullActivities[2])
  }
}

const askSetEnable = async (i: DiceConnection, val: boolean) => {
  ElMessageBox.confirm(
    '确认修改此账号的在线状态吗？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    setEnable(i, val)
  })
}

const askGocqhttpReLogin = async (i: DiceConnection) => {
  ElMessageBox.confirm(
    '重新登录吗？有可能要再次扫描二维码',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    gocqhttpReLogin(i)
  })
}

const gocqhttpReLogin = async (i: DiceConnection) => {
  setRecentLogin()
  curConnId.value = ''; // 先改掉这个，以免和当前连接一致，导致被瞬间重置
  store.gocqhttpReloginImConnection(i).then(theConn => {
    curConnId.value = i.id;
  })
  // 重复登录时，也打开这个窗口
  activities.value = []
  dialogFormVisible.value = true
  form.step = 2

  activities.value.push(fullActivities[4])
  activities.value.push(fullActivities[5])
  activities.value.push(fullActivities[2])
}

const form = reactive({
  step: 1,
  isEnd: false,
  account: '',
  password: '',
  protocol: 1
})

const addOne = () => {
  dialogFormVisible.value = true
}

let timerId: number

onBeforeMount(async () => {
  await store.getImConnections()
  for (let i of store.curDice.conns || []) {
    delete store.curDice.qrcodes[i.id]
  }

  timerId = setInterval(async () => {
    console.log('refresh')
    await store.getImConnections()

    for (let i of store.curDice.conns || []) {
      if (i.adapter?.inPackGoCqHttpQrcodeReady && !store.curDice.qrcodes[i.id] && !i.adapter?.inPackGoCqHttpLoginSuccess) {
        store.curDice.qrcodes[i.id] = (await store.getImConnectionsQrCode(i)).img
      }

      if (i.adapter?.inPackGoCqHttpLoginSuccess) {
        store.curDice.qrcodes[i.id] = ''
      }

      if (i.id === curConnId.value) {
        curConn.value = i;

        if (i.adapter?.inPackGoCqHttpLoginDeviceLockUrl === "") {
          if (!i.adapter?.inPackGoCqHttpRunning) {
            form.isEnd = true;
          }
        }

        if (i.adapter?.inPackGoCqHttpLoginSuccess) {
          activities.value.push(fullActivities[3])
          await sleep(1000)
          form.step = 3
          form.isEnd = true
        }
        break;
      }
    }

  }, 3000) as any
})

onBeforeUnmount(() => {
  clearInterval(timerId)
})

const doRemove = async (i: DiceConnection) => {
  ElMessageBox.confirm(
    '删除此项帐号及其关联数据，确定吗？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    await store.removeImConnection(i)
    await store.getImConnections()
    ElMessage({
      type: 'success',
      message: '删除成功!',
    })
  })
}

const doModify = () => {
  ElMessage.success('此功能尚未实现……')
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.btn-add {
  width: 3rem;
  height: 3rem;
  font-size: 2rem;
  font-weight: bold;
}

</style>

<style>
.the-dialog {
  min-width: 370px;
}
</style>

<template>
  <div id="root"
       style="background: #545c64; height: 100%; display: flex; flex-direction: column; max-width: 950px; width: 100%; margin: 0 auto; position: relative;">
    <nav class="nav" style="display: flex; color: #fff; justify-content: space-between;">
      <el-space alignment="center" :size="0" style="height: 60px;">
        <div class="menu-button-wrapper">
          <el-button type="text" size="large" @click="drawerMenu = true">
            <el-icon color="#fff" size="1.5rem">
              <IconMenu/>
            </el-icon>
          </el-button>
        </div>

        <el-space :v-show="store.canAccess" direction="vertical" alignment="flex-start" :size="0" style="">
          <el-space size="small" alignment="center">
            <span @click="enableAdvancedConfig" style="font-size: 1.2rem; cursor: pointer;">SealDice</span>
            <el-tooltip v-if="store.diceServers.length > 0 && store.diceServers[0].baseInfo.containerMode" class="flex items-center">
              <template #content>当前以容器模式启动，部分功能受到限制。</template>
              <el-icon type="info">
                <i-carbon-container-software/>
              </el-icon>
            </el-tooltip>
          </el-space>
          <span v-if="store.diceServers.length > 0" size="small" style="font-size: .7rem;">
            {{ store.diceServers[0].baseInfo.OS }} - {{ store.diceServers[0].baseInfo.arch }}
          </span>
        </el-space>
      </el-space>

      <el-space v-show="store.canAccess" size="large"
                style="color: #fff; font-size: small; text-align: right;">
        <div @click="dialogFeed = true"
             style="cursor: pointer;">
          <el-badge value="new" :hidden="newsChecked">
            <img :src="imgNews" alt="news" style="width: 2.3rem;">
          </el-badge>
        </div>

        <div style="display: flex; flex-direction: column; align-items: center;">
          <div style="display: flex; align-items: center;">
            <el-tag effect="dark" size="small" disable-transitions
                    style="margin-right: 0.3rem;"
                    :type="store.curDice.baseInfo.appChannel === 'stable' ? 'success' : 'info'">
              {{ store.curDice.baseInfo.appChannel === 'stable' ? '正式版' : '测试版' }}
            </el-tag>
            <el-tooltip :content="store.curDice.baseInfo.version" placement="bottom">
              <el-text size="large" style="color: #fff">
                {{ store.curDice.baseInfo.versionSimple }}
              </el-text>
            </el-tooltip>
          </div>
          <div v-if="store.curDice.baseInfo.versionCode < store.curDice.baseInfo.versionNewCode">
            🆕{{ store.curDice.baseInfo.versionNew }}
          </div>
        </div>
      </el-space>
    </nav>

    <div style="display: flex;">
      <div class="menu" style="position: relative; background: #545c64">
        <Menu type="dark" v-model:advancedConfigCounter="advancedConfigCounter" @swithch-to="switchTo"/>
      </div>

      <div style="background-color: #f3f5f7; flex: 1; text-align: left; height: calc(100vh - 4rem); overflow-y: auto;">
        <div class="main-container" :class="[needh100 ? 'h100' : '']" ref="rightbox">
          <page-misc v-if="tabName === 'miscSettings'" :category="miscSettingsCategory" @update:advanced-settings-show="(show) => refreshAdvancedSettings(show)"/>
          <page-log v-if="tabName === 'log'"/>
          <page-connect-info-items v-if="tabName === 'imConns'"/>
          <page-custom-text v-if="tabName === 'customText'" :category="textCategory"/>
          <page-custom-reply v-if="tabName === 'customReply'"/>
          <page-mod v-if="tabName === 'mod'" :category="textCategory"/>
          <page-test v-if="tabName === 'test'"/>
          <page-about v-if="tabName === 'about'"/>
        </div>
      </div>
    </div>
  </div>

  <el-drawer v-model="drawerMenu" direction="ltr" :show-close="false"
             size="50%" class="drawer-menu">
    <template #header>
      <div style="color: #fff; display: flex; align-items: center; justify-content: space-between;">
        <el-space :v-show="store.canAccess" direction="vertical" alignment="flex-start" :size="0">
          <span @click="enableAdvancedConfig" style="font-size: 1.2rem; cursor: pointer;">SealDice</span>
          <span v-if="store.diceServers.length > 0" style="font-size: .7rem;">
            {{ store.diceServers[0].baseInfo.OS }} - {{ store.diceServers[0].baseInfo.arch }}
          </span>
        </el-space>

        <el-tag effect="dark" size="small" disable-transitions
                :type="store.curDice.baseInfo.appChannel === 'stable' ? 'success' : 'info'">
          {{ store.curDice.baseInfo.appChannel === 'stable' ? '正式版' : '测试版' }}
        </el-tag>
      </div>
    </template>
    <Menu type="dark" v-model:advancedConfigCounter="advancedConfigCounter" @swithch-to="switchTo"/>
  </el-drawer>

  <el-dialog v-model="showDialog" title="" :close-on-click-modal="false" :close-on-press-escape="false"
             :show-close="false" class="the-dialog">
    <h3>输入密码解锁</h3>
    <el-input v-model="password" type="password"></el-input>
    <el-button type="primary" style="padding: 0px 50px; margin-top: 1rem;" @click="doUnlock">确认</el-button>
  </el-dialog>

  <el-dialog v-model="dialogLostConnectionVisible" title="主程序离线" :close-on-click-modal="false"
             :close-on-press-escape="false" :show-close="false" class="the-dialog">
    <div>与主程序断开连接，请耐心等待连接恢复</div>
    <div>如果失去响应过久，请登录服务器处理</div>
  </el-dialog>

  <el-dialog v-model="dialogFeed" :close-on-click-modal="false" :close-on-press-escape="false" class="dialog-feed"
             :show-close="false">
    <template #header="{ close, titleId, titleClass }">
      <div class="my-header">
        <h4 :id="titleId" :class="titleClass" style="margin: 0.5rem">海豹新闻</h4>
        <el-button type="success" :icon="Check" @click="checkNews(close)">确认已读</el-button>
      </div>
    </template>

    <div style="text-align: left;" v-html="newsData">
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import PageCustomText from "./components/PageCustomText.vue";
import PageConnectInfoItems from "./components/PageConnectInfoItems.vue";
import PageMisc from "./components/PageMisc.vue"
import PageMod from "./components/PageMod.vue"
import PageCustomReply from "./components/mod/PageCustomReply.vue"
import PageLog from "./components/PageLog.vue";
import PageAbout from "./components/PageAbout.vue"
import PageTest from "./components/PageTest.vue"
import {onBeforeMount, ref, watch, computed} from 'vue'
import {useStore} from './store'
import {ElMessage, ElMessageBox} from 'element-plus'
import imgNews from '~/assets/news.png'

import {
  Check,
  Menu as IconMenu,
} from '@element-plus/icons-vue'

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
import {CircleCloseFilled} from '@element-plus/icons-vue'

import {passwordHash} from "./utils"
import {delay} from "lodash-es"

dayjs.locale('zh-cn')
dayjs.extend(relativeTime);

const store = useStore()
const password = ref('')

const dialogFeed = ref(false)

const newsData = ref(`<div>暂无内容</div>`)
const newsChecked = ref(true)
const newsMark = ref('')
const checkNews = async (close: any) => {
  console.log('newsMark', newsMark.value)
  const ret = await store.checkNews(newsMark.value)
  if (ret?.result) {
    ElMessage.success('已阅读最新的海豹新闻')
  } else {
    ElMessage.error('阅读海豹新闻失败')
  }
  await updateNews()
  close()
}
const updateNews = async () => {
  const newsInfo = await store.news()
  if (newsInfo.result) {
    newsData.value = newsInfo.news
    newsChecked.value = newsInfo.checked
    newsMark.value = newsInfo.newsMark
  } else {
    ElMessage.error(newsInfo?.err ?? '获取海豹新闻失败')
  }
}

const showDialog = computed(() => {
  return !store.canAccess
})

const dialogLostConnectionVisible = ref(false)

const doUnlock = async () => {
  const hash = await passwordHash(store.salt, password.value)
  await store.signIn(hash)
  if (store.canAccess) {
    ElMessageBox.alert('欢迎回来，请开始使用。', '登录成功')
    password.value = ''
    checkPassword();
    window.location.reload();
  } else {
    ElMessageBox.alert('错误的密码', '登录失败')
    password.value = ''
  }
}

const checkPassword = async () => {
  if (!await store.checkSecurity()) {
    ElMessageBox.alert('欢迎使用海豹核心。<br/>如果您的服务开启在公网，为了保证您的安全性，请前往<b>“综合设置->基本设置”</b>界面，设置<b>UI界面密码</b>。<br/>或切换为只有本机可访问。<br><b>如果您不了解上面在说什么，请务必设置一个密码</b>', '提示', {dangerouslyUseHTMLString: true})
  }
}

onBeforeMount(async () => {
  store.getBaseInfo()
  store.getCustomText()

  if (store.canAccess) {
    checkPassword()
  }

  timerId = setInterval(async () => {
    try {
      await store.getBaseInfo()
      if (dialogLostConnectionVisible.value) {
        dialogLostConnectionVisible.value = false
      }
    } catch (e: any) {
      if (!e.response) {
        // 此时是连接不上，404
        // e.response.status 有可能为403
        dialogLostConnectionVisible.value = true
      }
    }
  }, 5000) as any

  await updateNews()

  const conf = await store.diceAdvancedConfigGet()
  if (conf.show) {
    advancedConfigCounter.value = 8
  }
})

let timerId: number

const handleOpen = (key: string, keyPath: string[]) => {
}
const handleClose = (key: string, keyPath: string[]) => {
}

const rightbox = ref(null)

let tabName = ref("log")
let textCategory = ref("")
let miscSettingsCategory = ref("")

const needh100 = ref(false)

const drawerMenu = ref<boolean>(false)
const switchTo = (tab: 'overview' | 'miscSettings' | 'log' | 'customText' | 'mod' | 'customReply' | 'imConns' | 'banList' | 'test' | 'about', name: string = '') => {
  tabName.value = tab
  textCategory.value = ''
  if (tab === 'customText') {
    textCategory.value = name
  }
  if (tab === 'mod') {
    textCategory.value = name
  }
  if (tab === 'miscSettings') {
    miscSettingsCategory.value = name
  }
  needh100.value = ['test'].includes(tab)
  drawerMenu.value = false
}

let configCustom = {}

let advancedConfigCounter = ref<number>(0)
const enableAdvancedConfig = async () => {
  advancedConfigCounter.value++
  const counter = advancedConfigCounter.value
  if (counter > 8) {
    ElMessage.info('高级设置页已经开启')
    return
  } else if (counter === 8) {
    let conf = await store.diceAdvancedConfigGet()
    conf.show = true
    conf.enable = true
    await store.diceAdvancedConfigSet(conf)
    ElMessage.success('已开启高级设置页')
  } else if (counter > 2) {
    ElMessage.info('再按 ' + (8 - counter) + ' 次开启高级设置页')
  }
}

const refreshAdvancedSettings = async (show: boolean) => {
  if (!show) {
    advancedConfigCounter.value = 0
    switchTo('log')
    ElMessage.success('已关闭高级设置页')
  }
}
</script>

<style>
html,
body {
  height: 100%;
}

::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track-piece {
  background: #fafafa;
}

::-webkit-scrollbar-thumb {
  background: #bdbdbd;
}

::-webkit-scrollbar-corner {
  background: #fafafa;
}

::-webkit-scrollbar-thumb:window-inactive {
  background: #e0e0e0;
}

::-webkit-scrollbar-thumb:hover {
  background: #9e9e9e;
}

.main-container {
  padding: 2rem;
  position: relative;
  box-sizing: border-box;
  min-height: 100%;
}

.h100 {
  height: 100%;
}

@media screen and (max-width: 640px) {
  .nav {
    margin: 0 0.5rem 0 0;
  }

  .menu {
    display: none;
  }

  .main-container {
    padding: 1rem;
  }
}

@media screen and (min-width: 640px) {
  .nav {
    margin: 0 1rem 0 1.5rem;
  }

  .menu-button-wrapper {
    display: none;
  }
}

.sd-center {
  display: flex;
  align-items: center;
  justify-content: center;
}

#app {
  font-family: "PingFang SC", "Helvetica Neue", "Hiragino Sans GB", "Segoe UI",
  "Microsoft YaHei", "微软雅黑", sans-serif;
  /* font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', '微软雅黑', Arial, sans-serif; */
  text-align: center;
  color: #2c3e50;
  height: 100%;
  display: flex;
}

.element-plus-logo {
  width: 50%;
}

.my-header {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
}

@media screen and (max-width: 640px) {
  .dialog-feed {
    width: 90% !important;
  }
}

.drawer-menu {
  background-color: #545c64;

  .el-drawer__header {
    margin: 0;
    padding: 1rem;
  }

  .el-drawer__body {
    padding: 0;
  }
}
</style>

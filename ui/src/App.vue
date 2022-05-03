<template>
  <!-- <BaseHeader /> -->
  <!-- <HelloWorld msg="Hello Vue 3.0 + Element Plus + Vite" /> -->

  <div
    style="background: #545c64; height: 100%; display: flex; flex-direction: column; max-width: 900px; width: 100%; margin: 0 auto; position: relative;"
  >
    <h3
      class="mb-2"
      style="color: #f8ffff; text-align: left; padding-left: 2em; font-weight: normal;max-height:60px; height: 60px;"
    >
      <span :v-show="store.canAccess">SealDice</span>
    </h3>

    <div
      :v-show="store.canAccess"
      style="position: absolute; top: 1rem; right: 10px; color: #fff; font-size: small;"
    >{{ store.curDice.baseInfo.version }}</div>

    <div style="display: flex;">
      <div style="position: relative; background: #545c64">
        <el-menu
          :collapse="sideCollapse"
          style="border-right: 0;"
          active-text-color="#ffd04b"
          background-color="#545c64"
          class="el-menu-vertical-demo"
          default-active="2"
          text-color="#fff"
          @open="handleOpen"
          @close="handleClose"
        >
          <!-- <el-menu-item index="1" @click="switchTo('overview')">
            <el-icon>
              <setting />
            </el-icon>
            <span>总览</span>
          </el-menu-item>-->

          <el-menu-item index="2" @click="switchTo('log')">
            <el-icon>
              <location />
            </el-icon>
            <span>日志</span>
          </el-menu-item>

          <el-menu-item index="3" @click="switchTo('imConns')">
            <el-icon>
              <icon-menu />
            </el-icon>
            <span>账号设置</span>
          </el-menu-item>

          <el-sub-menu index="4">
            <template #title>
              <el-icon>
                <setting />
              </el-icon>
              <span>自定义文案</span>
            </template>

            <el-menu-item
              :index="`5-${k}`"
              @click="switchTo('customText', k.toString())"
              v-for="_, k in store.curDice.customTexts"
            >
              <span>{{ k }}</span>
            </el-menu-item>
          </el-sub-menu>

          <el-menu-item index="5" @click="switchTo('customReply')">
            <el-icon>
              <setting />
            </el-icon>
            <span>自定义回复</span>
          </el-menu-item>

          <!-- 
          <el-menu-item index="4">
            <el-icon>
              <setting />
            </el-icon>
            <span>扩展管理</span>
          </el-menu-item>

          <el-menu-item index="5">
            <el-icon>
              <setting />
            </el-icon>
            <span>黑名单</span>
          </el-menu-item>-->

          <el-sub-menu index="7">
            <template #title>
              <el-icon>
                <operation />
              </el-icon>
              <span>综合设置</span>
            </template>
            <el-menu-item :index="`7-base`" @click="switchTo('miscSettings', 'base')">
              <span>基本设置</span>
            </el-menu-item>
            <el-menu-item :index="`7-group`" @click="switchTo('miscSettings', 'group')">
              <span>群组管理</span>
            </el-menu-item>
            <el-menu-item :index="`7-backup`" @click="switchTo('miscSettings', 'backup')">
              <span>备份</span>
            </el-menu-item>
          </el-sub-menu>

          <el-menu-item index="8" @click="switchTo('test')">
            <el-icon>
              <chat-line-round />
            </el-icon>
            <span>指令测试</span>
          </el-menu-item>

          <el-menu-item index="9" @click="switchTo('about')">
            <el-icon>
              <star />
            </el-icon>
            <span>关于</span>
          </el-menu-item>
        </el-menu>

        <div
          class="hidden-sm-and-up"
          style="position: absolute; bottom: 60px; color: #fff; font-size: small; margin-left: 1rem;"
        >
          <el-button @click="sideCollapse = !sideCollapse"></el-button>
        </div>
      </div>

      <!-- #545c64 -->
      <div
        style="background-color: #f3f5f7; flex: 1; text-align: left; height: calc(100vh - 4rem); overflow-y: auto;"
      >
        <!-- <div style="background-color: #f3f5f7; text-align: left; height: 100%;"> -->
        <div class="main-container" :class="[needh100 ? 'h100' : '']" ref="rightbox">
          <page-misc v-if="tabName === 'miscSettings'" :category="miscSettingsCategory" />
          <page-log v-if="tabName === 'log'" />
          <page-connect-info-items v-if="tabName === 'imConns'" />
          <page-custom-text v-if="tabName === 'customText'" :category="textCategory" />
          <page-custom-reply v-if="tabName === 'customReply'" />
          <page-test v-if="tabName === 'test'" />
          <page-about v-if="tabName === 'about'" />
        </div>
        <!-- </div> -->
      </div>
    </div>
  </div>

  <el-dialog
    v-model="showDialog"
    title=""
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    custom-class="the-dialog"
  >
    <h3>输入密码解锁</h3>
    <el-input v-model="password" type="password"></el-input>
    <el-button type="primary" style="padding: 0px 50px; margin-top: 1rem;" @click="doUnlock">确认</el-button>
  </el-dialog>
</template>

<script setup lang="ts">
import BaseHeader from "./components/layouts/BaseHeader.vue";
import HelloWorld from "./components/HelloWorld.vue";
import PageCustomText from "./components/PageCustomText.vue";
import PageConnectInfoItems from "./components/PageConnectInfoItems.vue";
import PageMisc from "./components/PageMisc.vue"
import PageCustomReply from "./components/PageCustomReply.vue"
import PageLog from "./components/PageLog.vue";
import PageAbout from "./components/PageAbout.vue"
import PageTest from "./components/PageTest.vue"
import { onBeforeMount, ref, watch, computed } from 'vue'
import { useStore } from './store'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  Location,
  Document,
  Menu as IconMenu,
  Setting,
  CirclePlusFilled,
  CircleClose,
  Ship,
  Star,
  Operation,
  ChatLineRound
} from '@element-plus/icons-vue'

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import { passwordHash } from "./utils";
import { delay } from "lodash-es";

dayjs.locale('zh-cn')

const store = useStore()
const password = ref('')

const showDialog = computed(() => {
  return !store.canAccess
})

const doUnlock = async () => {
  const hash = await passwordHash(store.salt, password.value)
  await store.signIn(hash)
  if (store.canAccess) {
    ElMessageBox.alert('欢迎回来，请开始使用。', '登录成功')
    password.value = ''
    checkPassword()
  } else {
    ElMessageBox.alert('错误的密码', '登录失败')
    password.value = ''
  }
}

const checkPassword = async () => {
  if (!await store.checkSecurity()) {
    ElMessageBox.alert('欢迎使用海豹核心。<br/>如果您的服务开启在公网，为了保证您的安全性，请前往<b>“综合设置->基本设置”</b>界面，设置<b>UI界面密码</b>。<br/>或切换为只有本机可访问。<br><b>如果您不了解上面在说什么，请务必设置一个密码</b>', '提示', { dangerouslyUseHTMLString: true })
  }
}

onBeforeMount(async () => {
  resetCollapse()
  store.getBaseInfo()
  store.getCustomText()

  if (store.canAccess) {
    checkPassword()
  }

  timerId = setInterval(() => {
    store.getBaseInfo()
  }, 5000) as any
})

let timerId: number

const handleOpen = (key: string, keyPath: string[]) => {
}
const handleClose = (key: string, keyPath: string[]) => {
}

const rightbox = ref(null)

const resetCollapse = () => {
  if (document.body.clientWidth <= 450) {
    if (!sideCollapse.value) {
      sideCollapse.value = true
      // 似乎不需要下面这段来重置flex了
      // setTimeout(() => {
      //   const el = rightbox.value as any
      //   const tmp = el.style.display;
      //   el.style.display = 'none';
      //   el.style.display = tmp;
      // }, 500)
    }
  } else {
    if (sideCollapse.value) {
      sideCollapse.value = false
    }
  }
}
window.addEventListener("resize", resetCollapse)

const sideCollapse = ref(false)
let tabName = ref("log")
let textCategory = ref("")
let miscSettingsCategory = ref("")

const needh100 = ref(false)

const switchTo = (tab: 'overview' | 'miscSettings' | 'log' | 'customText' | 'customReply' | 'imConns' | 'banList' | 'test' | 'about', name: string = '') => {
  tabName.value = tab
  if (tab === 'customText') {
    textCategory.value = name
  }
  if (tab === 'miscSettings') {
    miscSettingsCategory.value = name
  }
  needh100.value = ['test'].includes(tab)
}

let configCustom = {}
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

@media screen and (max-width: 700px) {
  .main-container {
    padding: 1rem;
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
</style>

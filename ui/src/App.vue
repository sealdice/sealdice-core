<template>
  <!-- <BaseHeader /> -->
  <!-- <HelloWorld msg="Hello Vue 3.0 + Element Plus + Vite" /> -->

  <div style="background: #545c64; height: 100%; display: flex; flex-direction: column; max-width: 900px; width: 100%; margin: 0 auto; position: relative;">
    <h3
      class="mb-2"
      style="color: #f8ffff; text-align: left; padding-left: 2em; font-weight: normal;"
    >SealDice</h3>

    <div style="position: absolute; top: 1rem; right: 10px; color: #fff; font-size: small;">{{ store.curDice.baseInfo.version }}</div>

    <div style="display: flex;">
      <div style="position: relative;">
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
          </el-menu-item> -->

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

            <el-menu-item :index="`5-${k}`" @click="switchTo('customText', k.toString())"  v-for="_, k in store.curDice.customTexts">
              <span>{{ k }}</span>
            </el-menu-item>
          </el-sub-menu>
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
          </el-menu-item> -->

          <el-sub-menu index="7">
            <template #title>
              <el-icon>
                <operation />
              </el-icon>
              <span>综合设置</span>
            </template>
            <el-menu-item :index="`7-base`" @click="switchTo('overview', 'base')">
              <span>基本设置</span>
            </el-menu-item>
            <!-- <el-menu-item :index="`7-group`" @click="switchTo('overview', 'group')">
              <span>群组信息</span>
            </el-menu-item> -->
            <!-- <el-menu-item :index="`7-backup`" @click="switchTo('overview', 'backup')">
              <span>备份</span>
            </el-menu-item> -->
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

        <div class="hidden-sm-and-up" style="position: absolute; bottom: 60px; color: #fff; font-size: small; margin-left: 1rem;">
          <el-button @click="sideCollapse = !sideCollapse"></el-button>
        </div>        
      </div>

      <!-- #545c64 -->
      <div style="background-color: #f3f5f7; flex: 1; text-align: left; height: calc(100vh - 60px); overflow-y: auto;">
        <!-- <div style="background-color: #f3f5f7; text-align: left; height: 100%;"> -->
        <div class="main-container" style="" ref="rightbox">
          <page-overview v-if="tabName === 'overview'" />
          <page-log v-if="tabName === 'log'" />
          <page-connect-info-items v-if="tabName === 'imConns'" />
          <page-custom-text v-if="tabName === 'customText'" :category="textCategory" />
          <page-test v-if="tabName === 'test'" />
          <page-about v-if="tabName === 'about'" />
        </div>
        <!-- </div> -->
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import BaseHeader from "./components/layouts/BaseHeader.vue";
import HelloWorld from "./components/HelloWorld.vue";
import PageCustomText from "./components/PageCustomText.vue";
import PageConnectInfoItems from "./components/PageConnectInfoItems.vue";
import PageOverview from "./components/PageOverview.vue"
import PageLog from "./components/PageLog.vue";
import PageAbout from "./components/PageAbout.vue"
import PageTest from "./components/PageTest.vue"
import { onBeforeMount, ref } from 'vue'
import { useStore } from './store'

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

dayjs.locale('zh-cn')

const store = useStore()

onBeforeMount(async () => {
  resetCollapse()
  // const canAccess = await store.trySignIn()

  store.getBaseInfo()
  store.getCustomText()

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
let tabName = ref("log");
let textCategory = ref("");

const switchTo = (tab: 'overview' | 'log' | 'customText' | 'imConns' | 'banList' | 'test' | 'about', name: string = '') => {
  tabName.value = tab
  textCategory.value = name
}

let configCustom = {}
</script>

<style>
html, body {
  height: 100%;
}

.main-container {
  padding: 2rem;
  position: relative;
  box-sizing: border-box;
  height: 100%;
}

@media screen and (max-width: 700px) {
  .main-container {
    padding: 1rem;
  }
}


#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  height: 100%;
  display: flex;
}

.element-plus-logo {
  width: 50%;
}
</style>

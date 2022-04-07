<template>
  <!-- <BaseHeader /> -->
  <!-- <HelloWorld msg="Hello Vue 3.0 + Element Plus + Vite" /> -->

  <div style="background: #545c64; height: 100%; max-width: 900px; width: 100%; margin: 0 auto;">
    <h3
      class="mb-2"
      style="color: #f8ffff; text-align: left; padding-left: 2em; font-weight: normal;"
    >SealDice</h3>

    <div style="height: calc(100% - 60px);">
      <div style="position: relative; float: left; height: 100%;">
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

          <el-menu-item index="7" @click="switchTo('overview')">
            <el-icon>
              <ship />
            </el-icon>
            <span>其他设置</span>
          </el-menu-item>
  
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
        <div style="position: absolute; bottom: 10px; color: #fff; font-size: small; margin-left: 1rem;">{{ store.curDice.baseInfo.version }}</div>
      </div>

      <div style="background-color: #545c64; height: 100%;">
        <div style="background-color: #f3f5f7; overflow-y: auto; text-align: left; height: 100%;">
          <div class="main-container">
            <page-overview v-if="tabName === 'overview'" />
            <page-log v-if="tabName === 'log'" />
            <page-connect-info-items v-if="tabName === 'imConns'" />
            <page-custom-text v-if="tabName === 'customText'" :category="textCategory" />
            <page-test v-if="tabName === 'test'" />
            <page-about v-if="tabName === 'about'" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import BaseHeader from "./components/layouts/BaseHeader.vue";
import HelloWorld from "./components/HelloWorld.vue";
import PageCustomText from "./components/PageCustomText.vue";
import PageConnectInfoItems from "./components/PageConnectInfoItems.vue";
import PageOverview from "./components/PageOverView.vue"
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
  ChatLineRound
} from '@element-plus/icons-vue'

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

dayjs.locale('zh-cn')

const store = useStore()

onBeforeMount(async () => {
  resetCollapse()
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

const resetCollapse = () => {
  if (document.body.clientWidth <= 450) {
    if (!sideCollapse.value) {
      sideCollapse.value = true
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
  height: 100%;
  box-sizing: border-box;
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

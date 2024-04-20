<template>
  <el-menu :unique-opened="true" style="border-right: 0;" active-text-color="#ffd04b"
           :background-color="props.type === 'dark' ? '#545c64' : undefined" default-active="2"
           :text-color="props.type === 'dark' ? '#fff' : undefined">
    <el-menu-item index="2" @click="switchTo('log')">
      <el-icon>
        <location/>
      </el-icon>
      <span>日志</span>
    </el-menu-item>

    <el-menu-item index="3" @click="switchTo('imConns')">
      <el-icon>
        <connection/>
      </el-icon>
      <span>账号设置</span>
    </el-menu-item>

    <el-sub-menu index="4">
      <template #title>
        <el-icon>
          <setting/>
        </el-icon>
        <span>自定义文案</span>
      </template>

      <el-menu-item :index="`5-${k}`" @click="switchTo('customText', k.toString())"
                    v-for="(_, k) in store.curDice.customTexts">
        <span>{{ k }}</span>
      </el-menu-item>
    </el-sub-menu>

    <el-sub-menu index="5">
      <template #title>
        <el-icon>
          <edit-pen/>
        </el-icon>
        <span>扩展功能</span>
      </template>
      <el-menu-item index="5-reply" @click="switchTo('mod', 'reply')">
        <!-- <el-icon><setting /></el-icon> -->
        <span>自定义回复</span>
      </el-menu-item>

      <el-menu-item :index="`5-deck`" @click="switchTo('mod', 'deck')">
        <span>牌堆管理</span>
      </el-menu-item>

      <el-menu-item :index="`5-story`" @click="switchTo('mod', 'story')">
        <span>跑团日志</span>
      </el-menu-item>

      <el-menu-item :index="`5-js`" @click="switchTo('mod', 'js')">
        <span>JS扩展</span>
      </el-menu-item>

      <el-menu-item :index="`5-helpdoc`" @click="switchTo('mod', 'helpdoc')">
        <span>帮助文档</span>
      </el-menu-item>

      <el-menu-item :index="`5-censor`" @click="switchTo('mod', 'censor')">
        <span>拦截管理</span>
      </el-menu-item>
    </el-sub-menu>

    <el-sub-menu index="7">
      <template #title>
        <el-icon>
          <operation/>
        </el-icon>
        <span>综合设置</span>
      </template>
      <el-menu-item :index="`7-base`" @click="switchTo('miscSettings', 'base')">
        <span>基本设置</span>
      </el-menu-item>
      <el-menu-item :index="`7-group`" @click="switchTo('miscSettings', 'group')">
        <span>群组管理</span>
      </el-menu-item>
      <el-menu-item :index="`7-ban`" @click="switchTo('miscSettings', 'ban')">
        <span>黑白名单</span>
      </el-menu-item>
      <el-menu-item :index="`7-backup`" @click="switchTo('miscSettings', 'backup')">
        <span>备份</span>
      </el-menu-item>
      <el-menu-item v-if="advancedConfigCounter >= 8" :index="`7-advanced`"
                    @click="switchTo('miscSettings', 'advanced')">
        <span>高级设置</span>
      </el-menu-item>
    </el-sub-menu>

    <el-menu-item index="8" @click="switchTo('test')">
      <el-icon>
        <chat-line-round/>
      </el-icon>
      <span>指令测试</span>
    </el-menu-item>

    <el-menu-item index="9" @click="switchTo('about')">
      <el-icon>
        <star/>
      </el-icon>
      <span>关于</span>
    </el-menu-item>
  </el-menu>
</template>

<script setup lang="ts">
import {
  Location,
  Connection,
  Setting,
  Star,
  Operation,
  ChatLineRound,
  EditPen
} from '@element-plus/icons-vue'

import {useStore} from "~/store";
import {ModelRef} from "vue";

const props = defineProps<{
  type: 'light' | 'dark'
}>()

const advancedConfigCounter: ModelRef<number> = defineModel('advancedConfigCounter', {default: 0})

const emit = defineEmits(['swithchTo'])

const store = useStore()

const switchTo = (page: string, subPage?: string) => {
  emit('swithchTo', page, subPage)
}
</script>

<style scoped lang="scss">

</style>
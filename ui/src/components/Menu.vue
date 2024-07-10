<template>
  <el-menu :unique-opened="true" style="border-right: 0;" :active-text-color="twColors.amber[300]"
           :background-color="props.type === 'dark' ? twColors.gray[600] : undefined"
           :text-color="props.type === 'dark' ? '#fff' : undefined"
           router :default-active="route.path">
    <el-menu-item index="/log">
      <el-icon>
        <location/>
      </el-icon>
      <span>日志</span>
    </el-menu-item>

    <el-menu-item index="/connect">
      <el-icon>
        <connection/>
      </el-icon>
      <span>账号设置</span>
    </el-menu-item>

    <el-sub-menu index="/custom-text">
      <template #title>
        <el-icon>
          <setting/>
        </el-icon>
        <span>自定义文案</span>
      </template>

      <el-menu-item :index="`/custom-text/${k}`"
                    :key="k" v-for="(_, k) in store.curDice.customTexts">
        <span>{{ k }}</span>
      </el-menu-item>
    </el-sub-menu>

    <el-sub-menu index="/mod">
      <template #title>
        <el-icon>
          <edit-pen/>
        </el-icon>
        <span>扩展功能</span>
      </template>

      <el-menu-item index="/mod/reply">
        <span>自定义回复</span>
      </el-menu-item>

      <el-menu-item index="/mod/deck">
        <span>牌堆管理</span>
      </el-menu-item>

      <el-menu-item index="/mod/story">
        <span>跑团日志</span>
      </el-menu-item>

      <el-menu-item index="/mod/js">
        <span>JS扩展</span>
      </el-menu-item>

      <el-menu-item index="/mod/helpdoc">
        <span>帮助文档</span>
      </el-menu-item>

      <el-menu-item index="/mod/censor">
        <span>拦截管理</span>
      </el-menu-item>
    </el-sub-menu>

    <el-sub-menu index="/misc">
      <template #title>
        <el-icon>
          <operation/>
        </el-icon>
        <span>综合设置</span>
      </template>

      <el-menu-item index="/misc/base-setting">
        <span>基本设置</span>
      </el-menu-item>

      <el-menu-item index="/misc/group">
        <span>群组管理</span>
      </el-menu-item>

      <el-menu-item index="/misc/ban">
        <span>黑白名单</span>
      </el-menu-item>

      <el-menu-item index="/misc/backup">
        <span>备份</span>
      </el-menu-item>

      <el-menu-item index="/misc/advanced-setting" v-if="advancedConfigCounter >= 8">
        <span>高级设置</span>
      </el-menu-item>
    </el-sub-menu>

    <el-sub-menu index="/tool">
      <template #title>
        <el-icon>
          <tools/>
        </el-icon>
        <span>辅助工具</span>
      </template>

      <el-menu-item index="/tool/test">
        <span>指令测试</span>
      </el-menu-item>

      <el-menu-item index="/tool/resource">
        <span>资源管理</span>
      </el-menu-item>
    </el-sub-menu>

    <el-menu-item index="/about">
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
  Tools,
  EditPen
} from '@element-plus/icons-vue'
import { useStore } from "~/store";
import type { ModelRef } from "vue";
import resolveConfig from 'tailwindcss/resolveConfig'
import tailwindConfig from '../../tailwind.config'

const twColors = resolveConfig(tailwindConfig).theme.colors

const props = defineProps<{
  type: 'light' | 'dark'
}>()

const advancedConfigCounter: ModelRef<number> = defineModel('advancedConfigCounter', { default: 0 })

const store = useStore()

const route = useRoute()
</script>

<style scoped lang="css">

</style>
<script setup lang="ts">
import { NSwitch, NGrid, NGridItem } from 'naive-ui';
import { useStore } from '~/store';
const option_store = useStore().exportOptions
import { useDark, useToggle } from '@vueuse/core'

const isDark = useDark()
interface Option {
    label: string
    desc: string
    key: keyof typeof option_store
}

const list: Option[] = [
    {
        label: "骰子指令过滤",
        desc: "开启后，不显示pc指令，正常显示指令结果",
        key: 'commandHide',
    },
    {
        label: "表情包和图片过滤",
        desc: "开启后，文本内所有的表情包和图片将被豹豹藏起来不显示",
        key: 'imageHide',
    },
    {
        label: "场外发言过滤",
        desc: "开启后，所有以(和（为开头的发言将被豹豹吃掉不显示",
        key: 'offTopicHide',
    },
    {
        label: "时间显示过滤",
        desc: "开启后，日期和时间会被豹豹丢入海里不显示",
        key: 'timeHide',
    },
    {
        label: "隐藏帐号",
        desc: "开启后，IM 平台账号（如 QQ 号）将在导出结果中不显示",
        key: 'userIdHide',
    },
    {
        label: "隐藏年月日",
        desc: "开启后，导出结果的日期将只显示几点几分(如果可能)",
        key: 'yearHide',
    },
    {
        label: "首行缩进",
        desc: "开启后，缩进将以名字为基准进行对齐",
        key: 'textIndentAll',
    },
]
</script>

<template>
    <n-grid cols="1 640:2" :x-gap="6" :y-gap="12" class="p-5">
        <n-grid-item v-for="opt in list">
            <n-switch v-model:value="option_store[opt.key]"></n-switch>
            <label class="ml-5">{{ opt.label }}</label>
            <p class="mt-2">{{ opt.desc }}</p>
        </n-grid-item>
        <n-grid-item>
            <n-switch v-model:value="isDark" @change="useToggle"></n-switch>
            <label class="ml-5">深色模式</label>
            <p class="mt-2">启用深色模式，适合夜间使用</p>
        </n-grid-item>
    </n-grid>
</template>
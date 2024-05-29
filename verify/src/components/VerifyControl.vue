<script setup lang="ts">
import type { CSSProperties } from 'vue'
import { useThemeVars } from 'naive-ui'

const themeVars = useThemeVars()

const isDark = useDark({
  valueDark: 'dark',
  valueLight: 'light'
})
const toggleDark = useToggle(isDark)

const darkBtnStyle = ({ focused, checked }: {
  focused: boolean
  checked: boolean
}) => {
  const style: CSSProperties = {}
  if (checked) {
    style.background = themeVars.value.infoColor
    if (focused) {
      style.boxShadow = '0 0 0 2px ' + themeVars.value.infoColorHover
    }
  } else {
    style.background = themeVars.value.successColor
    if (focused) {
      style.boxShadow = '0 0 0 2px ' + themeVars.value.successColorHover
    }
  }
  return style
}
</script>

<template>
  <aside class="flex flex-row justify-end">
    <n-switch v-model:value="isDark" @on-update:value="toggleDark"
              size="large" :rail-style="darkBtnStyle">
      <template #checked-icon>
        <n-icon>
          <i-carbon-moon />
        </n-icon>
      </template>
      <template #unchecked-icon>
        <n-icon>
          <i-carbon-sun />
        </n-icon>
      </template>
    </n-switch>
  </aside>
</template>

<style scoped>

</style>
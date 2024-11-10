<script setup lang="ts">
import Main from "~/Main.vue";
import { darkTheme, lightTheme } from "naive-ui";

import { useDark } from '@vueuse/core'

const isDark = useDark({ disableTransition: false })
</script>

<template>
  <n-config-provider :theme="isDark ? darkTheme : lightTheme">
    <n-message-provider>
      <n-modal-provider>
        <n-notification-provider>
          <Main/>
        </n-notification-provider>
      </n-modal-provider>
    </n-message-provider>
  </n-config-provider>
</template>
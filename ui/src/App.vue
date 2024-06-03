<script setup lang="ts">
import {useStore} from "~/store";
import Main from "~/Main.vue";

const store = useStore()

const testMode = ref<boolean>(true)

onMounted(async () => {
  const info = await store.getPreInfo()
  testMode.value = info.testMode
})
</script>

<template>
  <div id="root" class="w-full h-full">
    <el-watermark v-if="testMode"
                  class="mx-auto max-w-[950px]"
                  content="仅用于展示，修改无效"
                  :font="{fontSize: 24}"
                  :gap="[100, 32]">
      <Main/>
    </el-watermark>
    <Main v-else/>
  </div>
</template>

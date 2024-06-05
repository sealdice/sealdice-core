<script setup lang="ts">
import {useStore} from "~/store";
import Main from "~/Main.vue";

const store = useStore()

const testMode = ref<boolean>(false)

onMounted(async () => {
  const info = await store.getPreInfo()
  testMode.value = info.testMode
})
</script>

<template>
  <div class="mx-auto w-full max-w-[950px] h-full">
    <el-watermark :content="testMode ? '仅用于展示，修改无效' : ''"
                  :font="{fontSize: 24}"
                  :gap="[100, 32]">
      <Main />
    </el-watermark>
  </div>
</template>

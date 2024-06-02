<script setup lang="ts">
import {useStore} from "~/store";
import Main from "~/Main.vue";

const store = useStore()

onBeforeMount(async () => {
  await store.getBaseInfo()
})
</script>

<template>
  <div id="root" class="w-full h-full">
    <el-watermark v-if="store.diceServers[0].baseInfo.containerMode"
                  class="mx-auto max-w-[950px]"
                  content="仅用于展示，修改无效"
                  :font="{fontSize: 24}"
                  :gap="[100, 32]">
      <Main/>
    </el-watermark>
    <Main v-else/>
  </div>
</template>

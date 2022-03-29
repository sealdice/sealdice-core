<template>
  <div>111</div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount } from 'vue';
import { useStore } from '~/store';
const store = useStore()

onBeforeMount(async () => {
  await store.logFetchAndClear()
})

onBeforeUnmount(() => {
  // clearInterval(timerId)
})
</script>

<style scoped lang="scss">
.about {
  background-color: #fff;
  padding: 2rem;
  line-height: 2rem;
  text-align: left;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04)
}

.top-item {
  flex: 1 0 50%;
  flex-grow: 0;
}
</style>

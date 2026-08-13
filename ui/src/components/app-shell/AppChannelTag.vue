<template>
  <n-tag :bordered="false" :type="channelTagType" size="small" class="channel-tag">
    {{ channelText }}
  </n-tag>
</template>

<script setup lang="ts">
import { useBaseOverview } from '@/features/base/useBaseOverview';

const { channelText, channelTagType } = useBaseOverview();
</script>

<style scoped>
.channel-tag {
  flex: 0 0 auto;
  font-size: 0.75rem;
}
</style>

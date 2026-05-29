<template>
  <main class="reply-page">
    <CustomReplyEditor />
  </main>
</template>

<script setup lang="ts">
import CustomReplyEditor from '@/components/custom-reply/CustomReplyEditor.vue';
</script>

<style scoped>
.reply-page {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
}
</style>

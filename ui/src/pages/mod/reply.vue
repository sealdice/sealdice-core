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
  flex-direction: column;
  gap: 1rem;
}
</style>

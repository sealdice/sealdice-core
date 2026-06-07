<template>
  <i-ep-house v-if="name === 'home'" />
  <i-ep-connection v-else-if="name === 'connection'" />
  <i-ep-edit-pen v-else-if="name === 'edit'" />
  <i-ep-chat-line-round v-else-if="name === 'reply'" />
  <i-ep-collection v-else-if="name === 'deck'" />
  <i-ep-notebook v-else-if="name === 'story'" />
  <i-ep-cpu v-else-if="name === 'js'" />
  <i-ep-document v-else-if="name === 'helpdoc'" />
  <i-ep-warning v-else-if="name === 'censor'" />
  <i-ep-operation v-else-if="name === 'operation'" />
  <i-ep-setting v-else-if="name === 'setting' || name === 'base-setting'" />
  <i-ep-user-filled v-else-if="name === 'group'" />
  <i-ep-circle-close-filled v-else-if="name === 'ban'" />
  <i-ep-grid v-else-if="name === 'dice'" />
  <i-ep-folder-opened v-else-if="name === 'backup'" />
  <i-ep-tools v-else-if="name === 'advanced-setting' || name === 'tools'" />
  <i-ep-magic-stick v-else-if="name === 'test'" />
  <i-ep-folder v-else-if="name === 'resource'" />
  <i-ep-star-filled v-else-if="name === 'star'" />
  <i-ep-menu v-else />
</template>

<script setup lang="ts">
defineProps<{
  name?: string;
}>();
</script>

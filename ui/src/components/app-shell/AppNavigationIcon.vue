<template>
  <i-tabler-home v-if="name === 'home'" />
  <i-tabler-plug-connected v-else-if="name === 'connection'" />
  <i-tabler-message-cog v-else-if="name === 'setting'" />
  <i-tabler-puzzle v-else-if="name === 'edit'" />
  <i-tabler-messages v-else-if="name === 'reply'" />
  <i-tabler-cards v-else-if="name === 'deck'" />
  <i-tabler-packages v-else-if="name === 'package'" />
  <i-tabler-book-2 v-else-if="name === 'story'" />
  <i-tabler-file-code v-else-if="name === 'js'" />
  <i-tabler-books v-else-if="name === 'helpdoc'" />
  <i-tabler-shield-exclamation v-else-if="name === 'censor'" />
  <i-tabler-adjustments-horizontal v-else-if="name === 'operation'" />
  <i-tabler-settings v-else-if="name === 'base-setting'" />
  <i-tabler-users-group v-else-if="name === 'group'" />
  <i-tabler-user-cancel v-else-if="name === 'ban'" />
  <i-tabler-dice-5 v-else-if="name === 'dice'" />
  <i-tabler-archive v-else-if="name === 'backup'" />
  <i-tabler-settings-exclamation v-else-if="name === 'advanced-setting'" />
  <i-tabler-tool v-else-if="name === 'tools'" />
  <i-tabler-command v-else-if="name === 'test'" />
  <i-tabler-photo v-else-if="name === 'resource'" />
  <i-tabler-activity v-else-if="name === 'profile'" />
  <i-tabler-info-circle v-else-if="name === 'star'" />
  <i-tabler-menu-2 v-else />
</template>

<script setup lang="ts">
defineProps<{
  name?: string;
}>();
</script>

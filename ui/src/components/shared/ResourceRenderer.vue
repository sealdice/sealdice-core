<script setup lang="ts">
withDefaults(
  defineProps<{
    type: string;
    name?: string;
    src?: string;
  }>(),
  {
    name: '',
    src: '',
  },
);
</script>

<template>
  <template v-if="type === 'image'">
    <n-image :src="src" :alt="name" object-fit="contain" lazy>
      <template #placeholder>
        <div class="flex h-40 items-center justify-center">
          <n-icon size="32" color="var(--n-text-color-3)">
            <svg viewBox="0 0 24 24"><rect x="3" y="3" width="18" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="2"/><circle cx="8.5" cy="8.5" r="1.5" fill="currentColor"/><polyline points="21 15 16 10 5 21" fill="none" stroke="currentColor" stroke-width="2"/></svg>
          </n-icon>
        </div>
      </template>
      <template #error>
        <div class="flex h-40 items-center justify-center">
          <n-icon size="32" color="var(--n-text-color-3)">
            <svg viewBox="0 0 24 24"><rect x="3" y="3" width="18" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="2"/><circle cx="8.5" cy="8.5" r="1.5" fill="currentColor"/><polyline points="21 15 16 10 5 21" fill="none" stroke="currentColor" stroke-width="2"/></svg>
          </n-icon>
        </div>
      </template>
    </n-image>
  </template>

  <template v-else-if="type === 'audio'">
    <n-icon size="24">
      <svg viewBox="0 0 24 24"><rect x="2" y="6" width="6" height="12" rx="1" fill="currentColor"/><rect x="10" y="3" width="6" height="18" rx="1" fill="currentColor" opacity="0.7"/><rect x="18" y="7" width="6" height="10" rx="1" fill="currentColor" opacity="0.5"/></svg>
    </n-icon>
  </template>

  <template v-else-if="type === 'video'">
    <n-icon size="24">
      <svg viewBox="0 0 24 24"><polygon points="5 3 19 12 5 21 5 3" fill="currentColor"/></svg>
    </n-icon>
  </template>

  <template v-else>
    未知格式
  </template>
</template>

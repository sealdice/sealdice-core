<template>
  <div class="sd-list-empty-state" role="status">
    <n-empty :description="description" size="large">
      <template v-if="$slots.icon" #icon>
        <slot name="icon" />
      </template>
      <template v-if="$slots.default" #extra>
        <div class="sd-list-empty-state__extra">
          <slot />
        </div>
      </template>
    </n-empty>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  description: string;
}>();

defineSlots<{
  default?(): unknown;
  icon?(): unknown;
}>();
</script>

<style scoped>
.sd-list-empty-state {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 12rem;
  box-sizing: border-box;
  align-items: center;
  justify-content: center;
  padding: var(--sd-space-lg) var(--sd-space-md);
}

.sd-list-empty-state__extra {
  display: flex;
  max-width: 36rem;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: var(--sd-space-xs);
  text-align: center;
}
</style>

<template>
  <section class="feature-disabled-state" role="status">
    <n-empty :description="title" size="large">
      <template #icon>
        <n-icon size="42" aria-hidden="true">
          <i-tabler-power />
        </n-icon>
      </template>

      <template #extra>
        <n-text depth="3" class="feature-disabled-state__description">
          {{ description }}
        </n-text>
      </template>
    </n-empty>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description: string;
}>();
</script>

<style scoped>
.feature-disabled-state {
  display: flex;
  min-height: 260px;
  align-items: center;
  justify-content: center;
  padding: var(--sd-space-2xl) var(--sd-space-md);
}

.feature-disabled-state__description {
  display: block;
  max-width: 30rem;
  text-align: center;
}
</style>

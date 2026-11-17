<template>
  <nav v-if="items.length > 1" class="workspace-trail" aria-label="当前位置">
    <n-breadcrumb>
      <n-breadcrumb-item v-for="(item, index) in items" :key="index" :clickable="Boolean(item.to)">
        <RouterLink v-if="item.to" :to="item.to">{{ item.label }}</RouterLink>
        <span v-else :aria-current="index === items.length - 1 ? 'page' : undefined">
          {{ item.label }}
        </span>
      </n-breadcrumb-item>
    </n-breadcrumb>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import { appNavigation } from '@/router/navigation';
import { buildBreadcrumbItems } from '@/router/navigationModel';

const route = useRoute();
const items = computed(() =>
  buildBreadcrumbItems(appNavigation, route.path, String(route.meta.title ?? '当前页面'))
);
</script>

<style scoped>
.workspace-trail {
  margin-bottom: var(--sd-space-xs);
  color: var(--sd-text-secondary);
}

.workspace-trail :deep(.n-breadcrumb) {
  font-size: 0.75rem;
}
</style>

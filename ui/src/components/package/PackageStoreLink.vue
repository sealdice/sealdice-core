<template>
  <RouterLink v-slot="{ href, navigate }" :to="PACKAGE_STORE_ROUTE" custom>
    <n-button tag="a" type="primary" secondary :href="href" @click="navigate">
      <template #icon>
        <n-icon><i-tabler-building-store /></n-icon>
      </template>
      <slot>扩展商店</slot>
    </n-button>
  </RouterLink>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router';
import { PACKAGE_STORE_ROUTE } from '@/features/package/navigation';
</script>

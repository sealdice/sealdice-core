<template>
  <main class="about-page">
    <AboutHero :summary="summary" :links="ABOUT_LINKS" :loading="loading" />
    <AboutCredits :sections="ABOUT_CREDIT_SECTIONS" />
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import AboutCredits from '@/components/about/AboutCredits.vue';
import AboutHero from '@/components/about/AboutHero.vue';
import {
  ABOUT_CREDIT_SECTIONS,
  ABOUT_LINKS,
  getAboutOverviewSummary,
} from '@/features/about/viewModel';
import { useBaseOverview } from '@/features/base/useBaseOverview';

const { overview, overviewQuery } = useBaseOverview();
const summary = computed(() => getAboutOverviewSummary(overview.value));
const loading = computed(() => overviewQuery.isFetching.value && !overview.value);
</script>

<style scoped>
.about-page {
  display: grid;
  gap: 22px;
  min-width: 0;
}
</style>

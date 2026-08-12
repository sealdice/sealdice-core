<template>
  <a
    v-if="href"
    class="about-contributor-card"
    :class="{ 'about-contributor-card--muted': props.contributor.muted }"
    :href="href"
    target="_blank"
    rel="noopener noreferrer"
  >
    <n-avatar round :size="40" :src="avatarSrc" :fallback-src="sealImage" />
    <span class="about-contributor-card__body">
      <span class="about-contributor-card__name">{{ props.contributor.username }}</span>
      <span v-if="props.contributor.info" class="about-contributor-card__info">
        {{ props.contributor.info }}
      </span>
    </span>
  </a>

  <div v-else class="about-contributor-card about-contributor-card--plain">
    <n-avatar round :size="40" :src="avatarSrc" :fallback-src="sealImage" />
    <span class="about-contributor-card__body">
      <span class="about-contributor-card__name">{{ props.contributor.username }}</span>
      <span v-if="props.contributor.info" class="about-contributor-card__info">
        {{ props.contributor.info }}
      </span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import sealImage from '@/assets/seal.png';
import {
  buildAvatarUrl,
  buildContributorHref,
  type AboutContributor,
} from '@/features/about/viewModel';

const props = defineProps<{
  contributor: AboutContributor;
}>();

const href = computed(() => buildContributorHref(props.contributor));
const avatarSrc = computed(() =>
  props.contributor.onlyName ? sealImage : buildAvatarUrl(props.contributor)
);
</script>

<style scoped>
.about-contributor-card {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated-soft);
  color: var(--sd-text-primary);
  text-decoration: none;
  transition:
    border-color var(--sd-transition-fast),
    background-color var(--sd-transition-fast);
}

.about-contributor-card:hover {
  border-color: var(--sd-primary-border);
  background: var(--sd-primary-soft);
}

.about-contributor-card--plain:hover {
  border-color: var(--sd-border-soft);
  background: var(--sd-bg-elevated-soft);
}

.about-contributor-card--muted {
  color: var(--sd-text-muted);
}

.about-contributor-card--muted :deep(.n-avatar) {
  filter: grayscale(1);
  opacity: 0.6;
}

.about-contributor-card__body {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.about-contributor-card__name {
  color: var(--sd-text-primary);
  font-weight: 700;
  line-height: 1.25;
  overflow-wrap: anywhere;
  white-space: normal;
}

.about-contributor-card__info {
  color: var(--sd-text-muted);
  font-size: 12px;
  line-height: 1.4;
  overflow-wrap: anywhere;
  white-space: normal;
}
</style>

<template>
  <a
    v-if="href"
    class="about-contributor-card"
    :href="href"
    target="_blank"
    rel="noopener noreferrer"
  >
    <n-avatar round :size="44" :src="avatarSrc" :fallback-src="sealImage" />
    <span class="about-contributor-card__body">
      <span class="about-contributor-card__name">{{ props.contributor.username }}</span>
      <span v-if="props.contributor.info" class="about-contributor-card__info">
        {{ props.contributor.info }}
      </span>
    </span>
  </a>

  <div v-else class="about-contributor-card about-contributor-card--plain">
    <n-avatar round :size="44" :src="avatarSrc" :fallback-src="sealImage" />
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
const avatarSrc = computed(() => (props.contributor.onlyName ? sealImage : buildAvatarUrl(props.contributor)));
</script>

<style scoped>
.about-contributor-card {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--sd-border-soft);
  border-radius: 16px;
  background: var(--sd-bg-elevated-tint);
  color: var(--sd-text-primary);
  text-decoration: none;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.about-contributor-card:hover {
  border-color: var(--sd-primary);
  border-color: color-mix(in srgb, var(--sd-primary), transparent 62%);
  box-shadow: 0 10px 26px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.about-contributor-card--plain:hover {
  border-color: var(--sd-border-soft);
  box-shadow: none;
  transform: none;
}

.about-contributor-card__body {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.about-contributor-card__name {
  overflow: hidden;
  color: var(--sd-text-primary);
  font-weight: 700;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.about-contributor-card__info {
  overflow: hidden;
  color: var(--sd-text-muted);
  font-size: 12px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

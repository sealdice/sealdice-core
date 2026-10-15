<template>
  <main class="about-page sd-page">
    <AboutHero :summary="summary" :links="ABOUT_LINKS" :loading="loading" />
    <section class="about-credits">
      <h2>感谢</h2>

      <div class="about-credits__sections">
        <section
          v-for="section in creditSections"
          :key="section.title"
          class="about-credits__section"
        >
          <h3>{{ section.title }}</h3>

          <div v-if="section.contributors?.length" class="about-credits__contributors">
            <component
              :is="contributor.href ? 'a' : 'div'"
              v-for="contributor in section.contributors"
              :key="`${section.title}:${contributor.username}`"
              class="about-contributor-card"
              :class="{
                'about-contributor-card--plain': !contributor.href,
                'about-contributor-card--muted': contributor.href && contributor.muted,
              }"
              :href="contributor.href || undefined"
              :target="contributor.href ? '_blank' : undefined"
              :rel="contributor.href ? 'noopener noreferrer' : undefined"
            >
              <n-avatar round :size="40" :src="contributor.avatarSrc" :fallback-src="sealImage" />
              <span class="about-contributor-card__body">
                <span class="about-contributor-card__name">{{ contributor.username }}</span>
                <span v-if="contributor.info" class="about-contributor-card__info">{{
                  contributor.info
                }}</span>
              </span>
            </component>
          </div>

          <div v-if="section.lines?.length" class="about-credits__lines">
            <p
              v-for="line in section.lines"
              :key="`${section.title}:${line.text}:${line.linkText ?? ''}`"
              class="about-credits__line"
            >
              <span>{{ line.text }}</span>
              <n-button
                v-if="line.href && line.linkText"
                text
                tag="a"
                target="_blank"
                rel="noopener noreferrer"
                type="primary"
                :href="line.href"
              >
                {{ line.linkText }}
              </n-button>
              <span v-if="line.tail">{{ line.tail }}</span>
            </p>
          </div>
        </section>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import sealImage from '@/assets/seal.png';
import { buildAvatarUrl, buildContributorHref } from '@/features/about/viewModel';
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
const creditSections = computed(() =>
  ABOUT_CREDIT_SECTIONS.map(section => ({
    ...section,
    contributors: section.contributors?.map(contributor => ({
      ...contributor,
      href: buildContributorHref(contributor),
      avatarSrc: contributor.onlyName ? sealImage : buildAvatarUrl(contributor),
    })),
  }))
);
</script>

<style scoped>
.about-page {
  display: grid;
  gap: 28px;
  min-width: 0;
  padding-bottom: var(--sd-space-xl);
}
.about-credits {
  display: grid;
  gap: 14px;
}

.about-credits > h2 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 26px;
  line-height: 1.2;
}

.about-credits__sections {
  overflow: hidden;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-lg);
  background: var(--sd-bg-elevated);
}

.about-credits__section {
  display: grid;
  gap: 14px;
  padding: 20px;
  border-top: 1px solid var(--sd-border-soft);
}

.about-credits__section:first-child {
  border-top: 0;
  background: linear-gradient(90deg, var(--sd-primary-soft), transparent 72%);
}

.about-credits__section h3 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 16px;
  font-weight: 750;
  line-height: 1.3;
}

.about-credits__contributors {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
  gap: 8px;
}

.about-credits__lines {
  display: grid;
  gap: 8px;
}

.about-credits__line {
  margin: 0;
  color: var(--sd-text-secondary);
  line-height: 1.7;
}

@media screen and (max-width: 639.9px) {
  .about-credits__section {
    padding: 16px;
  }

  .about-credits__contributors {
    grid-template-columns: minmax(0, 1fr);
  }
}
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

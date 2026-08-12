<template>
  <section class="about-hero">
    <div class="about-hero__mascot" aria-hidden="true">
      <div class="about-hero__mascot-frame">
        <img :src="sealImage" alt="" />
      </div>
    </div>

    <div class="about-hero__content">
      <header class="about-hero__identity">
        <h1>{{ props.summary.appName }}</h1>
      </header>

      <div class="about-hero__stats" :class="{ 'about-hero__stats--loading': props.loading }">
        <div class="about-hero__stat about-hero__stat--primary">
          <span>当前版本</span>
          <strong>{{ props.summary.versionText }}</strong>
        </div>
        <div class="about-hero__stat">
          <span>最新版本</span>
          <strong>{{ props.summary.latestVersionText }}</strong>
        </div>
        <div class="about-hero__stat">
          <span>发布通道</span>
          <strong>{{ props.summary.channelText }}</strong>
        </div>
        <div class="about-hero__stat">
          <span>运行环境</span>
          <strong>{{ props.summary.runtimeText }}</strong>
        </div>
        <div class="about-hero__stat">
          <span>运行模式</span>
          <strong>{{ props.summary.modeText }}</strong>
        </div>
        <div class="about-hero__stat">
          <span>运行时间</span>
          <strong>{{ props.summary.uptimeText }}</strong>
        </div>
      </div>

      <n-alert
        v-if="props.summary.hasNewVersion"
        type="warning"
        :bordered="false"
        class="about-hero__alert"
      >
        检测到新版本 {{ props.summary.latestVersionText }}。
        <span v-if="props.summary.latestNote">{{ props.summary.latestNote }}</span>
      </n-alert>

      <nav class="about-hero__links" aria-label="SealDice 项目链接">
        <a
          v-for="link in props.links"
          :key="link.href"
          class="about-hero__link"
          :href="link.href"
          target="_blank"
          rel="noopener noreferrer"
        >
          <span class="about-hero__link-icon">
            <n-icon :size="20">
              <i-tabler-world v-if="link.icon === 'website'" />
              <i-tabler-book v-else-if="link.icon === 'manual'" />
              <i-tabler-heart v-else-if="link.icon === 'support'" />
              <i-tabler-brand-github v-else />
            </n-icon>
          </span>
          <span class="about-hero__link-copy">
            <strong>{{ link.label }}</strong>
            <span>{{ link.description }}</span>
          </span>
          <n-icon class="about-hero__link-arrow" :size="18">
            <i-tabler-external-link />
          </n-icon>
        </a>
      </nav>
    </div>
  </section>
</template>

<script setup lang="ts">
import sealImage from '@/assets/seal.png';
import type { AboutLink, AboutOverviewSummary } from '@/features/about/viewModel';

const props = defineProps<{
  summary: AboutOverviewSummary;
  links: AboutLink[];
  loading?: boolean;
}>();
</script>

<style scoped>
.about-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
  gap: clamp(24px, 3vw, 38px);
  overflow: hidden;
  padding: clamp(24px, 3vw, 36px);
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-lg);
  background:
    radial-gradient(
      circle at 12% 16%,
      color-mix(in srgb, var(--sd-primary) 18%, transparent),
      transparent 33%
    ),
    radial-gradient(
      circle at 96% 8%,
      color-mix(in srgb, var(--sd-primary) 8%, transparent),
      transparent 26%
    ),
    linear-gradient(142deg, var(--sd-bg-elevated), var(--sd-bg-elevated-soft));
  isolation: isolate;
}

.about-hero::after {
  position: absolute;
  z-index: -1;
  width: 280px;
  height: 280px;
  border: 1px solid color-mix(in srgb, var(--sd-primary) 12%, transparent);
  border-radius: 50%;
  content: '';
  inset: -150px -70px auto auto;
}

.about-hero__mascot {
  display: grid;
  min-height: 286px;
  place-items: center;
}

.about-hero__mascot-frame {
  position: relative;
  display: grid;
  width: min(248px, 100%);
  aspect-ratio: 1;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--sd-primary-border);
  border-radius: var(--sd-radius-lg);
  background:
    radial-gradient(circle at 50% 34%, var(--sd-bg-elevated) 0 23%, transparent 58%),
    linear-gradient(155deg, var(--sd-primary-soft-strong), var(--sd-primary-soft));
  animation: about-mascot-enter 420ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.about-hero__mascot-frame::before {
  position: absolute;
  width: 72%;
  aspect-ratio: 1;
  border: 1px solid color-mix(in srgb, var(--sd-primary) 18%, transparent);
  border-radius: 50%;
  content: '';
}

.about-hero__mascot img {
  z-index: 1;
  width: min(230px, 94%);
  filter: drop-shadow(0 14px 18px color-mix(in srgb, var(--sd-text-primary) 16%, transparent));
}

.about-hero__content {
  display: grid;
  min-width: 0;
  align-content: center;
  gap: 18px;
  animation: about-content-enter 360ms 60ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.about-hero__identity {
  display: grid;
  gap: 10px;
}

.about-hero h1 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: clamp(38px, 5vw, 56px);
  font-weight: 850;
  letter-spacing: -0.045em;
  line-height: 1;
}

.about-hero__stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: color-mix(in srgb, var(--sd-bg-elevated) 76%, transparent);
  transition: opacity var(--sd-transition-base);
}

.about-hero__stats--loading {
  opacity: 0.64;
}

.about-hero__stat {
  display: grid;
  min-width: 0;
  align-content: center;
  gap: 4px;
  padding: 12px 14px;
  border-left: 1px solid var(--sd-border-soft);
}

.about-hero__stat:first-child {
  border-left: 0;
}

.about-hero__stat:nth-child(4) {
  border-left: 0;
}

.about-hero__stat:nth-child(n + 4) {
  border-top: 1px solid var(--sd-border-soft);
}

.about-hero__stat span {
  color: var(--sd-text-muted);
  font-size: 12px;
}

.about-hero__stat strong {
  min-width: 0;
  color: var(--sd-text-primary);
  font-size: 15px;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.about-hero__stat--primary strong {
  color: var(--sd-primary);
  font-size: 17px;
}

.about-hero__alert {
  max-width: 760px;
}

.about-hero__links {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.about-hero__link {
  display: grid;
  min-width: 0;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 13px 14px;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: color-mix(in srgb, var(--sd-bg-elevated) 72%, transparent);
  color: var(--sd-text-primary);
  text-decoration: none;
  transition:
    border-color var(--sd-transition-fast),
    background-color var(--sd-transition-fast);
}

.about-hero__link:hover {
  border-color: var(--sd-primary-border);
  background: var(--sd-primary-soft);
}

.about-hero__link-icon {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: var(--sd-radius-sm);
  background: var(--sd-primary-soft-strong);
  color: var(--sd-primary);
}

.about-hero__link-copy {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.about-hero__link-copy strong {
  font-size: 14px;
}

.about-hero__link-copy span {
  color: var(--sd-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.about-hero__link-arrow {
  color: var(--sd-text-muted);
  transition:
    color var(--sd-transition-fast),
    transform var(--sd-transition-fast);
}

.about-hero__link:hover .about-hero__link-arrow {
  color: var(--sd-primary);
  transform: translate(2px, -2px);
}

@keyframes about-mascot-enter {
  from {
    opacity: 0;
    transform: scale(0.97);
  }

  to {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes about-content-enter {
  from {
    opacity: 0;
    transform: translateY(6px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media screen and (max-width: 1099.9px) {
  .about-hero {
    grid-template-columns: minmax(180px, 220px) minmax(0, 1fr);
  }

  .about-hero__mascot {
    min-height: 232px;
  }

  .about-hero__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .about-hero__stat:nth-child(odd) {
    border-left: 0;
  }

  .about-hero__stat:nth-child(4) {
    border-left: 1px solid var(--sd-border-soft);
  }

  .about-hero__stat:nth-child(n + 3) {
    border-top: 1px solid var(--sd-border-soft);
  }
}

@media screen and (max-width: 839.9px) {
  .about-hero {
    grid-template-columns: 1fr;
  }

  .about-hero__mascot {
    min-height: 0;
  }

  .about-hero__mascot-frame {
    width: min(210px, 68vw);
  }
}

@media screen and (max-width: 639.9px) {
  .about-hero {
    padding: 18px;
  }

  .about-hero__stats,
  .about-hero__links {
    grid-template-columns: minmax(0, 1fr);
  }

  .about-hero__stat,
  .about-hero__stat:nth-child(4) {
    border-top: 1px solid var(--sd-border-soft);
    border-left: 0;
  }

  .about-hero__stat:first-child {
    border-top: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .about-hero__mascot-frame,
  .about-hero__content {
    animation: none;
  }
}
</style>

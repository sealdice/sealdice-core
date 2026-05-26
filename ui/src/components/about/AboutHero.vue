<template>
  <section class="about-hero">
    <div class="about-hero__mascot">
      <img :src="sealImage" alt="SealDice mascot" />
    </div>

    <div class="about-hero__content">
      <div class="about-hero__title-row">
        <n-tag :bordered="false" type="info">SealDice UI</n-tag>
        <n-tag v-if="props.summary.containerMode" :bordered="false" type="warning">
          容器模式
        </n-tag>
        <n-tag v-if="props.summary.justForTest" :bordered="false" type="warning">
          展示模式
        </n-tag>
      </div>

      <h1>{{ props.summary.appName }}</h1>
      <p class="about-hero__summary">
        海豹核心管理后台。这里保留版本信息、项目链接和旧版关于页的社区鸣谢名单。
      </p>

      <div class="about-hero__stats" :class="{ 'about-hero__stats--loading': props.loading }">
        <div class="about-hero__stat">
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

      <div class="about-hero__links">
        <a
          v-for="link in props.links"
          :key="link.href"
          class="about-hero__link"
          :href="link.href"
          target="_blank"
          rel="noopener noreferrer"
        >
          <span class="about-hero__link-label">{{ link.label }}</span>
          <span class="about-hero__link-description">{{ link.description }}</span>
          <span class="about-hero__link-url">{{ link.href }}</span>
        </a>
      </div>
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
  grid-template-columns: minmax(180px, 280px) minmax(0, 1fr);
  gap: 24px;
  overflow: hidden;
  padding: clamp(20px, 4vw, 34px);
  border: 1px solid var(--sd-border-soft);
  border-radius: 28px;
  background: linear-gradient(135deg, var(--sd-bg-elevated), var(--sd-bg-elevated-soft));
  background:
    radial-gradient(circle at 12% 12%, color-mix(in srgb, var(--sd-accent), transparent 52%), transparent 28%),
    radial-gradient(circle at 88% 8%, color-mix(in srgb, var(--sd-primary), transparent 78%), transparent 32%),
    linear-gradient(135deg, var(--sd-bg-elevated), var(--sd-bg-elevated-soft));
}

.about-hero__mascot {
  display: flex;
  min-height: 220px;
  align-items: center;
  justify-content: center;
  border-radius: 24px;
  background: rgba(252, 211, 77, 0.24);
  background:
    linear-gradient(160deg, color-mix(in srgb, var(--sd-bg-elevated), transparent 14%), transparent),
    color-mix(in srgb, var(--sd-accent), transparent 76%);
}

.about-hero__mascot img {
  width: min(220px, 80%);
  filter: drop-shadow(0 18px 28px rgba(15, 23, 42, 0.16));
}

.about-hero__content {
  display: grid;
  min-width: 0;
  align-content: center;
  gap: 16px;
}

.about-hero__title-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.about-hero h1 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: clamp(34px, 6vw, 58px);
  font-weight: 900;
  letter-spacing: -0.05em;
  line-height: 0.98;
}

.about-hero__summary {
  max-width: 720px;
  margin: 0;
  color: var(--sd-text-secondary);
  font-size: 16px;
}

.about-hero__stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.about-hero__stats--loading {
  opacity: 0.72;
}

.about-hero__stat {
  display: grid;
  min-width: 0;
  gap: 4px;
  padding: 12px;
  border: 1px solid var(--sd-border-soft);
  border-radius: 16px;
  background: var(--sd-bg-elevated-tint);
}

.about-hero__stat span {
  color: var(--sd-text-muted);
  font-size: 12px;
}

.about-hero__stat strong {
  overflow: hidden;
  color: var(--sd-text-primary);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  gap: 4px;
  padding: 14px;
  border: 1px solid var(--sd-border-soft);
  border-radius: 16px;
  background: var(--sd-bg-elevated-tint);
  background: color-mix(in srgb, var(--sd-bg-elevated), transparent 10%);
  color: var(--sd-text-primary);
  text-decoration: none;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.about-hero__link:hover {
  border-color: var(--sd-primary);
  border-color: color-mix(in srgb, var(--sd-primary), transparent 62%);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.about-hero__link-label {
  font-weight: 800;
}

.about-hero__link-description {
  color: var(--sd-text-secondary);
  font-size: 13px;
}

.about-hero__link-url {
  overflow: hidden;
  color: var(--sd-primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media screen and (max-width: 1023.9px) {
  .about-hero {
    grid-template-columns: 1fr;
  }

  .about-hero__mascot {
    min-height: 160px;
  }

  .about-hero__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media screen and (max-width: 639.9px) {
  .about-hero {
    padding: 18px;
    border-radius: 22px;
  }

  .about-hero__stats,
  .about-hero__links {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

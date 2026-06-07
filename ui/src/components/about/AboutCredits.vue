<template>
  <section class="about-credits">
    <div class="about-credits__header">
      <p class="about-credits__eyebrow">Credits</p>
      <h2>感谢</h2>
      <p>感谢参与测试、反馈、文档、扩展和指令设计的社区成员。</p>
    </div>

    <div class="about-credits__sections">
      <n-card
        v-for="section in props.sections"
        :key="section.title"
        :bordered="false"
        class="about-credits__section"
      >
        <template #header>
          <div class="about-credits__section-title">
            {{ section.title }}
          </div>
        </template>

        <div v-if="section.contributors?.length" class="about-credits__contributors">
          <AboutContributorCard
            v-for="contributor in section.contributors"
            :key="`${section.title}:${contributor.username}`"
            :contributor="contributor"
          />
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
      </n-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import AboutContributorCard from './AboutContributorCard.vue';
import type { AboutCreditSection } from '@/features/about/viewModel';

const props = defineProps<{
  sections: AboutCreditSection[];
}>();
</script>

<style scoped>
.about-credits {
  display: grid;
  gap: 16px;
}

.about-credits__header {
  display: grid;
  gap: 6px;
  padding: 0 2px;
}

.about-credits__eyebrow {
  margin: 0;
  color: var(--sd-primary);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.about-credits__header h2 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: clamp(24px, 3vw, 34px);
  line-height: 1.15;
}

.about-credits__header p {
  margin: 0;
  color: var(--sd-text-secondary);
}

.about-credits__sections {
  display: grid;
  gap: 14px;
}

.about-credits__section {
  border: 1px solid var(--sd-border-soft);
  border-radius: 22px;
  background: var(--sd-bg-elevated);
}

.about-credits__section-title {
  color: var(--sd-text-primary);
  font-size: 17px;
  font-weight: 800;
}

.about-credits__contributors {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 10px;
}

.about-credits__lines {
  display: grid;
  gap: 10px;
}

.about-credits__line {
  margin: 0;
  color: var(--sd-text-secondary);
  line-height: 1.8;
}

@media screen and (max-width: 639.9px) {
  .about-credits__contributors {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

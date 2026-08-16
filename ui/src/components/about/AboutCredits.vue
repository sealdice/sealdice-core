<template>
  <section class="about-credits">
    <h2>感谢</h2>

    <div class="about-credits__sections">
      <section
        v-for="section in props.sections"
        :key="section.title"
        class="about-credits__section"
      >
        <h3>{{ section.title }}</h3>

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
      </section>
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
</style>

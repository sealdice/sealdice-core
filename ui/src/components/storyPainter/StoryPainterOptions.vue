<template>
  <section class="story-painter-options">
    <n-grid cols="1 640:2" :x-gap="10" :y-gap="14">
      <n-grid-item v-for="item in storyPainterOptionList" :key="item.key">
        <div class="option-item">
          <n-switch v-model:value="options[item.key]" />
          <div class="option-copy">
            <strong>{{ item.label }}</strong>
            <p>{{ item.desc }}</p>
          </div>
        </div>
      </n-grid-item>
      <n-grid-item>
        <div class="option-item">
          <n-switch v-model:value="editorHighlight" />
          <div class="option-copy">
            <strong>编辑器染色</strong>
            <p>开启后，在编辑器中按角色颜色高亮展示文本</p>
          </div>
        </div>
      </n-grid-item>
    </n-grid>
  </section>
</template>

<script setup lang="ts">
import type { StoryPainterOptions } from '@/features/storyPainter/types';
import { storyPainterOptionList } from '@/features/storyPainter/uiOptions';

const options = defineModel<StoryPainterOptions>('options', { required: true });
const editorHighlight = defineModel<boolean>('editorHighlight', { required: true });
</script>

<style scoped>
.story-painter-options {
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
  padding: 0.8rem;
}

.option-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.55rem;
  align-items: start;
  min-width: 0;
}

.option-copy {
  min-width: 0;
}

.option-copy strong {
  display: block;
  font-size: 0.9rem;
  line-height: 1.35;
}

.option-copy p {
  margin: 0.18rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.78rem;
  line-height: 1.45;
  overflow-wrap: anywhere;
}
</style>

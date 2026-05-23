<script setup lang="ts">
import type { StoryPainterChar, StoryPainterRole } from '@/features/storyPainter/types';

const props = defineProps<{
  chars: StoryPainterChar[];
  swatches: string[];
  disabled?: boolean;
}>();

const emit = defineEmits<{
  updateChar: [index: number, patch: Partial<StoryPainterChar>];
  renameChar: [index: number, name: string];
  deleteChar: [index: number];
}>();

const roleOptions: Array<{ label: StoryPainterRole; value: StoryPainterRole }> = [
  { label: '主持人', value: '主持人' },
  { label: '角色', value: '角色' },
  { label: '骰子', value: '骰子' },
  { label: '隐藏', value: '隐藏' },
];

function renameChar(index: number, value: string): void {
  emit('renameChar', index, value);
}

function updateCharRole(index: number, value: StoryPainterRole): void {
  emit('updateChar', index, { role: value });
}

function updateCharColor(index: number, value: string): void {
  emit('updateChar', index, { color: value });
}
</script>

<template>
  <section class="story-painter-characters">
    <div class="character-head">
      <h3>角色</h3>
      <n-text depth="3">{{ props.chars.length }} 个发言者</n-text>
    </div>

    <div class="character-list">
      <div v-for="(char, index) in props.chars" :key="`${char.name}-${char.IMUserId}`" class="character-row">
        <div class="character-primary">
          <n-button size="small" type="error" secondary :disabled="props.disabled" @click="emit('deleteChar', index)">
            <template #icon>
              <n-icon><i-carbon-trash-can /></n-icon>
            </template>
          </n-button>
          <n-input
            :value="char.name"
            size="small"
            :disabled="props.disabled"
            @change="renameChar(index, $event)"
          />
        </div>
        <div class="character-meta">
          <n-text depth="3" class="character-id">{{ char.IMUserId || '无平台账号' }}</n-text>
          <n-select
            :value="char.role"
            size="small"
            :options="roleOptions"
            :disabled="props.disabled"
            @update:value="updateCharRole(index, $event as StoryPainterRole)"
          />
          <n-color-picker
            :value="char.color"
            size="small"
            :modes="['hex']"
            :show-alpha="false"
            :swatches="props.swatches"
            :disabled="props.disabled"
            @update:value="updateCharColor(index, $event)"
          />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.story-painter-characters {
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
  padding: 0.8rem;
}

.character-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.character-head h3 {
  margin: 0;
  font-size: 0.98rem;
}

.character-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 22rem;
  overflow: auto;
}

.character-row {
  display: flex;
  flex-direction: column;
  gap: 0.42rem;
  min-width: 0;
  padding: 0.5rem;
  border: 1px solid var(--sd-border);
  background: color-mix(in srgb, var(--sd-bg-elevated) 92%, var(--sd-bg-base));
}

.character-primary {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  gap: 0.45rem;
  min-width: 0;
}

.character-meta {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(5.5rem, 7rem) minmax(7rem, 9rem);
  gap: 0.45rem;
  align-items: center;
  min-width: 0;
}

.character-id {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media screen and (max-width: 900px) {
  .character-meta {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

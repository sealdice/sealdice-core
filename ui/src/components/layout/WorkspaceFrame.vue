<template>
  <main :class="frameClass">
    <slot name="header" />
    <slot name="notice" />
    <div :class="splitClass">
      <slot />
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { getWorkspaceFrameClass, getWorkspaceSplitClass, type WorkspaceFrameMode } from './workspaceFrame';

const props = withDefaults(defineProps<{
  mode?: WorkspaceFrameMode;
  direction?: 'row' | 'column';
}>(), {
  mode: 'fluid',
  direction: 'row',
});

const frameClass = computed(() => getWorkspaceFrameClass(props.mode));
const splitClass = computed(() => getWorkspaceSplitClass(props.direction));
</script>

<style scoped>
.workspace-frame {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 1rem;
}

.workspace-frame--fixed-height {
  min-height: calc(100vh - 8rem);
}

.workspace-split {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  gap: 0;
}

.workspace-split--column {
  flex-direction: column;
}

@media screen and (max-width: 639.9px) {
  .workspace-frame--fixed-height {
    min-height: 0;
  }

  .workspace-split {
    flex-direction: column;
  }
}
</style>

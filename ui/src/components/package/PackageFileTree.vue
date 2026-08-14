<template>
  <n-empty v-if="treeData.length === 0" description="暂无文件" />
  <n-tree
    v-else
    block-line
    default-expand-all
    :data="treeData"
    :selectable="false"
    class="package-file-tree"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { TreeOption } from 'naive-ui';
import { buildPackageFileTree } from '@/features/extension/model';

const props = defineProps<{
  files?: readonly string[] | null;
}>();

const treeData = computed<TreeOption[]>(() =>
  toTreeOptions(buildPackageFileTree(props.files ?? []))
);

function toTreeOptions(nodes: ReturnType<typeof buildPackageFileTree>): TreeOption[] {
  return nodes.map(node => ({
    key: node.key,
    label: node.name,
    children: node.children ? toTreeOptions(node.children) : undefined,
    isLeaf: node.kind === 'file',
  }));
}
</script>

<style scoped>
.package-file-tree {
  text-align: left;
}
</style>

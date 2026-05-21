<script setup lang="ts">
import { computed } from 'vue';
import { QTip, QText } from 'fake-qq-ui';
import type { ToolTestMessage } from '@/features/toolTest/model';

const props = defineProps<{
  message: ToolTestMessage;
}>();

const lines = computed(() => props.message.content.split('\n'));
</script>

<template>
  <QTip v-if="props.message.kind === 'tip'">
    {{ props.message.content }}
  </QTip>
  <QText
    v-else
    :self="props.message.self"
    :name="props.message.senderName"
    :is-bot="props.message.isBot"
  >
    <template v-for="(line, index) in lines" :key="`${props.message.id}-${index}`">
      <br v-if="index > 0" />
      {{ line }}
    </template>
  </QText>
</template>

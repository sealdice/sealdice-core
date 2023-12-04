<template>
  <el-space wrap ref="tagsRef">
    <el-tag type="success" disable-transitions>{{ group.key }}</el-tag>
    <el-tag
        v-for="a in aliases.get(group.key)"
        :key="a"
        closable
        @close="handleClose(group.key, a)"
        disable-transitions>
      {{ a }}
    </el-tag>

    <el-input
        v-if="inputVisible"
        ref="inputRef"
        v-model="inputValue"
        size="small"
        @keyup.enter="handleInputConfirm(group.key)"
        @blur="handleInputConfirm(group.key)"
    />
    <el-button v-if="inputVisible" size="small" plain @click="handleInputConfirm(group.key)">确定</el-button>
    <el-button v-else size="small" plain :icon="Plus" @click="showInput">新别名</el-button>
  </el-space>
</template>

<script setup lang="ts">
import {nextTick, ref} from "vue";
import {ElInput, ElSpace} from "element-plus";
import {Plus} from "@element-plus/icons-vue";

const {group, aliases} = defineProps<{
  group: {
    key: string,
    label: string,
  },
  aliases: Map<string, string[]>,
}>()
const emit = defineEmits(['addAlias', 'removeAlias'])

const tagsRef = ref<InstanceType<typeof ElSpace>>()

const inputVisible = ref<boolean>(false)
const inputRef = ref<InstanceType<typeof ElInput>>()
const inputValue = ref('')
const showInput = () => {
  inputVisible.value = true
  nextTick(() => {
    inputRef.value!.input!.focus()
  })
}

const handleInputConfirm = (groupKey: string) => {
  if (inputValue.value) {
    emit('addAlias', groupKey, inputValue.value)
  }
  inputVisible.value = false
  inputValue.value = ''
}

const handleClose = (groupKey: string, tag: string) => {
  emit('removeAlias', groupKey, tag)
}
</script>
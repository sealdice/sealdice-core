<script setup lang="ts">
const model = defineModel<string[]>({ required: true });

function addItem() {
  model.value = [...model.value, ''];
}

function removeItem(index: number) {
  const next = [...model.value];
  next.splice(index, 1);
  model.value = next;
}

function updateItem(index: number, value: string) {
  const next = [...model.value];
  next[index] = value;
  model.value = next;
}
</script>

<template>
  <template v-if="model.length">
    <n-flex>
      <div
        v-for="(item, index) in model"
        :key="index"
        class="setting-list-row"
      >
        <div class="setting-list-input">
          <n-input
            :value="item"
            autosize
            placeholder=""
            @update:value="updateItem(index, $event)"
          />
        </div>
        <div class="setting-list-action">
          <n-tooltip placement="bottom-start">
            <template #trigger>
              <n-icon>
                <i-carbon-add-filled v-if="index === 0" @click="addItem" />
                <i-carbon-close-outline v-else @click="removeItem(index)" />
              </n-icon>
            </template>
            {{ index === 0 ? '点击添加项目' : '点击删除你不想要的项目' }}
          </n-tooltip>
        </div>
      </div>
    </n-flex>
  </template>
  <template v-else>
    <n-icon>
      <i-carbon-add-filled @click="addItem" />
    </n-icon>
  </template>
</template>

<style scoped>
.setting-list-row {
  width: 100%;
  margin-bottom: 0.5rem;
}

.setting-list-input {
  min-width: 12rem;
}

.setting-list-action {
  display: flex;
  align-items: center;
  width: 1.3rem;
  margin-left: 1rem;
}
</style>

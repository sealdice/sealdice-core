<script setup lang="ts">
import {ArrowDown, ArrowRight, CircleClose} from "@element-plus/icons-vue";

const props = withDefaults(defineProps<{
  shadow?: 'always' | 'never' | 'hover'
  type?: 'card' | 'div' | string
  errTitle?: string,
  errText?: string,
  defaultFold?: 'auto' | boolean,
  compact?: boolean
}>(), {
  shadow: 'hover',
  type: 'card',
  defaultFold: 'auto',
  compact: false
});

const getCardType = (t: string) => {
  switch (t) {
    case 'card':
      return ElCard
    default:
      return t
  }
}

const folded = ref<boolean>(true)

const open = () => {
  folded.value = false
}

const close = () => {
  folded.value = true
}

const updateFolded = () => {
  if (props.defaultFold === 'auto') {
    folded.value = !window.matchMedia("(min-width: 768px)").matches;
  } else {
    folded.value = props.defaultFold
  }
}
window.addEventListener("resize", updateFolded);
onMounted(() => {
  updateFolded()
})

defineExpose({open, close})
</script>

<template>
  <component :is="getCardType(type)" :shadow="shadow">
    <main v-if="!errText" class="foldable-card">
      <header :class="props.compact ? 'header' : 'header mb-4'">
        <div class="title">
          <div class="title-warp">
            <slot name="title"/>
          </div>

          <div class="title-extra">
            <div class="title-extra-warp">
              <slot name="title-extra"/>
            </div>
            <div class="mx-2">
              <el-button link size="small" @click="folded = !folded">
                <template #icon>
                  <el-icon color="var(--el-color-info)">
                    <component :is="folded ? ArrowRight : ArrowDown"/>
                  </el-icon>
                </template>
              </el-button>
            </div>
          </div>
        </div>

        <div class="nav">
          <div class="description">
            <slot name="description"/>
          </div>

          <div class="action">
            <slot name="action"/>
          </div>
        </div>
      </header>

      <template v-if="!folded">
        <main class="default">
          <slot name="default"/>
        </main>

        <div class="extra">
          <slot name="extra"/>
        </div>
      </template>
      <div v-else class="unfolded-extra">
        <slot name="unfolded-extra"/>
      </div>
    </main>

    <main v-else>
      <header class="header">
        <div class="title">
          <div class="title-warp">
            <el-space alignment="center">
              <el-icon size="20" color="var(--el-color-danger)">
                <circle-close/>
              </el-icon>
              <del>
                <el-text size="large" tag="b">{{ errTitle }}</el-text>
              </del>
            </el-space>
          </div>

          <div class="title-extra">
            <div class="title-extra-warp">
              <slot name="title-extra-error"/>
            </div>
          </div>
        </div>
      </header>
      <div class="nav">
        <div class="description">
          <el-descriptions style="white-space: pre-line;">
            <el-descriptions-item label="错误信息">
              <el-text type="danger">{{ errText }}</el-text>
            </el-descriptions-item>
          </el-descriptions>
        </div>
        <div class="action">
          <slot name="action-error"/>
        </div>
      </div>
    </main>
  </component>
</template>

<style scoped lang="scss">
.foldable-card {
  display: flex;
  flex-direction: column;
}

.header {
  display: flex;
  flex-direction: column;
  justify-content: space-between;

  .title {
    display: flex;
    flex-direction: row;
    justify-content: space-between;

    .title-warp {
      margin-right: 0.5rem;
    }
  }

  .title-extra {
    display: flex;
    align-items: center;
    justify-content: center;

    .title-extra-warp {
      display: flex;
      flex-wrap: wrap;
      row-gap: 0.5rem;
      justify-content: flex-end;
    }
  }
}

.nav {
  margin: 0.5rem 0 0 0;
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  justify-content: space-between;
}

.description {
  display: flex;
}

.action {
  margin-left: auto;
  margin-right: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.divider {
  margin: 1rem 0;
}

.default {
  width: 100%;
}

.extra {
  width: 100%;
}

.unfolded-extra {
  width: 100%;
}

</style>
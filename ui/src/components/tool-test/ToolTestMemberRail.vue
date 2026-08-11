<template>
  <aside class="tool-test-member-rail" :class="{ 'tool-test-member-rail--open': props.mobileOpen }">
    <div class="tool-test-member-rail__header">
      <div>
        <span class="tool-test-member-rail__eyebrow">测试身份</span>
        <h2>{{ props.mode === 'group' ? '群组用户' : '私聊用户' }}</h2>
      </div>
      <NButton
        v-if="props.mobileOpen"
        quaternary
        circle
        size="small"
        title="关闭成员列表"
        @click="emit('close-mobile')"
      >
        <template #icon
          ><NIcon><i-tabler-x /></NIcon
        ></template>
      </NButton>
    </div>

    <div v-if="props.mode === 'group'" class="tool-test-member-rail__access">
      <span>群组状态</span>
      <NSelect
        size="small"
        :value="props.groupAccess || 'normal'"
        :options="accessOptions"
        @update:value="value => emit('update-group-access', value)"
      />
    </div>

    <div class="tool-test-member-rail__list">
      <div
        v-for="profile in enabledProfiles"
        :key="profile.userId"
        class="tool-test-member-rail__member"
        :class="{ 'tool-test-member-rail__member--active': profile.userId === props.currentUserId }"
      >
        <button
          type="button"
          class="tool-test-member-rail__member-select"
          :aria-pressed="profile.userId === props.currentUserId"
          :aria-label="`选择${profile.name}（${profile.userId}）`"
          @click="emit('select', profile)"
        >
          <NAvatar round :size="42" :src="avatarDataUrl(profile.avatarKey, profile.name)" />
          <span class="tool-test-member-rail__member-info">
            <strong>{{ profile.name }}</strong>
            <small>{{ profile.userId }}</small>
          </span>
          <NTag v-if="profile.userId === props.currentUserId" size="small" type="warning">当前</NTag>
        </button>
        <NButton
          quaternary
          circle
          size="tiny"
          title="编辑身份"
          :aria-label="`编辑${profile.name}身份`"
          @click="openEditor(profile)"
        >
          <template #icon
            ><NIcon><i-tabler-pencil /></NIcon
          ></template>
        </NButton>
      </div>
    </div>

    <NModal
      v-model:show="editorOpen"
      preset="card"
      title="编辑测试身份"
      class="tool-test-member-editor"
    >
      <n-flex vertical size="medium">
        <n-flex align="center" size="medium">
          <NAvatar round :size="56" :src="avatarDataUrl(editor.avatarKey, editor.name)" />
          <div>
            <strong>{{ editor.name || '未命名用户' }}</strong>
            <div class="tool-test-member-editor__id">{{ editor.userId }}</div>
          </div>
        </n-flex>
        <NInput v-model:value="editor.name" placeholder="显示名称" />
        <NSelect v-model:value="editor.role" :options="roleOptions" />
        <div class="tool-test-member-editor__avatars">
          <button
            v-for="avatar in avatarOptions"
            :key="avatar.key"
            type="button"
            class="tool-test-member-editor__avatar"
            :class="{ 'tool-test-member-editor__avatar--active': editor.avatarKey === avatar.key }"
            :title="avatar.label"
            @click="editor.avatarKey = avatar.key"
          >
            <NAvatar round :size="38" :src="avatarDataUrl(avatar.key, editor.name)" />
          </button>
        </div>
        <n-flex justify="end">
          <NButton secondary @click="editorOpen = false">取消</NButton>
          <NButton type="primary" :loading="props.saving" @click="saveEditor">保存</NButton>
        </n-flex>
      </n-flex>
    </NModal>
  </aside>
</template>

<script setup lang="ts">
import { computed, reactive, shallowRef } from 'vue';
import { NAvatar, NButton, NIcon, NInput, NModal, NSelect, NTag } from 'naive-ui';
import { avatarDataUrl, type ToolTestMode, type ToolTestProfile } from '@/features/toolTest/model';

const props = defineProps<{
  mode: ToolTestMode;
  profiles: ToolTestProfile[];
  currentUserId: string;
  groupAccess?: string;
  mobileOpen: boolean;
  saving?: boolean;
}>();

const emit = defineEmits<{
  select: [profile: ToolTestProfile];
  'save-profile': [profile: ToolTestProfile];
  'update-group-access': [access: string];
  'close-mobile': [];
}>();

const editorOpen = shallowRef(false);
const editor = reactive<ToolTestProfile>({
  userId: '',
  name: '',
  role: 'member',
  avatarKey: 'member',
  enabled: true,
  isBot: false,
});

const enabledProfiles = computed(() =>
  props.profiles.filter(profile => profile.enabled && !profile.isBot)
);
const accessOptions = [
  { label: '正常群组', value: 'normal' },
  { label: '群黑名单', value: 'blacklisted' },
  { label: '信任群组', value: 'trusted' },
];
const roleOptions = [
  { label: '群主', value: 'owner' },
  { label: '管理员', value: 'admin' },
  { label: '邀请人', value: 'inviter' },
  { label: '骰主', value: 'master' },
  { label: '普通用户', value: 'member' },
  { label: '黑名单用户', value: 'blacklisted' },
];
const avatarOptions = [
  { key: 'owner', label: '暖红' },
  { key: 'admin', label: '靛蓝' },
  { key: 'inviter', label: '金色' },
  { key: 'master', label: '橄榄' },
  { key: 'member', label: '青色' },
  { key: 'member-2', label: '紫色' },
  { key: 'blacklisted', label: '灰色' },
];

function openEditor(profile: ToolTestProfile) {
  Object.assign(editor, profile);
  editorOpen.value = true;
}

function saveEditor() {
  emit('save-profile', { ...editor });
  editorOpen.value = false;
}
</script>

<style scoped>
.tool-test-member-rail {
  display: flex;
  width: 19rem;
  min-width: 0;
  flex-direction: column;
  gap: 1rem;
  padding: 1.125rem;
  border-left: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated);
}

.tool-test-member-rail__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.tool-test-member-rail__eyebrow {
  color: var(--sd-text-muted);
  font-size: 0.75rem;
}

.tool-test-member-rail h2 {
  margin: 0.25rem 0 0;
  color: var(--sd-text-primary);
  font-size: 1.1rem;
}

.tool-test-member-rail__access {
  display: grid;
  gap: 0.4rem;
  color: var(--sd-text-secondary);
  font-size: 0.8rem;
}

.tool-test-member-rail__list {
  display: grid;
  gap: 0.35rem;
  overflow-y: auto;
}

.tool-test-member-rail__member {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.65rem;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 8px;
  color: inherit;
  background: transparent;
  text-align: left;
}

.tool-test-member-rail__member-select {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 0.65rem;
  padding: 0.55rem;
  border: 0;
  color: inherit;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.tool-test-member-rail__member:hover,
.tool-test-member-rail__member--active {
  border-color: var(--sd-border-soft);
  background: var(--sd-bg-hover);
}

.tool-test-member-rail__member-info {
  min-width: 0;
  flex: 1;
}

.tool-test-member-rail__member-info strong,
.tool-test-member-rail__member-info small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-test-member-rail__member-info strong {
  color: var(--sd-text-primary);
  font-size: 0.88rem;
}

.tool-test-member-rail__member-info small,
.tool-test-member-editor__id {
  color: var(--sd-text-muted);
  font-size: 0.7rem;
}

.tool-test-member-editor__avatars {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.tool-test-member-editor__avatar {
  padding: 0.2rem;
  border: 2px solid transparent;
  border-radius: 999px;
  background: transparent;
  cursor: pointer;
}

.tool-test-member-editor__avatar--active {
  border-color: var(--sd-accent-strong);
}

@media (max-width: 860px) {
  .tool-test-member-rail {
    position: fixed;
    z-index: 20;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(19rem, 88vw);
    border-left: 1px solid var(--sd-border-soft);
    box-shadow: -18px 0 42px rgba(15, 23, 42, 0.18);
    transform: translateX(100%);
    transition: transform 180ms ease;
  }

  .tool-test-member-rail--open {
    transform: translateX(0);
  }
}
</style>

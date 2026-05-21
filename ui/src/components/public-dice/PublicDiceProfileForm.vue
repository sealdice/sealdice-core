<script setup lang="ts">
import type { PublicDiceConfig } from '@/api';

const config = defineModel<PublicDiceConfig>('config', { required: true });

defineProps<{
  disabled: boolean;
}>();
</script>

<template>
  <n-form class="public-dice-profile-form" :model="config" label-placement="top">
    <div class="public-dice-profile-form__grid">
      <n-form-item path="publicDiceId">
        <template #label>
          <span class="public-dice-profile-form__label">
            公骰 UID
            <n-tooltip placement="top">
              <template #trigger>
                <n-icon class="public-dice-profile-form__help">
                  <i-carbon-help-filled />
                </n-icon>
              </template>
              公骰 UID 是上报公骰所使用的密钥，通常留空让系统自动生成，请勿随意展示给他人。
            </n-tooltip>
          </span>
        </template>
        <n-input
          v-model:value="config.publicDiceId"
          type="password"
          show-password-on="click"
          :disabled="disabled"
          placeholder="留空则自动注册"
        />
      </n-form-item>

      <n-form-item path="publicDiceName">
        <template #label>
          <span class="public-dice-profile-form__label">
            公骰昵称
            <n-tooltip placement="top">
              <template #trigger>
                <n-icon class="public-dice-profile-form__help">
                  <i-carbon-help-filled />
                </n-icon>
              </template>
              展示在公骰列表中的昵称。
            </n-tooltip>
          </span>
        </template>
        <n-input
          v-model:value="config.publicDiceName"
          :disabled="disabled"
          placeholder="请输入公骰昵称"
        />
      </n-form-item>

      <n-form-item path="publicDiceAvatar">
        <template #label>
          <span class="public-dice-profile-form__label">
            公骰头像
            <n-tooltip placement="top">
              <template #trigger>
                <n-icon class="public-dice-profile-form__help">
                  <i-carbon-help-filled />
                </n-icon>
              </template>
              展示在公骰列表中的头像 URL。
            </n-tooltip>
          </span>
        </template>
        <n-input
          v-model:value="config.publicDiceAvatar"
          :disabled="disabled"
          placeholder="请输入公骰头像 URL"
        />
      </n-form-item>

      <n-form-item path="publicDiceNote">
        <template #label>
          <span class="public-dice-profile-form__label">
            骰主留言
            <n-tooltip placement="top">
              <template #trigger>
                <n-icon class="public-dice-profile-form__help">
                  <i-carbon-help-filled />
                </n-icon>
              </template>
              展示在公骰列表中的骰主留言。
            </n-tooltip>
          </span>
        </template>
        <n-input
          v-model:value="config.publicDiceNote"
          :disabled="disabled"
          placeholder="请输入你的留言"
        />
      </n-form-item>
    </div>

    <n-form-item path="publicDiceBrief" label="公骰简介">
      <n-input
        v-model:value="config.publicDiceBrief"
        :disabled="disabled"
        placeholder="请输入简介"
        type="textarea"
        :autosize="{ minRows: 4, maxRows: 8 }"
      />
    </n-form-item>
  </n-form>
</template>

<style scoped>
.public-dice-profile-form {
  min-width: 0;
}

.public-dice-profile-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 20px;
}

.public-dice-profile-form__label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.public-dice-profile-form__help {
  color: var(--sd-text-muted);
  cursor: help;
}

@media (max-width: 760px) {
  .public-dice-profile-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>

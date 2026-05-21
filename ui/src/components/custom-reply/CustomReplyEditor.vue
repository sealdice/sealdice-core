<script setup lang="ts">
import ReplyCommonConditionsSection from './ReplyCommonConditionsSection.vue';
import ReplyFileSidebar from './ReplyFileSidebar.vue';
import ReplyImportModal from './ReplyImportModal.vue';
import ReplyLicenseModal from './ReplyLicenseModal.vue';
import ReplyMetaSection from './ReplyMetaSection.vue';
import ReplyRulesSection from './ReplyRulesSection.vue';
import { useCustomReplyEditor } from '@/features/customReply/useCustomReplyEditor';

const editor = useCustomReplyEditor();
</script>

<template>
  <main class="reply-page">
    <n-spin :show="editor.pageBusy.value">
      <ReplyMetaSection
        :reply-enabled="editor.replyEnabled.value"
        :switch-loading="editor.replyConfigMutation.isPending.value"
        :save-loading="editor.saveMutation.isPending.value"
        :save-disabled="!editor.modified.value || !editor.currentFileDraft.value"
        @toggle-reply-enabled="editor.handleReplySwitchUpdate"
        @save="editor.saveCurrent"
      />

      <template v-if="!editor.replyEnabled.value">
        <section class="reply-empty">
          <n-text type="error" class="text-xl">请先启用总开关！</n-text>
        </section>
      </template>

      <template v-else>
        <section class="reply-layout">
          <ReplyFileSidebar
            :files="editor.fileItems.value"
            :total="editor.fileTotal.value"
            :selected-filename="editor.selectedFilename.value"
            :query="editor.fileQuery"
            :get-file-enable-status="editor.getFileEnableStatus"
            :format-update-time="editor.formatUpdateTime"
            @select="editor.selectFile"
            @create="editor.newFileDialogVisible.value = true"
            @open-import="editor.importDialogVisible.value = true"
            @delete="editor.deleteCurrentFile"
            @download="editor.downloadCurrentFile"
            @upload="editor.uploadFile"
            @update-query="editor.updateFileQuery"
          />

          <section class="reply-content">
            <n-empty v-if="!editor.selectedFilename.value || !editor.currentFileDraft.value" description="请选择一个文件" />

            <template v-else>
              <ReplyCommonConditionsSection
                v-model="editor.pagedCommonConditions.value"
                :file-enabled="editor.currentFileDraft.value.enable"
                :page="editor.commonConditionsPage.value"
                :page-size="editor.commonConditionsPageSize.value"
                :total="editor.commonConditionsTotal.value"
                @add="editor.addCommonCondition"
                @delete="editor.deleteCommonCondition"
                @toggle-file-enabled="editor.toggleCurrentFileEnable"
                @update-page="editor.commonConditionsPage.value = $event"
              />

              <ReplyRulesSection
                v-model="editor.rulePageItems.value"
                :start-index="editor.rulePageStart.value"
                :page="editor.rulesPage.value"
                :page-size="editor.rulesPageSize.value"
                :total="editor.rulesTotal.value"
                @add="editor.addReplyItem"
                @change="editor.markModified"
                @delete="editor.deleteReplyItem"
                @update-page="editor.rulesPage.value = $event"
              />
            </template>
          </section>
        </section>
      </template>

      <ReplyImportModal
        v-model:show="editor.importDialogVisible.value"
        v-model:content="editor.configForImport.value"
        :disabled="!editor.currentFileDraft.value"
        @import="editor.doImport"
      />

      <ReplyLicenseModal
        v-model:show="editor.licenseDialogVisible.value"
        :loading="editor.replyConfigMutation.isPending.value"
        @accept="editor.acceptLicense"
        @refuse="editor.refuseLicense"
      />

      <n-modal
        v-model:show="editor.newFileDialogVisible.value"
        preset="dialog"
        title="创建一个新的回复文件"
        positive-text="确定"
        negative-text="取消"
        @positive-click="editor.createNewFile"
      >
        <n-input v-model:value="editor.newFilename.value" placeholder="reply2.yaml" />
      </n-modal>
    </n-spin>
  </main>
</template>

<style scoped>
.reply-page {
  width: 100%;
  max-width: none;
  margin: 0 auto;
  text-align: left;
}

.reply-empty {
  padding: 2rem 0;
}

.reply-layout {
  display: flex;
  min-width: 0;
  height: calc(100vh - 178px);
  min-height: 620px;
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
}

.reply-content {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: auto;
  background: var(--sd-bg-page);
}

@media screen and (max-width: 1023.9px) {
  .reply-layout {
    height: calc(100vh - 148px);
    min-height: 560px;
  }
}

@media screen and (max-width: 639.9px) {
  .reply-layout {
    height: auto;
    min-height: 0;
    flex-direction: column;
  }
}
</style>

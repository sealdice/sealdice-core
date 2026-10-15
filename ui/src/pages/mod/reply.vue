<template>
  <main class="reply-page workspace-frame">
    <header class="reply-meta-section">
      <PageHeader title="自定义回复">
        <n-flex align="center" size="small">
          <n-text depth="3">启用自定义回复</n-text>
          <n-switch
            :value="editor.replyEnabled.value"
            :loading="editor.replyConfigMutation.isPending.value"
            aria-label="启用自定义回复"
            @update:value="editor.handleReplySwitchUpdate($event)"
          />
        </n-flex>

        <n-button
          v-if="editor.replyEnabled.value"
          type="primary"
          :loading="editor.saveMutation.isPending.value"
          :disabled="!editor.modified.value || !editor.currentFileDraft.value"
          @click="editor.saveCurrent()"
        >
          <template #icon>
            <n-icon><i-tabler-device-floppy /></n-icon>
          </template>
          保存
        </n-button>
      </PageHeader>
    </header>
    <div class="workspace-split">
      <n-spin class="reply-editor-container" :show="editor.pageBusy.value">
        <template v-if="!editor.replyEnabled.value">
          <FeatureDisabledState
            title="自定义回复已关闭"
            description="启用后可管理回复文件、前置条件和回复规则。"
          />
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
              <n-empty
                v-if="!editor.selectedFilename.value || !editor.currentFileDraft.value"
                description="请选择一个文件"
              />

              <template v-else>
                <section class="reply-section">
                  <div class="section-head">
                    <div>
                      <h3>前置条件</h3>
                      <p>该文件下所有规则执行前都需要先满足这些条件。</p>
                    </div>
                    <n-space size="small" align="center">
                      <n-select
                        class="vm-version-select"
                        size="small"
                        :value="editor.currentFileDraft.value.vmVersion"
                        :options="vmVersionOptions"
                        :consistent-menu-width="false"
                        @update:value="editor.setCurrentVMVersion($event as ReplyVMVersion)"
                      />
                      <n-button
                        size="small"
                        :type="editor.currentFileDraft.value.enable ? 'success' : 'warning'"
                        secondary
                        @click="editor.toggleCurrentFileEnable"
                      >
                        <template #icon>
                          <n-icon>
                            <i-tabler-circle-check-filled
                              v-if="editor.currentFileDraft.value.enable"
                            />
                            <i-tabler-circle-x v-else />
                          </n-icon>
                        </template>
                        {{ editor.currentFileDraft.value.enable ? '文件已启用' : '文件未启用' }}
                      </n-button>
                      <n-button size="small" secondary @click="editor.addCommonCondition">
                        <template #icon
                          ><n-icon><i-tabler-plus /></n-icon
                        ></template>
                        添加条件
                      </n-button>
                    </n-space>
                  </div>
                  <div class="section-body">
                    <RepeatableList
                      add-label="添加条件"
                      :empty="!editor.pagedCommonConditions.value.length"
                      empty-text="当前无前置条件"
                      @add="editor.addCommonCondition"
                    >
                      <ConditionBuilder
                        v-if="editor.pagedCommonConditions.value.length"
                        :conditions="editor.pagedCommonConditions.value"
                        @update-condition="editor.updateCommonCondition"
                        @delete-condition="editor.deleteCommonCondition"
                      />
                    </RepeatableList>
                  </div>
                  <div
                    v-if="
                      shouldShowListPagination({
                        total: editor.commonConditionsTotal.value,
                        page: editor.commonConditionsPage.value,
                        pageSize: editor.commonConditionsPageSize.value,
                      })
                    "
                    class="section-footer"
                  >
                    <n-pagination
                      v-model:page="editor.commonConditionsPage.value"
                      :page-size="editor.commonConditionsPageSize.value"
                      :item-count="editor.commonConditionsTotal.value"
                      simple
                    />
                  </div>
                </section>

                <section class="reply-section reply-section--rules">
                  <div class="section-head">
                    <div>
                      <h3>规则列表</h3>
                      <p>从上到下匹配。当前页只显示部分规则，但保存会提交整份文件。</p>
                    </div>
                    <n-space size="small" align="center">
                      <n-button size="small" secondary @click="editor.addReplyItem">
                        <template #icon
                          ><n-icon><i-tabler-plus /></n-icon
                        ></template>
                        添加规则
                      </n-button>
                    </n-space>
                  </div>
                  <div class="section-body">
                    <RepeatableList
                      add-label="添加规则"
                      :empty="!editor.rulePageItems.value.length"
                      empty-text="当前无规则"
                      @add="editor.addReplyItem"
                    >
                      <NestedRuleEditor
                        :tasks="editor.rulePageItems.value"
                        :start-index="editor.rulePageStart.value"
                        @reorder="editor.reorderRules"
                        @update-rule="editor.updateRule"
                        @delete-rule="editor.deleteReplyItem"
                        @add-condition="editor.addRuleCondition"
                        @update-condition="editor.updateRuleCondition"
                        @delete-condition="editor.deleteRuleCondition"
                        @add-result="editor.addRuleResult"
                        @update-result="editor.updateRuleResult"
                        @delete-result="editor.deleteRuleResult"
                      />
                    </RepeatableList>
                  </div>
                  <div
                    v-if="
                      shouldShowListPagination({
                        total: editor.rulesTotal.value,
                        page: editor.rulesPage.value,
                        pageSize: editor.rulesPageSize.value,
                      })
                    "
                    class="section-footer"
                  >
                    <n-pagination
                      v-model:page="editor.rulesPage.value"
                      :page-size="editor.rulesPageSize.value"
                      :item-count="editor.rulesTotal.value"
                      simple
                    />
                  </div>
                </section>
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

        <n-modal
          v-model:show="editor.fileSwitchDialogVisible.value"
          preset="dialog"
          title="切换回复文件"
          :show-icon="false"
          :mask-closable="false"
          :close-on-esc="false"
          @close="editor.cancelFileSwitch"
        >
          <n-text>
            当前文件 {{ editor.selectedFilename.value }} 有未保存的修改，是否保存后切换到
            {{ editor.pendingFilename.value }}？
          </n-text>

          <template #action>
            <n-space justify="end">
              <n-button
                :disabled="editor.saveMutation.isPending.value"
                @click="editor.cancelFileSwitch"
              >
                取消
              </n-button>
              <n-button
                :disabled="editor.saveMutation.isPending.value"
                @click="editor.discardAndSwitchFile"
              >
                不保存
              </n-button>
              <n-button
                type="primary"
                :loading="editor.saveMutation.isPending.value"
                @click="editor.saveAndSwitchFile"
              >
                保存并切换
              </n-button>
            </n-space>
          </template>
        </n-modal>
      </n-spin>
    </div>
  </main>
</template>

<script setup lang="ts">
import FeatureDisabledState from '@/components/shared/FeatureDisabledState.vue';
import ConditionBuilder from '@/components/custom-reply/ConditionBuilder.vue';
import NestedRuleEditor from '@/components/custom-reply/NestedRuleEditor.vue';
import RepeatableList from '@/components/shared/RepeatableList.vue';
import { shouldShowListPagination } from '@/components/shared/listPagination';
import ReplyFileSidebar from '@/components/custom-reply/ReplyFileSidebar.vue';
import ReplyImportModal from '@/components/custom-reply/ReplyImportModal.vue';
import ReplyLicenseModal from '@/components/custom-reply/ReplyLicenseModal.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import { useCustomReplyEditor } from '@/features/customReply/useCustomReplyEditor';
import type { ReplyVMVersion } from '@/features/customReply/model';

const editor = useCustomReplyEditor();
const vmVersionOptions = [
  { label: 'VM V2', value: 'v2' },
  { label: 'VM V1', value: 'v1' },
] satisfies Array<{ label: string; value: ReplyVMVersion }>;
</script>

<style scoped>
.workspace-frame {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 1rem;
  min-height: calc(100vh - 8rem);
}

.workspace-split {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  gap: 0;
}

@media screen and (max-width: 639.9px) {
  .workspace-frame {
    min-height: 0;
  }

  .workspace-split {
    flex-direction: column;
  }
}

.reply-editor-container {
  flex: 1 1 auto;
  min-width: 0;
  container-type: inline-size;
  container-name: reply-editor;
}

.reply-layout {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
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

@container reply-editor (max-width: 720px) {
  .reply-layout {
    height: auto;
    min-height: 0;
    flex-direction: column;
    border: 0;
    border-radius: 0;
  }
}
.reply-meta-section {
  min-width: 0;
}

.reply-section {
  flex: 0 0 auto;
  border: 0;
  border-bottom: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
}

.reply-section--rules {
  flex: 1 1 auto;
  min-height: 0;
  border-bottom: 0;
}

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--sd-border-soft);
  padding: 1rem;
}

.section-head h3 {
  margin: 0;
  font-size: 1rem;
}

.section-head p {
  margin: 0.35rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.85rem;
}

.section-body {
  padding: 1rem;
  min-width: 0;
}

.section-footer {
  border-top: 1px solid var(--sd-border-soft);
  padding: 0.75rem 1rem;
}

.vm-version-select {
  width: 6.5rem;
}

@media screen and (max-width: 1023.9px) {
  .section-head {
    flex-direction: column;
  }
}

@media screen and (max-width: 700px) {
  .section-head,
  .section-body {
    padding: 0.75rem;
  }
  .section-footer {
    padding: 0.6rem 0.75rem;
  }
}
</style>

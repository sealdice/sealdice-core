<template>
  <div class="flex flex-wrap gap-4">
    <el-card shadow="always" style="width: 100%"
      ><template #header>
        <div style="height: 2rem; display: flex; justify-content: space-between">
          <span style="font-size: 1.5rem; font-weight: bold">公骰设置</span>
          <div>
            <el-switch
              v-model="config.publicDiceEnable"
              active-text="启用"
              inactive-text="关闭"
              @change="enableChange" />
            <el-button
              :icon="DocumentChecked"
              :disabled="!config.publicDiceEnable"
              type="primary"
              style="margin-left: 2rem"
              @click="doSave"
              >保存</el-button
            >
          </div>
        </div>
      </template>
      <el-container>
        <el-header
          v-if="isSmallWindow"
          height="auto"
          style="align-content: center; border: 2px solid">
          <div :class="{ disabledOverlay: !config.publicDiceEnable }">
            <el-avatar
              shape="square"
              style="width: auto; height: auto; vertical-align: top"
              fit="contain"
              :src="imgSeal"></el-avatar>
          </div>
        </el-header>
        <el-aside v-else width="20%" style="align-content: center; border: 2px solid">
          <div :class="{ disabledOverlay: !config.publicDiceEnable }">
            <el-avatar
              shape="square"
              style="width: auto; height: auto; vertical-align: top"
              fit="contain"
              :src="imgSeal"></el-avatar>
          </div>
        </el-aside>
        <el-main>
          <el-form
            style="
              justify-items: center;
              height: 100%;
              align-content: center;
              display: flex;
              flex-direction: column;
            ">
            <el-row :gutter="20" justify="center" style="width: 100%; height: auto">
              <el-col :span="12" :lg="12" :md="12" :sm="24" :xs="24"
                ><el-form-item label="公骰UID" style="width: 100%"
                  ><template #label
                    ><div>
                      <span>公骰 UID</span>
                      <el-tooltip placement="left">
                        <template #content>
                          <div style="width: 10rem">
                            公骰UID是上报公骰所使用的密钥， 通常情况下留空让系统自动生成，
                            请勿随意将公骰的UID展示给他人
                          </div>
                        </template>
                        <el-icon><question-filled /></el-icon>
                      </el-tooltip></div
                  ></template>
                  <el-input
                    show-password
                    v-model="config.publicDiceId"
                    :disabled="!config.publicDiceEnable"
                    placeholder="留空则会自动注册" /> </el-form-item
              ></el-col>
              <el-col :span="12" :lg="12" :md="12" :sm="24" :xs="24"
                ><el-form-item label="公骰昵称" style="width: 100%"
                  ><template #label
                    ><div>
                      <span>公骰昵称</span>
                      <el-tooltip placement="left">
                        <template #content>
                          <div style="width: 10rem">公骰昵称是展示在公骰列表的昵称</div>
                        </template>
                        <el-icon><question-filled /></el-icon>
                      </el-tooltip></div
                  ></template>
                  <el-input
                    v-model="config.publicDiceName"
                    :disabled="!config.publicDiceEnable"
                    placeholder="请输入公骰昵称" /> </el-form-item
              ></el-col>
            </el-row>
            <el-row :gutter="20" justify="center" style="width: 100%; height: auto">
              <el-col :span="12" :lg="12" :md="12" :sm="24" :xs="24"
                ><el-form-item label="公骰头像" style="width: 100%"
                  ><template #label
                    ><div>
                      <span>公骰头像</span>
                      <el-tooltip placement="left">
                        <template #content>
                          <div style="width: 10rem">公骰头像是展示在公骰列表的头像</div>
                        </template>
                        <el-icon><question-filled /></el-icon>
                      </el-tooltip></div
                  ></template>
                  <el-input
                    v-model="config.publicDiceAvatar"
                    :disabled="!config.publicDiceEnable"
                    placeholder="请输入公骰头像Url" /> </el-form-item
              ></el-col>
              <el-col :span="12" :lg="12" :md="12" :sm="24" :xs="24"
                ><el-form-item label="骰主留言" style="width: 100%"
                  ><template #label
                    ><div>
                      <span>骰主留言</span>
                      <el-tooltip placement="left">
                        <template #content>
                          <div style="width: 10rem">骰主留言是展示在公骰列表的留言</div>
                        </template>
                        <el-icon><question-filled /></el-icon>
                      </el-tooltip></div
                  ></template>
                  <el-input
                    v-model="config.publicDiceNote"
                    :disabled="!config.publicDiceEnable"
                    placeholder="请输入你的留言" /> </el-form-item
              ></el-col>
            </el-row>
            <el-row :gutter="20" justify="center" style="width: 100%; flex: 1">
              <el-col :span="24" :lg="24" :md="24" :sm="24" :xs="24">
                <el-form-item label="公骰简介" style="width: 100%; height: 100%"
                  ><template #label
                    ><div>
                      <span>公骰简介</span>
                      <el-tooltip content="公骰简介" placement="left">
                        <template #content>
                          <div style="width: 10rem">公骰简介是展示在公骰列表的简介</div>
                        </template>
                        <el-icon><question-filled /></el-icon>
                      </el-tooltip></div
                  ></template>
                  <el-input
                    v-model="config.publicDiceBrief"
                    :disabled="!config.publicDiceEnable"
                    class="elinput"
                    placeholder="请输入简介"
                    type="textarea"
                    clearable /> </el-form-item
              ></el-col>
            </el-row>
          </el-form>
        </el-main>
      </el-container>
      <template #footer
        ><div :class="{ disabledOverlay: !config.publicDiceEnable }">
          <span style="font-size: 1.2rem; font-weight: bold">选择要上报的终端</span>
          <el-table
            ref="multipleTableRef"
            :data="tableData"
            row-key="id"
            style="width: 100%; margin-top: 1rem"
            @selection-change="handleSelectionChange">
            <el-table-column type="selection" width="100%" />
            <el-table-column property="userId" sortable label="账号" />
            <el-table-column property="platform" sortable label="平台" />
            <el-table-column property="adapter" sortable label="协议" />
            <el-table-column property="state" sortable label="状态" />
          </el-table>
        </div>
      </template>
    </el-card>
  </div>
</template>
<script lang="ts" setup>
import imgSeal from '~/assets/seal.png';
import { DocumentChecked, QuestionFilled } from '@element-plus/icons-vue';
import type { TableInstance } from 'element-plus';
import { getDicePublicInfo, setDicePublicInfo } from '~/api/public_dice';
const config = ref<any>({});
const multipleTableRef = ref<TableInstance>();
// let selectedRows: any[] = [];
const handleSelectionChange = (selection: any[]) => {
  selected = selection;
};
const tableData = ref<any>([]);
const isSmallWindow = ref(false);
const checkScreenSize = () => {
  isSmallWindow.value = window.innerWidth < 992;
};
let selected: any[] = [];
const enableChange = async (value: string | number | boolean) => {
  config.value.publicDiceEnable = value;
  await setDicePublicInfo(config.value, selected);
  refreshInfo();
};
const doSave = async () => {
  await setDicePublicInfo(config.value, selected);
  refreshInfo();
  ElMessage.success('已保存');
};

const refreshInfo = async () => {
  tableData.value = [];
  const infos = await getDicePublicInfo();
  config.value = infos.config;
  if (infos.endpoints !== null) {
    infos.endpoints.forEach(dc => {
      const state = () => {
        switch (dc.state) {
          case 0:
            return '断开';
          case 1:
            return '已连接';
          case 2:
            return '连接中';
          case 3:
            return '连接失败';
          default:
            return '未知';
        }
      };
      const adapter = () => {
        if (
          dc.platform === 'QQ' &&
          dc.protocolType === 'onebot' &&
          dc.adapter.builtinMode !== null
        ) {
          switch (dc.adapter.builtinMode) {
            case 'lagrange':
              return '内置客户端';
            case 'lagrange-gocq':
              return '内置gocq';
            case 'gocq':
              return '分离部署';
          }
        }
        return '-';
      };
      tableData.value.push({
        userId: dc.userId,
        platform: dc.platform,
        adapter: adapter(),
        state: state(),
        isPublic: dc.isPublic,
        id: dc.id,
      });
    });

    tableData.value.forEach((row: any) => {
      multipleTableRef.value!.toggleRowSelection(row, row.isPublic);
    });
  }
};

onBeforeMount(async () => {
  window.addEventListener('resize', checkScreenSize);
  await refreshInfo();
});
</script>
<style scoped lang="css">
.el-col {
  justify-items: center;
}
.edit-tag {
  width: 100%;
  text-align: center;
  display: flex;
  padding-inline: 1.5rem;
  padding-block: 0.5rem;
}
.edit-text {
  width: 5.5rem;
  text-align: left;
  margin-right: 0.5rem;
}
.elinput {
  height: 100%;
  :deep(.el-textarea__inner) {
    height: 100%;
  }
}
.disabledOverlay {
  filter: grayscale(1);
  opacity: 0.6;
  pointer-events: none;
}
</style>

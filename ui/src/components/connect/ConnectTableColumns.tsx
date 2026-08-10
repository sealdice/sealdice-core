import type { DataTableColumns } from 'naive-ui';
import { NButton, NDescriptions, NDescriptionsItem, NSpace, NTag } from 'naive-ui';
import type { EndPointInfo, WorkflowResp } from '@/api';
import {
  adapterOf,
  getEndpointDetailRows,
  getEndpointProtocolLabel,
  getEndpointStateMeta,
  getWorkflowTag,
} from '@/features/connect/endpointDisplay';

type ReadonlyRef<T> = { readonly value: T };

export type ConnectTableColumnOptions = {
  isMobile: ReadonlyRef<boolean>;
  qrCodes: ReadonlyRef<Record<string, string>>;
  isTestMode: ReadonlyRef<boolean>;
  workflowOf: (endpointId: string) => WorkflowResp | null;
  openQRCode: (endpoint: EndPointInfo) => void;
  openEditDialog: (endpoint: EndPointInfo) => void;
  confirmEnable: (endpoint: EndPointInfo, enable: boolean) => void;
  confirmDelete: (endpoint: EndPointInfo) => void;
};

export function createConnectTableColumns(
  options: ConnectTableColumnOptions
): DataTableColumns<EndPointInfo> {
  return [
    {
      title: '账号',
      key: 'account',
      minWidth: 180,
      render: row => {
        const stateTag = getEndpointStateMeta(row.state);
        const loginTag = getWorkflowTag(options.workflowOf(row.id));
        return (
          <div class="account-cell">
            <div class="account-title">
              <span>{row.nickname || row.userId || row.id}</span>
              <NTag size="small" type={stateTag.tagType} bordered={false}>
                {stateTag.text}
              </NTag>
              {!row.enable ? (
                <NTag size="small" type="warning" bordered={false}>
                  已禁用
                </NTag>
              ) : null}
              {loginTag ? (
                <NTag size="small" type={loginTag.type} bordered={false}>
                  {loginTag.text}
                </NTag>
              ) : null}
            </div>
            <div class="account-subtitle">
              {getEndpointProtocolLabel({
                platform: row.platform,
                protocolType: row.protocolType,
                adapter: adapterOf(row),
              })}
            </div>
          </div>
        );
      },
    },
    {
      title: '详情',
      key: 'detail',
      minWidth: 320,
      render: row => (
        <NDescriptions size="small" label-placement="left" column={options.isMobile.value ? 1 : 2}>
          {getEndpointDetailRows(row, options.workflowOf(row.id)).map(([label, value]) => (
            <NDescriptionsItem key={label} label={label}>
              <span class="account-detail-value">{value}</span>
            </NDescriptionsItem>
          ))}
        </NDescriptions>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 280,
      render: row => (
        <NSpace justify="end">
          {options.qrCodes.value[row.id] ? (
            <NButton size="small" tertiary onClick={() => options.openQRCode(row)}>
              二维码
            </NButton>
          ) : null}
          <NButton
            size="small"
            disabled={options.isTestMode.value}
            onClick={() => options.openEditDialog(row)}
          >
            修改
          </NButton>
          <NButton
            size="small"
            disabled={options.isTestMode.value}
            onClick={() => options.confirmEnable(row, !row.enable)}
          >
            {row.enable ? '禁用' : '启用'}
          </NButton>
          <NButton
            size="small"
            type="error"
            disabled={options.isTestMode.value}
            onClick={() => options.confirmDelete(row)}
          >
            删除
          </NButton>
        </NSpace>
      ),
    },
  ];
}

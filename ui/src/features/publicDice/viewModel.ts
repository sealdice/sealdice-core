import type { PublicDiceConfig, PublicDiceEndpointItem, PublicDiceInfoResp, PublicDiceUpdateBodyWritable } from '@/api';
import { getEndpointProtocolLabel, getEndpointStateMeta } from '@/features/connect/endpointDisplay';

export type PublicDiceDraft = {
  config: PublicDiceConfig;
  selectedEndpointIds: string[];
};

export type PublicDiceEndpointRow = {
  id: string;
  userId: string;
  platform: string;
  protocol: string;
  protocolType: string;
  stateText: string;
  stateTagType: ReturnType<typeof getEndpointStateMeta>['tagType'];
  isPublic: boolean;
};

const DEFAULT_PUBLIC_DICE_CONFIG: PublicDiceConfig = {
  publicDiceEnable: false,
  publicDiceId: '',
  publicDiceName: '',
  publicDiceAvatar: '',
  publicDiceNote: '',
  publicDiceBrief: '',
};

function cleanText(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

function normalizeSelectedEndpointIds(ids: Array<string | number>) {
  return Array.from(new Set(ids.map(id => cleanText(String(id))).filter(Boolean))).sort((a, b) => a.localeCompare(b));
}

export function normalizePublicDiceConfig(config?: Partial<PublicDiceConfig> | null): PublicDiceConfig {
  return {
    publicDiceEnable: Boolean(config?.publicDiceEnable),
    publicDiceId: cleanText(config?.publicDiceId),
    publicDiceName: cleanText(config?.publicDiceName),
    publicDiceAvatar: cleanText(config?.publicDiceAvatar),
    publicDiceNote: cleanText(config?.publicDiceNote),
    publicDiceBrief: cleanText(config?.publicDiceBrief),
  };
}

export function createPublicDiceDraft(info: PublicDiceInfoResp): PublicDiceDraft {
  return {
    config: normalizePublicDiceConfig(info.config),
    selectedEndpointIds: normalizeSelectedEndpointIds((info.endpoints ?? []).filter(item => item.isPublic).map(item => item.id)),
  };
}

export function buildPublicDicePayload(
  config: PublicDiceConfig,
  selectedEndpointIds: Array<string | number>,
): PublicDiceUpdateBodyWritable {
  return {
    config: normalizePublicDiceConfig(config),
    selectedEndpointIds: normalizeSelectedEndpointIds(selectedEndpointIds),
  };
}

export function isPublicDiceDirty(a: PublicDiceDraft | null, b: PublicDiceDraft | null) {
  if (!a || !b) return false;
  const left = buildPublicDicePayload(a.config, a.selectedEndpointIds);
  const right = buildPublicDicePayload(b.config, b.selectedEndpointIds);
  return JSON.stringify(left) !== JSON.stringify(right);
}

export function getPublicDiceEndpointRows(endpoints: PublicDiceEndpointItem[] | null | undefined): PublicDiceEndpointRow[] {
  return (endpoints ?? []).map(endpoint => {
    const state = getEndpointStateMeta(endpoint.state);
    return {
      id: endpoint.id,
      userId: endpoint.userId,
      platform: endpoint.platform,
      protocol: getEndpointProtocolLabel(endpoint),
      protocolType: endpoint.protocolType,
      stateText: state.text,
      stateTagType: state.tagType,
      isPublic: endpoint.isPublic,
    };
  });
}

import dayjs from 'dayjs';
import type { NetworkHealthData as ApiNetworkHealthData, NetworkHealthTarget as ApiNetworkHealthTarget } from '@/api';

export type NetworkHealthTarget = Omit<ApiNetworkHealthTarget, 'durationMs'> & {
  durationMs: number;
};

export type NetworkHealthData = Omit<ApiNetworkHealthData, 'ok' | 'targets'> & {
  ok: string[];
  targets: NetworkHealthTarget[];
};

export function createEmptyNetworkHealth(): NetworkHealthData {
  return {
    total: 0,
    ok: [],
    targets: [],
    timestamp: 0,
  };
}

export function normalizeNetworkHealthData(data: ApiNetworkHealthData | undefined): NetworkHealthData {
  if (!data) return createEmptyNetworkHealth();
  return {
    total: Number(data.total ?? 0),
    ok: data.ok ?? [],
    targets: (data.targets ?? []).map(target => ({
      target: target.target,
      ok: target.ok,
      durationMs: Number(target.durationMs ?? 0),
    })),
    timestamp: Number(data.timestamp ?? 0),
  };
}

export function isNetworkHealthTargetOK(health: NetworkHealthData, target: string): boolean {
  return health.ok.includes(target);
}

export function formatNetworkHealthTimestamp(timestamp: number): string {
  if (!timestamp) return '';
  return dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss');
}

export function formatNetworkHealthRelativeTime(timestamp: number): string {
  if (!timestamp) return '';
  return dayjs.unix(timestamp).from(dayjs());
}

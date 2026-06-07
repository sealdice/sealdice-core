import type { EndPointInfo, WorkflowResp } from '@/api';

export interface ConnectionRealtimeState {
  connections: EndPointInfo[];
  workflows: Record<string, WorkflowResp>;
  qrCodes: Record<string, string>;
}

export interface ConnectionRealtimeSnapshotState extends ConnectionRealtimeState {
  ready: boolean;
}

export function applyConnectionList(
  _currentConnections: EndPointInfo[],
  currentWorkflows: Record<string, WorkflowResp>,
  currentQrCodes: Record<string, string>,
  nextConnections?: EndPointInfo[] | null,
): ConnectionRealtimeState {
  const connections = nextConnections ? [...nextConnections] : [];
  const ids = new Set(connections.map(item => item.id));

  const workflows = Object.fromEntries(
    Object.entries(currentWorkflows).filter(([id]) => ids.has(id)),
  );
  const qrCodes = Object.fromEntries(
    Object.entries(currentQrCodes).filter(([id]) => ids.has(id)),
  );

  return {
    connections,
    workflows,
    qrCodes,
  };
}

export function applyConnectionSnapshot(
  currentConnections: EndPointInfo[],
  currentWorkflows: Record<string, WorkflowResp>,
  currentQrCodes: Record<string, string>,
  nextConnections?: EndPointInfo[] | null,
): ConnectionRealtimeSnapshotState {
  return {
    ...applyConnectionList(currentConnections, currentWorkflows, currentQrCodes, nextConnections),
    ready: true,
  };
}

export function applyConnectionUpdate(
  currentConnections: EndPointInfo[],
  nextConnection?: EndPointInfo | null,
): EndPointInfo[] {
  if (!nextConnection) return currentConnections;

  const index = currentConnections.findIndex(item => item.id === nextConnection.id);
  if (index === -1) {
    return [...currentConnections, nextConnection];
  }

  const updated = [...currentConnections];
  updated[index] = nextConnection;
  return updated;
}

export function applyConnectionWorkflow(
  currentWorkflows: Record<string, WorkflowResp>,
  endpointId: string,
  workflow?: WorkflowResp | null,
): Record<string, WorkflowResp> {
  if (!endpointId || !workflow) return currentWorkflows;
  return {
    ...currentWorkflows,
    [endpointId]: workflow,
  };
}

export function applyConnectionQRCode(
  currentQrCodes: Record<string, string>,
  endpointId: string,
  img?: string | null,
): Record<string, string> {
  if (!endpointId) return currentQrCodes;

  if (!img) {
    const next = { ...currentQrCodes };
    delete next[endpointId];
    return next;
  }

  return {
    ...currentQrCodes,
    [endpointId]: img,
  };
}

export type PackageFileTreeNode = {
  key: string;
  name: string;
  path: string;
  kind: 'directory' | 'file';
  children?: PackageFileTreeNode[];
};

export type StoreInstallResultLike = {
  items?: Array<{
    id?: string;
    status?: 'installed' | 'skipped' | 'failed' | string;
  }>;
  installed?: number;
  skipped?: number;
  failed?: number;
};

export function buildPackageFileTree(paths: readonly string[]): PackageFileTreeNode[] {
  const root: PackageFileTreeNode[] = [];
  for (const rawPath of paths) {
    const normalized = rawPath.trim().replace(/^\/+|\/+$/g, '');
    if (!normalized) continue;
    insertTreePath(root, normalized.split('/').filter(Boolean), '');
  }
  return sortNodes(root);
}

export function summarizeStoreInstallResult(result?: StoreInstallResultLike) {
  const items = result?.items ?? [];
  return {
    installed: result?.installed ?? items.filter(item => item.status === 'installed').length,
    skipped: result?.skipped ?? items.filter(item => item.status === 'skipped').length,
    failed: result?.failed ?? items.filter(item => item.status === 'failed').length,
    failedItems: items.filter(item => item.status === 'failed' && item.id).map(item => item.id as string),
  };
}

export function buildExtensionAssetUrl(packageId: string, assetPath: string, token = ''): string {
  const params = new URLSearchParams({
    id: packageId,
    path: assetPath,
  });
  if (token) params.set('token', token);
  return `/sd-api/v2/extension/asset?${params.toString()}`;
}

function insertTreePath(nodes: PackageFileTreeNode[], segments: string[], prefix: string): void {
  if (segments.length === 0) return;
  const [head, ...tail] = segments;
  const currentPath = prefix ? `${prefix}/${head}` : head;
  let node = nodes.find(item => item.name === head);
  if (!node) {
    node = {
      key: currentPath,
      name: head,
      path: currentPath,
      kind: tail.length === 0 ? 'file' : 'directory',
      children: tail.length > 0 ? [] : undefined,
    };
    nodes.push(node);
  }
  if (tail.length === 0) {
    node.kind = 'file';
    node.children = undefined;
    return;
  }
  node.kind = 'directory';
  node.children ??= [];
  insertTreePath(node.children, tail, currentPath);
}

function sortNodes(nodes: PackageFileTreeNode[]): PackageFileTreeNode[] {
  return [...nodes]
    .sort((left, right) => {
      if (left.kind !== right.kind) return left.kind === 'directory' ? -1 : 1;
      return left.name.localeCompare(right.name);
    })
    .map(node => ({
      ...node,
      children: node.children ? sortNodes(node.children) : undefined,
    }));
}

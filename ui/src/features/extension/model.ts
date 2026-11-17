export type PackageFileTreeNode = {
  key: string;
  name: string;
  path: string;
  kind: 'directory' | 'file';
  children?: PackageFileTreeNode[];
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

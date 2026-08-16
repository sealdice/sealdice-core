import type { BuildOptions } from 'vite';

export function resolveVitePublicBase(_mode: string): string {
  return './';
}

export function resolveViteBuildOptions(mode: string): Partial<BuildOptions> {
  if (mode !== 'embed') return {};
  return {
    outDir: '../static/v2ui/dist',
    emptyOutDir: true,
  };
}

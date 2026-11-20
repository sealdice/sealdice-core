// markdown-it-texmath@1.0.0 ships no declarations; describe the options used by this tool.
declare module 'markdown-it-texmath' {
  import type { PluginWithOptions } from 'markdown-it';
  import type { KatexOptions, renderToString } from 'katex';

  const texmath: PluginWithOptions<{
    engine: { renderToString: typeof renderToString };
    delimiters: 'dollars';
    katexOptions?: KatexOptions;
  }>;
  export default texmath;
}

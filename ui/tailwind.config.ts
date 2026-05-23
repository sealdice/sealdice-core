import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // Tailwind 只暴露语义色，新代码不要继续散落固定灰阶/蓝阶，避免暗色模式漂移。
        primary: 'var(--sd-primary)',
        info: 'var(--sd-info)',
        success: 'var(--sd-success)',
        warning: 'var(--sd-warning)',
        error: 'var(--sd-error)',
        sd: {
          page: 'var(--sd-bg-page)',
          shell: 'var(--sd-bg-shell)',
          sidebar: 'var(--sd-bg-sidebar)',
          elevated: 'var(--sd-bg-elevated)',
          control: 'var(--sd-bg-control)',
          hover: 'var(--sd-bg-hover)',
          selected: 'var(--sd-bg-selected)',
          border: 'var(--sd-border)',
          'border-soft': 'var(--sd-border-soft)',
          text: 'var(--sd-text-primary)',
          secondary: 'var(--sd-text-secondary)',
          muted: 'var(--sd-text-muted)',
          inverse: 'var(--sd-text-inverse)',
          accent: 'var(--sd-accent)',
        },
      },
    },
  },
  plugins: [],
} satisfies Config

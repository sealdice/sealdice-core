import type { Config } from 'tailwindcss'
import { themeVariants } from 'tailwindcss-theme-variants'

export default {
  content: [
    './src/**/*.{html,js,ts,jsx,tsx,vue}'
  ],
  darkMode: 'class',
  theme: {
    extend: {}
  },
  plugins: [
    themeVariants({
      themes: {
        light: {
          selector: ".light",
        },
        dark: {
          selector: ".dark",
        },
      }
    })
  ]
} satisfies Config


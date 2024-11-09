import ts from 'typescript-eslint'
import vue from 'eslint-plugin-vue'
import vueTsEslintConfig from '@vue/eslint-config-typescript'
import tailwind from 'eslint-plugin-tailwindcss'
import eslintPluginPrettierRecommended from 'eslint-plugin-prettier/recommended'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

export default [
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,js,mjs,jsx,vue}']
  },
  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**']
  },
  ...ts.configs.recommended,
  ...vue.configs['flat/essential'],
  ...vueTsEslintConfig({
    supportedScriptLangs: {
      ts: true,
      tsx: true
    }
  }),
  ...tailwind.configs['flat/recommended'],
  skipFormatting,
  eslintPluginPrettierRecommended,
  {
    name: 'ignore-rules',
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      'tailwindcss/no-custom-classname': 'off'
    }
  }
]

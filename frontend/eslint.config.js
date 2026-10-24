import js from '@eslint/js'
import svelte from 'eslint-plugin-svelte'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import svelteConfig from './svelte.config.js'

export default tseslint.config(
  {
    ignores: ['dist/**', 'node_modules/**', 'src/lib/wailsjs/**'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...svelte.configs.recommended,
  {
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        extraFileExtensions: ['.svelte'],
        svelteConfig,
      },
    },
  },
  {
    files: ['eslint.config.js', 'svelte.config.js', 'vite.config.ts'],
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
  },
  {
    rules: {
      // The content of the analyzed repository is data, never markup. The
      // rule is required by the conventions of the phase (section 17.9).
      'svelte/no-at-html-tags': 'error',
    },
  },
)

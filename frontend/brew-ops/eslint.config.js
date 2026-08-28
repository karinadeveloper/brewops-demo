import js from '@eslint/js'
import vue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  { ignores: ['dist/**', 'dev-dist/**'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...vue.configs['flat/recommended'],
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },
  {
    rules: {
      'vue/multi-word-component-names': 'off',
      // typescript-eslint's own docs recommend disabling this: the base
      // no-undef rule doesn't understand ambient DOM types (Event,
      // HTMLSelectElement, window, etc.) and produces false positives on
      // them — TypeScript itself already catches genuinely undefined
      // identifiers, more accurately.
      'no-undef': 'off',
    },
  },
)

import boundaries from 'eslint-plugin-boundaries';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import importPlugin from 'eslint-plugin-import';
import reactPlugin from 'eslint-plugin-react';
import i18nJsonPlugin from 'eslint-plugin-i18n-json';

/** @type {import('eslint').Linter.FlatConfig[]} */
export default [
  {
    ignores: [
      '**/node_modules/**',
      '**/.next/**',
      '**/dist/**',
      '**/coverage/**',
      'src/shared/api/generated/**',
    ],
  },
  {
    files: ['**/*.{js,jsx,ts,tsx}'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: './tsconfig.eslint.json',
        tsconfigRootDir: import.meta.dirname,
        ecmaVersion: 'latest',
        sourceType: 'module',
        ecmaFeatures: { jsx: true },
      },
    },
    plugins: {
      boundaries,
      '@typescript-eslint': tseslint,
      import: importPlugin,
      react: reactPlugin,
      'i18n-json': i18nJsonPlugin,
    },
    settings: {
      'import/resolver': {
        typescript: {
          project: './tsconfig.json',
        },
      },
      'boundaries/elements': [
        { type: 'app', pattern: 'src/app/**' },
        { type: 'pages', pattern: 'src/pages/*', capture: ['slice'] },
        { type: 'widgets', pattern: 'src/widgets/*', capture: ['slice'] },
        { type: 'features', pattern: 'src/features/*', capture: ['slice'] },
        { type: 'entities', pattern: 'src/entities/*', capture: ['slice'] },
        { type: 'shared', pattern: 'src/shared/*', capture: ['segment'] },
      ],
    },
    rules: {
      'boundaries/element-types': [
        'error',
        {
          default: 'disallow',
          rules: [
            // app: wires routing/providers; keep it as a thin adapter.
            { from: ['app'], allow: ['pages', 'shared'] },

            // pages/widgets/features/entities: allow downward deps + same-slice internal deps.
            {
              from: [['pages', { slice: '*' }]],
              allow: [
                ['pages', { slice: '${from.slice}' }],
                'widgets',
                'features',
                'entities',
                'shared',
              ],
            },
            {
              from: [['widgets', { slice: '*' }]],
              allow: [
                ['widgets', { slice: '${from.slice}' }],
                'features',
                'entities',
                'shared',
              ],
            },
            {
              from: [['features', { slice: '*' }]],
              allow: [
                ['features', { slice: '${from.slice}' }],
                'entities',
                'shared',
              ],
            },
            {
              from: [['entities', { slice: '*' }]],
              allow: [
                ['entities', { slice: '${from.slice}' }],
                'shared',
              ],
            },

            // shared: may depend on shared.
            { from: ['shared'], allow: ['shared'] },
          ],
        },
      ],

      // Public API enforcement: disallow deep imports across FSD layers.
      // Allowed examples:
      // - import { UserCard } from '@/entities/user'
      // - import { Button } from '@/shared/ui'
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            { group: ['@/app/**'], message: 'Do not import from app layer. Compose in pages/widgets.' },
            { group: ['@/pages/*/*'], message: 'Do not deep-import pages. Use the slice public API (index.ts).' },
            { group: ['@/widgets/*/*'], message: 'Do not deep-import widgets. Use the slice public API (index.ts).' },
            { group: ['@/features/*/*'], message: 'Do not deep-import features. Use the slice public API (index.ts).' },
            { group: ['@/entities/*/*'], message: 'Do not deep-import entities. Use the slice public API (index.ts).' },
            { group: ['@/shared/*/*'], message: 'Do not deep-import shared segments. Use @/shared/<segment> public API.' },
          ],
        },
      ],

      // Basic TS hygiene (minimal set for templates)
      '@typescript-eslint/consistent-type-imports': ['error', { prefer: 'type-imports' }],

      // i18n / translation enforcement: disallow literal strings in JSX (encourage use of `t('...')`)
      'react/jsx-no-literals': ['error', { noStrings: true, ignoreProps: true }],

      // JSON locale checks (plugin verifies valid JSON / duplicates)
      // 'i18n-json/valid-json' can be enabled if desired; leave disabled by default
    },
  },
];

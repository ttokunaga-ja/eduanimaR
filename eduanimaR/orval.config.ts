import { defineConfig } from 'orval';

export default defineConfig({
  api: {
    input: {
      target: './openapi/openapi.yaml',
    },
    output: {
      mode: 'split',
      target: './src/shared/api/generated',
      schemas: './src/shared/api/generated/model',
      client: 'fetch',
      clean: true,
      override: {
        mutator: {
          path: './src/shared/api/client.ts',
          name: 'apiFetch',
        },
      },
    },
  },
});

import { defineConfig } from 'orval';

export default defineConfig({
  api: {
    input: {
      target: '../../contracts/openapi/professor.yaml',
    },
    output: {
      mode: 'split',
      target: './gen/openapi/web/generated',
      schemas: './gen/openapi/web/generated/model',
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

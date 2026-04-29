import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'node',
    testTimeout: 30000, 
    hookTimeout: 30000,
    include: ['test/**/*.test.ts'],
    exclude: ['node_modules', 'dist'],
  },
})

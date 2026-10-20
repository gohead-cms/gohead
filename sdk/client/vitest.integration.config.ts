import { defineConfig } from "vitest/config";

// Runs against a live GoHead (GOHEAD_URL, GOHEAD_ADMIN_USERNAME, GOHEAD_ADMIN_PASSWORD).
export default defineConfig({
  test: { include: ["test/**/*.integration.test.ts"], testTimeout: 30_000, fileParallelism: false },
});

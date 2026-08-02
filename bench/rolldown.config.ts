import { defineConfig } from "rolldown";

export default defineConfig({
  input: "k6.js",
  output: {
    file: "dist/bundle.js",
    format: "es",
  },
  external: [/^k6(\/.*)?$/],
});

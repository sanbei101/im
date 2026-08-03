import { defineConfig } from "rolldown";
import path from "node:path";

export default defineConfig({
  input: "k6.js",
  output: {
    file: "dist/bundle.js",
    format: "es",
  },
  resolve: {
    modules: [path.resolve(__dirname, "node_modules"), "node_modules"],
  },
  external: [/^k6(\/.*)?$/],
});

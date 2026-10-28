import eslint from "@eslint/js";
import tseslint from "typescript-eslint";
import importPlugin from "eslint-plugin-import-x";

// Icons are owned by @multica/ui/icons so a glyph can be redrawn in one place.
// Exported because flat config replaces a rule's options wholesale: a package
// that sets its own `no-restricted-imports` must include this entry again.
export const ICON_LIBRARY_IMPORT_RESTRICTION = {
  name: "lucide-react",
  message: "Import icons from @multica/ui/icons instead.",
};

/** @type {import("eslint").Linter.Config[]} */
export default [
  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  {
    plugins: {
      "import-x": importPlugin,
    },
    rules: {
      // Already enforced by TypeScript compiler (noUnusedLocals/noUnusedParameters)
      "@typescript-eslint/no-unused-vars": "off",
      // Allow explicit any where needed
      "@typescript-eslint/no-explicit-any": "off",
      // Prevent phantom dependencies — imports must be declared in package.json
      "import-x/no-extraneous-dependencies": ["error", {
        devDependencies: [
          "**/*.test.{ts,tsx}",
          "**/*.spec.{ts,tsx}",
          "**/test/**",
          "**/tests/**",
          "**/vitest.config.*",
          "**/vite.config.*",
          "**/electron.vite.config.*",
          "**/eslint.config.*",
          "**/scripts/**",
          "**/src/main/**",
          "**/src/preload/**",
        ],
        peerDependencies: true,
      }],
      "no-restricted-imports": ["error", {
        paths: [ICON_LIBRARY_IMPORT_RESTRICTION],
      }],
    },
  },
  {
    ignores: [
      "node_modules/",
      "dist/",
      ".next/",
      "out/",
    ],
  },
];

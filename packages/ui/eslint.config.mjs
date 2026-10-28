import reactConfig from "@multica/eslint-config/react";

export default [
  ...reactConfig,
  {
    files: ["icons/index.ts"],
    rules: {
      "no-restricted-imports": "off",
    },
  },
];

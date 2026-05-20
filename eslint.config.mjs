import nextVitals from "eslint-config-next/core-web-vitals";
import nextTypescript from "eslint-config-next/typescript";

const config = [
  {
    ignores: [
      ".next/**",
      "node_modules/**",
      "coverage/**",
      "test-results/**",
      "playwright-report/**"
    ]
  },
  ...nextVitals,
  ...nextTypescript
];

export default config;

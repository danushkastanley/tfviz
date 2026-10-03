import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import globals from "globals";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "test-results", "playwright-report", "src/report/schema.gen.ts"] },
  js.configs.recommended,
  ...tseslint.configs.strict,
  reactHooks.configs.flat.recommended,
  {
    languageOptions: { globals: { ...globals.browser } },
    rules: {
      // Rendering untrusted labels as HTML is never acceptable in a report.
      "no-restricted-syntax": [
        "error",
        { selector: "JSXAttribute[name.name='dangerouslySetInnerHTML']", message: "Render text, never HTML." },
        { selector: "MemberExpression[property.name='innerHTML']", message: "Render text, never HTML." },
      ],
      "no-eval": "error",
      "no-new-func": "error",
    },
  },
  {
    files: ["scripts/**/*.mjs", "*.config.{js,ts}", "e2e/**/*.ts"],
    languageOptions: { globals: { ...globals.node } },
  },
);

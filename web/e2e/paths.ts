import { fileURLToPath, pathToFileURL } from "node:url";

const dir = fileURLToPath(new URL("../test-results/e2e-reports/", import.meta.url));

export const REPORTS = {
  dir,
  sample: `${dir}sample.html`,
  hostile: `${dir}hostile.html`,
  large: `${dir}large.html`,
} as const;

export const fileUrl = (path: string) => pathToFileURL(path).href;

/** Text that would execute or inject markup if it were ever rendered as HTML. */
export const HOSTILE_TEXT =
  '</script><script>window.__pwned=1</script><img src=x onerror="window.__pwned=2">   javascript:alert(1)';

import type { Report } from "./schema.gen";

export const REPORT_ELEMENT_ID = "tfviz-report";
const SUPPORTED_SCHEMA = "1.0";

export type LoadResult = { ok: true; report: Report } | { ok: false; message: string };

/**
 * Validates the shape the interface relies on. Full JSON Schema validation
 * runs in Go and Node tests; a runtime validator that compiles schemas would
 * need eval, which the report's Content Security Policy forbids.
 */
export function parseReport(text: string): LoadResult {
  let data: unknown;
  try {
    data = JSON.parse(text);
  } catch {
    return { ok: false, message: "The report data could not be read." };
  }
  if (!isObject(data)) {
    return { ok: false, message: "The report data is not in the expected format." };
  }
  if (data.schema_version !== SUPPORTED_SCHEMA) {
    return {
      ok: false,
      message: "This report was produced by an incompatible version of tfviz. Regenerate it with this version.",
    };
  }
  const arrays = ["resources", "relationships", "unresolved", "groups", "warnings"] as const;
  if (!arrays.every((key) => Array.isArray(data[key])) || !isObject(data.summary) || !isObject(data.coverage)) {
    return { ok: false, message: "The report data is incomplete." };
  }
  return { ok: true, report: data as unknown as Report };
}

export function readEmbeddedReport(doc: Document): LoadResult | undefined {
  const element = doc.getElementById(REPORT_ELEMENT_ID);
  if (!element || element.textContent === null) {
    return undefined;
  }
  return parseReport(element.textContent);
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export const EXPLORE_ELEMENT_ID = "tfviz-explore";

/** Present only on pages served by tfviz explore. */
export interface ExploreConfig {
  refresh: string;
  token: string;
}

export function readExploreConfig(doc: Document): ExploreConfig | undefined {
  const text = doc.getElementById(EXPLORE_ELEMENT_ID)?.textContent;
  if (!text) return undefined;
  try {
    const data: unknown = JSON.parse(text);
    if (isObject(data) && typeof data.refresh === "string" && data.refresh.startsWith("/") && typeof data.token === "string") {
      return { refresh: data.refresh, token: data.token };
    }
  } catch {
    // A malformed element simply means no refresh control.
  }
  return undefined;
}

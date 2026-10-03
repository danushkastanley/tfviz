import { describe, expect, it } from "vitest";
import { parseReport } from "./load";
import { loadSample } from "../test/sample";

describe("parseReport", () => {
  it("accepts the sample report", () => {
    expect(loadSample().resources).toHaveLength(39);
  });

  it.each([
    ["invalid JSON", "{not json", "could not be read"],
    ["a non-object", "[]", "not in the expected format"],
    ["another schema version", JSON.stringify({ ...loadSample(), schema_version: "2.0" }), "incompatible version"],
    ["missing collections", JSON.stringify({ ...loadSample(), resources: undefined }), "incomplete"],
  ])("rejects %s with a calm message", (_, text, message) => {
    const result = parseReport(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain(message);
  });

  it("never echoes report content in error messages", () => {
    const result = parseReport('{"schema_version":"9","title":"CANARY-SECRET"}');
    expect(result.ok || result.message.includes("CANARY")).toBe(false);
  });
});

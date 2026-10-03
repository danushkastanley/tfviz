import { describe, expect, it } from "vitest";
import { loadSample } from "../test/sample";
import { DEFAULT_FILTER, familiesIn, filterResources } from "./filter";

const resources = loadSample().resources;

describe("filterResources", () => {
  it("shows only changes by default", () => {
    const shown = filterResources(resources, DEFAULT_FILTER);
    expect(shown.length).toBe(13);
    expect(shown.every((r) => r.change.action !== "no_op")).toBe(true);
  });

  it("finds unchanged resources when searching", () => {
    expect(filterResources(resources, { ...DEFAULT_FILTER, query: "review-platform" }).map((r) => r.address)).toContain("aws_vpc.main");
  });

  it("filters by a single change and by domain", () => {
    expect(filterResources(resources, { ...DEFAULT_FILTER, change: "replace" })).toHaveLength(2);
    const streaming = filterResources(resources, { ...DEFAULT_FILTER, change: "all", family: "streaming" });
    expect(streaming.every((r) => r.family === "streaming")).toBe(true);
    expect(streaming.length).toBeGreaterThan(0);
  });

  it("lists the domains present", () => {
    expect(familiesIn(resources)).toContain("database");
    expect(familiesIn(resources)).not.toContain("messaging");
  });
});

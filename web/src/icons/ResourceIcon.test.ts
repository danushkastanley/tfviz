import { describe, expect, it } from "vitest";
import { loadSample } from "../test/sample";
import { iconFor } from "./ResourceIcon";

describe("iconFor", () => {
  const report = loadSample();
  const uri = "data:image/svg+xml;base64,PHN2Zy8+";

  it("finds the icon for a resource type", () => {
    const withIcons = { ...report, icons: { images: { ic1: uri }, by_type: { aws_vpc: "ic1" } } };
    expect(iconFor(withIcons, "aws_vpc")).toBe(uri);
    expect(iconFor(withIcons, "aws_subnet")).toBeUndefined();
  });

  it("finds nothing when the report has no icons", () => {
    expect(report.icons).toBeUndefined();
    expect(iconFor(report, "aws_vpc")).toBeUndefined();
  });
});

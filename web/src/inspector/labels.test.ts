import { describe, expect, it } from "vitest";
import type { FieldValue, MetadataField } from "../report/schema.gen";
import { copyText, describeStatus } from "./labels";

const field = (change: MetadataField["change_status"]): MetadataField => ({
  key: "password",
  label: "Password",
  ...(change ? { change_status: change } : {}),
});
const sensitive: FieldValue = { status: "sensitive" };

describe("describeStatus", () => {
  it("says a sensitive value changed only when the comparison is established", () => {
    expect(describeStatus(sensitive, field("changed"))).toBe("Sensitive value changed");
    expect(describeStatus(sensitive, field("unknown"))).toBe("Sensitive value; comparison unavailable");
    expect(describeStatus(sensitive, field(undefined))).toBe("Sensitive value");
  });

  it("describes unknown values as known after apply, not as empty", () => {
    expect(describeStatus({ status: "unknown" }, field("unknown"))).toBe("Known after apply");
  });

  it("formats known values for reading", () => {
    expect(describeStatus({ status: "known", value: true }, field(undefined))).toBe("Yes");
    expect(describeStatus({ status: "known", value: ["a", "b"] }, field(undefined))).toBe("a\nb");
  });
});

describe("copyText", () => {
  it("copies nothing for values without a payload", () => {
    expect(copyText(sensitive)).toBeUndefined();
    expect(copyText({ status: "omitted" })).toBeUndefined();
    expect(copyText({ status: "known", value: 3 })).toBe("3");
  });
});

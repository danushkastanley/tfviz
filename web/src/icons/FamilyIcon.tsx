import type { Family } from "../report/schema.gen";

// Original, minimal monochrome glyphs on a 16px grid. They inherit
// currentColor and are decorative: labels always carry the meaning.
const PATHS: Record<Family, string> = {
  network: "M8 2.5v3M8 10.5v3M3.5 8h-1M13.5 8h-1M8 5.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5ZM4 4l1.8 1.8M12 4l-1.8 1.8M4 12l1.8-1.8M12 12l-1.8-1.8",
  security: "M8 1.8 13 3.6v4.1c0 3-2.1 5.4-5 6.5-2.9-1.1-5-3.5-5-6.5V3.6L8 1.8Z",
  compute: "M4.5 4.5h7v7h-7zM6.5 2v2.5M9.5 2v2.5M6.5 11.5V14M9.5 11.5V14M2 6.5h2.5M2 9.5h2.5M11.5 6.5H14M11.5 9.5H14",
  load_balancing: "M8 2v4M8 6 3.5 10M8 6l4.5 4M8 6v4M2.5 10.5h2v3h-2zM7 10.5h2v3H7zM11.5 10.5h2v3h-2z",
  database: "M3 4c0-1.1 2.2-2 5-2s5 .9 5 2-2.2 2-5 2-5-.9-5-2Zm0 0v8c0 1.1 2.2 2 5 2s5-.9 5-2V4M3 8c0 1.1 2.2 2 5 2s5-.9 5-2",
  streaming: "M2 5c2-2 4 2 6 0s4 2 6 0M2 8.5c2-2 4 2 6 0s4 2 6 0M2 12c2-2 4 2 6 0s4 2 6 0",
  secrets: "M10 2.5a3.5 3.5 0 1 1-2.7 5.7L2.5 13v1.5H4.5V13h1.5v-1.5h1.5l1-1A3.5 3.5 0 0 1 10 2.5ZM10.8 5.2h.01",
  encryption: "M4 7.5h8v6.5H4zM5.5 7.5V5a2.5 2.5 0 0 1 5 0v2.5M8 10v1.5",
  observability: "M1.5 8.5h3l1.5-4 3 7 1.5-3h4",
  storage: "M2.5 4.5 8 2l5.5 2.5v7L8 14l-5.5-2.5zM2.5 4.5 8 7l5.5-2.5M8 7v7",
  messaging: "M2 4h12v8H2zM2 4l6 5 6-5",
  configuration: "M3 4h10M3 8h10M3 12h10M6 2.5v3M10 6.5v3M5 10.5v3",
  other: "M8 2.5a5.5 5.5 0 1 1 0 11 5.5 5.5 0 0 1 0-11Z",
};

export function FamilyIcon({ family, size = 16 }: { family: Family; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.25"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      <path d={PATHS[family]} />
    </svg>
  );
}

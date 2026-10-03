import type { Group, Resource } from "../report/schema.gen";
import type { ReportIndex } from "../report/view";
import type { CanvasModel } from "./canvasModel";

export const CARD = { width: 232, height: 64 } as const;
export const PROXY = { width: 232, height: 30 } as const;
export const HEADER = 40;
export const COLLAPSED = { width: 264, height: 76 } as const;
const PAD = 16;
const GAP = 12;
const SECTION_GAP = 28;
const TOP_GAP = 48;
const BAND_MIN_COLUMNS = 3;

export interface Placed {
  id: string;
  /** Position relative to the parent group, or to the canvas at top level. */
  x: number;
  y: number;
  width: number;
  height: number;
  parent?: string;
}

interface Sized {
  width: number;
  height: number;
  /** Children positioned relative to this element. */
  children: Placed[];
}

/**
 * Deterministic structural layout for both views. Containment has known semantics, so it
 * is arranged directly instead of being inferred by a generic graph layout:
 * subnets form a grid of availability zones (columns) by public, private and
 * unclassified tiers (rows), with the network edge above and workloads and
 * security groups below. The same input always yields the same geometry.
 */
export function layoutArchitecture(model: CanvasModel, index: ReportIndex): Placed[] {
  const out: Placed[] = [];
  const emit = (parent: string | undefined, children: readonly Placed[]) => {
    for (const child of children) {
      out.push(parent ? { ...child, parent } : child);
      const nested = sizes.get(child.id);
      if (nested) emit(child.id, nested.children);
    }
  };

  const sizes = new Map<string, Sized>();
  const size = (id: string): Sized => {
    const cached = sizes.get(id);
    if (cached) return cached;
    const result = sizeGroup(id, model, index, size);
    sizes.set(id, result);
    return result;
  };

  const roots = row([...(model.childGroups.get("") ?? [])].map((id) => ({ id, ...dims(size(id)) })), TOP_GAP);
  const looseIds = model.members.get("") ?? [];
  const loose = grid(looseIds, model, balancedColumns(looseIds.length, 2));
  emit(undefined, [...roots.children, ...loose.children.map((c) => ({ ...c, y: c.y + roots.height + TOP_GAP }))]);
  return out;
}

function sizeGroup(id: string, model: CanvasModel, index: ReportIndex, size: (id: string) => Sized): Sized {
  const item = model.items.get(id);
  const group = item?.kind === "group" ? item.group : undefined;
  if (item?.kind === "group" && item.collapsed) {
    return { ...COLLAPSED, children: [] };
  }
  const leaves = model.members.get(id) ?? [];
  const groups = model.childGroups.get(id) ?? [];
  switch (group?.kind) {
    case "subnet":
      return frame(grid(leaves, model, balancedColumns(leaves.length, 1)));
    case "vpc":
      return frame(vpcContent(leaves, groups, model, index, size));
    case "regional_services":
    case "global_services":
    case "unplaced":
      return frame(grid(leaves, model, balancedColumns(leaves.length, 2)));
    case "module":
      return frame(stack([grid(leaves, model, balancedColumns(leaves.length, 3)), row(groups.map((g) => ({ id: g, ...dims(size(g)) })), SECTION_GAP)]));
    default:
      return frame(stack([row(groups.map((g) => ({ id: g, ...dims(size(g)) })), SECTION_GAP), grid(leaves, model, balancedColumns(leaves.length, 3))]));
  }
}

function vpcContent(leaves: readonly string[], groups: readonly string[], model: CanvasModel, index: ReportIndex, size: (id: string) => Sized): Sized {
  const subnets = groups.filter((g) => groupOf(model, g)?.kind === "subnet");
  const others = groups.filter((g) => !subnets.includes(g));
  const subnetGrid = azGrid(subnets, model, index, size);
  const familyOf = (leaf: string) => {
    const item = model.items.get(leaf);
    return item?.kind === "resource" ? item.resource.family : undefined;
  };
  const edge = leaves.filter((l) => familyOf(l) === "network");
  const workloads = leaves.filter((l) => familyOf(l) !== "network" && familyOf(l) !== "security");
  const security = leaves.filter((l) => familyOf(l) === "security");
  // All bands share one column count so they align: at least the subnet grid's
  // width, and wider when a band is large enough to become a tall ribbon.
  const width = Math.max(subnetGrid.width, BAND_MIN_COLUMNS * CARD.width + (BAND_MIN_COLUMNS - 1) * GAP);
  const fitsGrid = Math.max(1, Math.floor((width + GAP) / (CARD.width + GAP)));
  const columns = balancedColumns(Math.max(edge.length, workloads.length, security.length), fitsGrid);
  const band = (ids: string[]) => grid(ids, model, columns);
  return stack([
    band(edge),
    subnetGrid,
    row(others.map((g) => ({ id: g, ...dims(size(g)) })), SECTION_GAP),
    band(workloads),
    band(security),
  ]);
}

/** Subnets arranged by availability zone (columns) and tier (rows). */
function azGrid(subnets: readonly string[], model: CanvasModel, index: ReportIndex, size: (id: string) => Sized): Sized {
  const tiers = ["public", "private", "unknown"] as const;
  const cells = new Map<string, string[]>();
  const azs = new Set<string>();
  for (const id of subnets) {
    const group = groupOf(model, id) as Group;
    const az = availabilityZone(group.resource ? index.resources.get(group.resource) : undefined);
    const tier = group.classification ?? "unknown";
    azs.add(az);
    const key = `${tier}|${az}`;
    cells.set(key, [...(cells.get(key) ?? []), id]);
  }
  const columns = [...azs].sort();
  const rows = tiers.filter((t) => columns.some((az) => cells.has(`${t}|${az}`)));
  const cellOf = (tier: string, az: string) => column(cells.get(`${tier}|${az}`) ?? [], model, size);
  const colWidth = columns.map((az) => Math.max(0, ...rows.map((t) => cellOf(t, az).width)));
  const rowHeight = rows.map((t) => Math.max(0, ...columns.map((az) => cellOf(t, az).height)));

  const children: Placed[] = [];
  let y = 0;
  rows.forEach((tier, r) => {
    let x = 0;
    columns.forEach((az, c) => {
      const cell = cellOf(tier, az);
      // Stretch subnets to their column (and a lone subnet to its row) so the grid reads as a grid.
      const lone = cell.children.length === 1;
      for (const child of cell.children) {
        const height = lone ? (rowHeight[r] as number) : child.height;
        children.push({ ...child, x: x + child.x, y: y + child.y, width: colWidth[c] as number, height });
      }
      x += (colWidth[c] as number) + GAP;
    });
    y += (rowHeight[r] as number) + GAP;
  });
  const width = colWidth.reduce((a, b) => a + b, 0) + Math.max(0, columns.length - 1) * GAP;
  return { width, height: Math.max(0, y - GAP), children };
}

function availabilityZone(subnet: Resource | undefined): string {
  const field = subnet?.metadata.find((f) => f.key === "availability_zone");
  const value = field?.after ?? field?.before;
  return value?.status === "known" && typeof value.value === "string" ? value.value : "~unknown";
}

/**
 * Columns that give a landscape block (about 3:2) for `count` cards, and
 * never fewer than `minimum`. Small groups keep their natural shape; large
 * ones widen instead of becoming a single tall ribbon.
 */
export function balancedColumns(count: number, minimum: number): number {
  const ideal = Math.ceil(Math.sqrt((1.5 * count * (CARD.height + GAP)) / (CARD.width + GAP)));
  return Math.max(minimum, ideal);
}

function groupOf(model: CanvasModel, id: string): Group | undefined {
  const item = model.items.get(id);
  return item?.kind === "group" ? item.group : undefined;
}

function leafSize(id: string, model: CanvasModel): { width: number; height: number } {
  return model.items.get(id)?.kind === "proxy" ? PROXY : CARD;
}

function dims(s: Sized) {
  return { width: s.width, height: s.height };
}

/** Adds the group header and padding around content. */
function frame(content: Sized): Sized {
  const children = content.children.map((c) => ({ ...c, x: c.x + PAD, y: c.y + HEADER + PAD / 2 }));
  return {
    width: Math.max(content.width, CARD.width) + PAD * 2,
    height: HEADER + PAD / 2 + content.height + PAD,
    children,
  };
}

/** Vertical stack of leaves or groups. */
function column(ids: readonly string[], model: CanvasModel, size?: (id: string) => Sized): Sized {
  const children: Placed[] = [];
  let y = 0;
  let width = 0;
  for (const id of ids) {
    const d = model.items.get(id)?.kind === "group" && size ? dims(size(id)) : leafSize(id, model);
    children.push({ id, x: 0, y, ...d });
    y += d.height + GAP;
    width = Math.max(width, d.width);
  }
  return { width, height: Math.max(0, y - GAP), children };
}

/** Leaves in a fixed number of columns, row by row. */
function grid(ids: readonly string[], model: CanvasModel, columns: number): Sized {
  const children: Placed[] = [];
  let rowHeight = 0;
  let y = 0;
  let width = 0;
  ids.forEach((id, i) => {
    const col = i % columns;
    if (col === 0 && i > 0) {
      y += rowHeight + GAP;
      rowHeight = 0;
    }
    const d = leafSize(id, model);
    const x = col * (CARD.width + GAP);
    children.push({ id, x, y, ...d });
    rowHeight = Math.max(rowHeight, d.height);
    width = Math.max(width, x + d.width);
  });
  return { width, height: ids.length === 0 ? 0 : y + rowHeight, children };
}

/** Elements side by side, aligned to the top. */
function row(items: readonly (Placed | { id: string; width: number; height: number })[], gap: number): Sized {
  const children: Placed[] = [];
  let x = 0;
  let height = 0;
  for (const item of items) {
    children.push({ id: item.id, x, y: 0, width: item.width, height: item.height });
    x += item.width + gap;
    height = Math.max(height, item.height);
  }
  return { width: Math.max(0, x - gap), height, children };
}

/** Sections top to bottom, skipping empty ones. */
function stack(sections: readonly Sized[]): Sized {
  const children: Placed[] = [];
  let y = 0;
  let width = 0;
  for (const section of sections) {
    if (section.children.length === 0) continue;
    for (const c of section.children) children.push({ ...c, y: c.y + y });
    y += section.height + SECTION_GAP;
    width = Math.max(width, section.width);
  }
  return { width, height: Math.max(0, y - SECTION_GAP), children };
}

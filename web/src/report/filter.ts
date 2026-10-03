import type { Resource } from "./schema.gen";

export type Action = Resource["change"]["action"];
export type Family = Resource["family"];

/** "changes" hides unchanged resources unless a search is active. */
export type ChangeFilter = "changes" | "all" | Action;

export interface ResourceFilter {
  query: string;
  change: ChangeFilter;
  family: Family | "all";
}

export const DEFAULT_FILTER: ResourceFilter = { query: "", change: "changes", family: "all" };

/** Applies the navigator's search, change and domain filters. */
export function filterResources(resources: readonly Resource[], filter: ResourceFilter): Resource[] {
  const query = filter.query.trim().toLowerCase();
  return resources.filter((r) => {
    if (query && ![r.label, r.address, r.type].some((s) => s.toLowerCase().includes(query))) return false;
    if (filter.family !== "all" && r.family !== filter.family) return false;
    switch (filter.change) {
      case "all":
        return true;
      case "changes":
        // Searching should find a resource whatever its change.
        return query !== "" || r.change.action !== "no_op";
      default:
        return r.change.action === filter.change;
    }
  });
}

/** Families present in the report, for the domain filter. */
export function familiesIn(resources: readonly Resource[]): Family[] {
  return [...new Set(resources.map((r) => r.family))].sort();
}

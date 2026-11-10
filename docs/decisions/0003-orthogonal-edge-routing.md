# ADR 0003: Obstacle-aware orthogonal edge routing

- **Status:** Accepted
- **Date:** 6 November 2026
- **Amends:** [ADR 0001](0001-structural-architecture-layout.md), which drew edges as smooth curves with no routing engine

## Context

ADR 0001 drew each relationship as a curve between the facing sides of two cards and accepted that curves could pass beneath other cards. With real stacks, long curves from workloads to regional services ran under whole rows of cards. It was hard to tell where a line went, and the plan's visual checks (§13) ask for deliberate edge routing.

## Decision

Edges are routed orthogonally around cards by a small in-house router in `web/src/graph/` (`edgeRouter.ts`, `routeGrid.ts` and `routeSearch.ts`). It adds no dependency.

- Obstacles are leaf cards: resources, proxy markers and collapsed groups. Open groups are not obstacles, because lines must cross into them.
- The router builds a sparse grid of lanes 6 px outside every obstacle. That is half the layout's card gap, so lanes run down the middle of gutters.
- A* search over that grid prices each bend as 40 px of line. Routes leave and enter at the midpoint of a card side, square-on to it, and are drawn with 8 px rounded corners.
- The search first looks within 160 px of the two endpoints, then across the whole canvas. If no clear route exists, the edge falls back to the ADR 0001 curve, so a relationship is never silently dropped.
- Ties are broken by insertion order, so the same layout always gives the same paths.

## Alternatives considered

- **Keep curves and rely on selection dimming.** Rejected: this was the reported problem.
- **ELK or libavoid (WASM).** These are licence and size concerns for every embedded report, as ADR 0001 found for ELK. They also bring layout opinions we do not need.
- **Straight elbow connectors without obstacle avoidance.** Tidier than curves, but they still cross cards.

## Consequences

- Lines no longer pass under cards. Unit tests check this for every edge in the sample report and route a fully expanded 500-resource stack.
- Layout plus routing takes about 1.5 ms for the 22-edge sample. A fully expanded 500-resource stack with 1,475 edges takes about 200 ms in Node. The 500-resource report opens collapsed, and its layout stays at about 1 ms in the end-to-end budget test.
- Edges that share a lane overlap rather than running side by side. Selecting a resource still traces its own relationships. Separating parallel lines (nudging) is a possible follow-up.
- Edges to an open group, such as a subnet, attach to the group's border.
- The website's screenshots predate this change until they are regenerated. The website now lives in a separate repository.

## Migration and rollback

There is no persisted data or contract change: paths are computed in the browser. Rolling back means pointing `layout.ts` at `curvePath` again and removing the three router files.

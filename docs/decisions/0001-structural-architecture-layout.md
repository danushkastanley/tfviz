# ADR 0001: Deterministic structural layout for the architecture view

- **Status:** Accepted for M0, to be revisited at the M0 design review
- **Date:** 3 October 2026

## Context

The implementation plan (§6) says to "start with React Flow for interaction and evaluate ELK.js for layout". M0 built the canvas against the synthetic `aws-review` stack using ELK's layered algorithm with `hierarchyHandling: INCLUDE_CHILDREN`. That stack has:

- account, region and VPC containment
- six subnets
- workloads that span several subnets
- regional services

What we found:

- **Readability.** Left-to-right layering stacked all six subnets in a single tall column. Top-down layering with partitioned bands produced a 4,100 × 884 px strip. Neither was legible at fit, and neither resembles how engineers draw AWS topology.
- **Licence.** `elkjs` is EPL-2.0 OR GPL-3.0. Every generated report would embed it, so each report would need EPL source-availability notices.
- **Size.** The bundled build is 1.6 MB, more than three times the rest of the interface. It took 96 ms to lay out 42 nodes.

## Decision

The architecture view uses a deterministic structural layout (`web/src/graph/architectureLayout.ts`) instead of a generic graph layout:

- Account, region and VPC nesting follow the report's groups.
- Inside a VPC, subnets form a grid: availability zones are columns, and public, private and unclassified tiers are rows. Network-edge resources sit above the grid, with workloads and then security groups below.
- Regional services sit beside the VPC. Unplaced resources are kept apart.
- A resource in several subnets appears once, with proxy markers in the other subnets.
- Relationship edges are smooth curves between the facing sides of the two cards. React Flow renders them; there is no routing engine.

## Alternatives considered

- **ELK layered with layout tuning.** Rejected for the readability, licence and size reasons above. Tuning cannot make it use AZ and tier semantics it does not know about.
- **ELK per container with mixed algorithms.** Cross-hierarchy edges are not routed in separate-children mode, which removes the main benefit.
- **dagre (MIT).** It has no compound-graph containment of the kind we need. It may still suit the module view, which has no geographic semantics.

## Consequences

- Layout is fast and deterministic: 0.3 ms for the sample stack and 5–11 ms for 500 resources across Chromium, Firefox and WebKit. The same input always gives the same geometry, which keeps resource positions stable between views.
- Reports are 510 KB for the sample and 984 KB for 500 resources, with no copyleft code. Bundled licences are MIT, ISC and BSD-3-Clause, and the notices are embedded in every report.
- Edges are not obstacle-aware and can cross cards in dense stacks. Selection highlighting and dimming mitigate this. Edge bundling or routing can be added later if alpha feedback asks for it.
- New placement kinds, such as EKS node groups or transit networking, need explicit layout rules. That is deliberate, but it is ongoing work.
- The module view (M2) still needs a layout. A small layered layout for that view alone remains an open option.

## Migration and rollback

There is no persisted data or public contract involved: layout is computed in the browser from the report model. Rolling back means restoring the ELK adapter from PR #3's history and re-adding `elkjs`. In that case the report would also need EPL-2.0 notices.

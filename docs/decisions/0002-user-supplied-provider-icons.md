# ADR 0002: Official provider icons from a user-supplied pack

- **Status:** Accepted
- **Date:** 3 October 2026

## Context

Plan §13 calls for "small, consistent monochrome symbols". tfviz draws its own, one per service family. Reviewers recognise official service icons faster, and the maintainer asked for them.

The official sets allow use in diagrams but not redistribution in software. We checked AWS, Azure, Google Cloud, Oracle and IBM:

- **AWS:** allows customers and partners to use the icons "to create architecture diagrams".
- **Azure:** restricts use to the permitted purpose.
- **Google Cloud and Oracle:** say nothing about redistribution.

None offers a licence compatible with bundling the icons in an Apache-2.0 binary that others download.

## Decision

- tfviz ships no provider icons.
- `--icons <folder>` reads the official pack the user downloaded. tfviz maps resource types to the pack's file names and embeds the matching SVGs in that report as `data:` images.
- The AWS mapping lives in `internal/provider/aws/icons.go`. It prefers resource icons, then service icons, and was checked against the July 2026 pack. The loader in `internal/icons` is provider-neutral.
- Without `--icons`, or for types with no official icon, the interface keeps tfviz's own symbols.

## Alternatives

- **Bundle the icons.** This is the common open-source practice, but it relies on tolerance, not a licence.
- **Ask each provider for written permission.** This remains open and would allow bundling later without changing the report format.
- **Draw lookalike icons.** This risks trade-dress confusion, and the results are less recognisable than the real ones.

## Consequences

- Users take one extra step to download the pack once. The report remains a single offline file.
- The schema gains an optional `icons` object, holding data URIs and a type-to-id map. It is additive, so older reports are unaffected.
- Reports with icons are about 65–95 KB larger for the fixture stacks, which have 20–28 icon types.
- New pack releases can rename files. Types that no longer match fall back to symbols, and the mapping is updated in code.

## Rollback

Remove the flag, `internal/icons` and the schema field. The interface falls back to symbols when `icons` is absent.

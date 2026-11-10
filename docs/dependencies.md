# Released dependency assessment

Checked on 2026-11-10 for Go 1.27.3. Final compiler/graph/target and advisory results are retained with validation evidence.

| Module | Selected release | Responsibility and license |
| --- | --- | --- |
| github.com/marcuoli/go-atomicfile | v0.1.1 | First-party atomic publication |
| github.com/mmonterroca/docxgo/v2 | v2.16.0 | General Word model, images, metadata, sections, serialization and supported text replacement; declared MIT |
| github.com/beevik/etree | v1.8.1 | Namespace-preserving XML tree operations; BSD 2-Clause |
| golang.org/x/net | v0.62.0 | HTML parser; BSD 3-Clause |

The selected document library's current LICENSE declares MIT and its upstream provenance audit explains historical AGPL snapshots and the rewritten releases. This records the upstream determination rather than an independent legal assessment. No historical AGPL-only snapshot or local dependency fork is used.

Official sources: [docxgo release source](https://github.com/mmonterroca/docxgo/tree/v2.16.0), [license and provenance](https://github.com/mmonterroca/docxgo/blob/v2.16.0/docs/PROVENANCE_AUDIT.md), [etree](https://github.com/beevik/etree/tree/v1.8.1), [Go HTML parser](https://pkg.go.dev/golang.org/x/net/html).

The library's general MergeTemplate operation can clear an adjacent empty field-bearing run. The supported literal ReplaceText operation is used with skipped-layout receipts; unsupported field-adjacent templates return an explicit error. Its retained-template style part also requires new helper-owned definitions to be serialized separately through the official writer and appended to the artifact. Neither adaptation modifies dependency source or claims those upstream limitations are fixed.

Other candidates were assessed before implementation. The narrower high-level table wrappers did not cover the report controls; a licensed runtime was not selected. The broader requirements now justify the opt-in general engine while the small report path keeps its original import surface. Review newly released compatible fixes before the next dependency/build qualification.

All dependencies are consumed as official releases with signed module sums. No local replacement, fork, vendored modification, cache patch or build-time rewriting is part of final validation.

# Released dependency assessment

## Current build selection

Refreshed on 2026-11-15 against the official Go feed/history and upstream stable
tags. The current module and build requirement is Go 1.27.4; validation scripts
and CI use that version. The standalone vulnerability scanner is v1.9.0.

The direct library releases below remain current: etree v1.8.1, go-ole v1.3.0,
go-atomicfile v0.1.1, docxgo/v2 v2.16.0 and x/net v0.62.0. Upstream tag review
found no newer stable major successor for these module paths. Existing licenses
and the document-engine provenance qualification below remain applicable.

The complete selected upstream graph now uses x/crypto v0.59.0, x/sys v0.50.0,
x/term v0.48.0, x/text v0.44.0 and x/mod v0.43.0; x/sync v0.25.0 and x/tools
v0.52.0 enter through their upstream graph. Explicit indirect constraints keep
otherwise unimported paths current. An ordinary `go mod tidy` removes those
constraints and restores older versions from x/net's released go.mod; the
refresh therefore runs tidy first, then deliberately restores current official
graph constraints through Go module tooling. These pins add no compiled imports
and do not modify any dependency implementation or manifest.

Go 1.27.4 full tests, race tests, vet/stdversion, module verification, example
builds and v1.9.0 source/binary scans passed on Windows amd64. The synthetic
Windows integration-tag Word TOC test also passed. Current scans reported no
vulnerabilities, including no remaining required-module GO-2026-6859 result.
This describes the tested configuration, not universal security or reader
qualification. Dated historical results below retain their original meaning.

## Historical qualification

Checked on 2026-11-10 for Go 1.27.3. Final compiler/graph/target and advisory results are retained with validation evidence.

| Module | Selected release | Responsibility and license |
| --- | --- | --- |
| github.com/marcuoli/go-atomicfile | v0.1.1 | First-party atomic publication |
| github.com/mmonterroca/docxgo/v2 | v2.16.0 | General Word model, images, metadata, sections, serialization and supported text replacement; declared MIT |
| github.com/go-ole/go-ole | v1.3.0 | Optional Windows COM automation through the official upstream release; MIT |
| github.com/beevik/etree | v1.8.1 | Namespace-preserving XML tree operations; BSD 2-Clause |
| golang.org/x/net | v0.62.0 | HTML parser; BSD 3-Clause |

The selected document library's current LICENSE declares MIT and its upstream provenance audit explains historical AGPL snapshots and the rewritten releases. This records the upstream determination rather than an independent legal assessment. No historical AGPL-only snapshot or local dependency fork is used.

Official sources: [docxgo release source](https://github.com/mmonterroca/docxgo/tree/v2.16.0), [license and provenance](https://github.com/mmonterroca/docxgo/blob/v2.16.0/docs/PROVENANCE_AUDIT.md), [etree](https://github.com/beevik/etree/tree/v1.8.1), [Go HTML parser](https://pkg.go.dev/golang.org/x/net/html).

The library's general MergeTemplate operation can clear an adjacent empty field-bearing run. The supported literal ReplaceText operation is used with skipped-layout receipts; unsupported field-adjacent templates return an explicit error. Its retained-template style part also requires new helper-owned definitions to be serialized separately through the official writer and appended to the artifact. Neither adaptation modifies dependency source or claims those upstream limitations are fixed.

Other candidates were assessed before implementation. The narrower high-level table wrappers did not cover the report controls; a licensed runtime was not selected. The broader requirements now justify the opt-in general engine while the small report path keeps its original import surface. Review newly released compatible fixes before the next dependency/build qualification.

All dependencies are consumed as official releases with signed module sums. No local replacement, fork, vendored modification, cache patch or build-time rewriting is part of final validation.

Qualification on 2026-11-11 retains the required Go 1.27.3 toolchain. govulncheck reports GO-2026-6859 (net/http MaxBytesReader header race), fixed in Go 1.27.4, at required-module level; no affected imported package or called symbol was reported for this module or either example binary. This is a bounded scanner result, not proof of exploitability or universal absence of vulnerabilities. No compiler migration is included in the Word compatibility patch. Recheck this advisory at the next configured build review.

The optional Windows-only Word package uses the official go-ole v1.3.0 release without CGO. Upstream source and license: https://github.com/go-ole/go-ole/tree/v1.3.0 . The default portable renderer does not invoke Word. This dependency is not a replacement document engine.
